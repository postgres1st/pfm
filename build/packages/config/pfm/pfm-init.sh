#!/bin/bash
# pfm-init.sh — first-boot provisioning for the native systemd stack.
# Installed to /usr/share/pfm/pfm-init.sh, run by pfm-init.service (oneshot).
#
# This is the native-systemd port of the first-boot block in
# build/docker/server/entrypoint.sh. Key simplification vs the container:
# systemd runs this as the real `pfm` user, so ALL of the entrypoint's
# NSS-wrapper / arbitrary-UID handling is dropped — there is a real passwd
# entry and /srv is owned by pfm (set by the RPM %post + tmpfiles.d).
#
# Contract: must be safe to run on EVERY boot (RemainAfterExit keeps it from
# re-running within a boot, but it fires again after each reboot).
set -o errexit
set -o nounset
set -o pipefail

readonly SRV=/srv
readonly DIST_FILE="${SRV}/pmm-distribution"
readonly POSTGRES_DATA_DIR="${SRV}/postgres14"
readonly POSTGRES_PASSWORD_FILE="${SRV}/.postgres_password"
readonly PG_BIN=/usr/pgsql-14/bin

log() { echo "pfm-init: $*"; }

# Two-phase guard. PG_VERSION (written by initdb) protects the DANGEROUS,
# never-repeat op — we must never re-initdb over a real cluster. DIST_FILE is the
# "provisioning complete" sentinel, written LAST, and gates the RECOVERABLE,
# idempotent post-init steps (grafana DB, extension, markers). A crash between
# initdb and DIST_FILE is thus repaired on the next boot rather than skipped
# forever (the earlier single PG_VERSION guard extended "done" to these
# recoverable steps and left Grafana's DB uncreated permanently).
pg_cluster_exists() { [[ -s "${POSTGRES_DATA_DIR}/PG_VERSION" ]]; }
srv_provisioned()   { [[ -f "${DIST_FILE}" ]]; }

ensure_srv_dirs() {
    mkdir -p "${SRV}"/{backup,clickhouse,grafana/plugins,logs,nginx,prometheus/rules,victoriametrics}
    log "copying grafana plugins ..."
    # Copy the directory contents (trailing /.) rather than a glob: an empty
    # panels dir leaves `*` unexpanded, and under `set -e` the resulting cp error
    # aborts the entire first-boot provision.
    cp -r /usr/share/percona-dashboards/panels/. "${SRV}/grafana/plugins/"
}

init_postgres() {
    log "initializing PostgreSQL ..."
    install -d -m 750 "${POSTGRES_DATA_DIR}"
    local pgpw
    pgpw=$(openssl rand -hex 16)
    printf '%s' "${pgpw}" > "${POSTGRES_PASSWORD_FILE}"
    chmod 600 "${POSTGRES_PASSWORD_FILE}"
    # scram for local too, not trust: on a native multi-user host `trust` lets any
    # local OS user authenticate as any PG role (incl. superuser) over the socket.
    # First-boot provisioning is the only local-socket client and it authenticates
    # with PGPASSWORD (see provision_databases), so scram keeps it working.
    "${PG_BIN}/initdb" -D "${POSTGRES_DATA_DIR}" \
        --auth-host=scram-sha-256 --auth-local=scram-sha-256 \
        --username=postgres --pwfile="${POSTGRES_PASSWORD_FILE}"
}

# provision_databases brings PostgreSQL up briefly (the pg_ctl-owned instance,
# before pfm-postgresql.service) for the idempotent one-time DB setup. A RETURN
# trap stops PG even if a step fails under errexit, so a mid-window failure never
# leaves a stray postmaster for systemd to SIGKILL (unclean shutdown).
# pmm-managed self-creates its own DB/role on startup, so only Grafana's is here.
provision_databases() {
    local pgpw
    pgpw=$(cat "${POSTGRES_PASSWORD_FILE}")
    log "starting PostgreSQL for first-boot provisioning ..."
    "${PG_BIN}/pg_ctl" start -D "${POSTGRES_DATA_DIR}" -o "-c logging_collector=off"
    trap 'unset PGPASSWORD; "${PG_BIN}/pg_ctl" stop -D "${POSTGRES_DATA_DIR}" -m fast >/dev/null 2>&1 || true' RETURN

    export PGPASSWORD="${pgpw}"
    local psql=(/usr/bin/psql -U postgres -h /run/postgresql -d postgres)

    log "enabling pg_stat_statements ..."
    "${psql[@]}" -c 'CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public'

    log "creating grafana database and user ..."
    if [ "$("${psql[@]}" -tAc "SELECT 1 FROM pg_database WHERE datname='grafana'")" != "1" ]; then
        "${psql[@]}" -c 'CREATE DATABASE grafana'
    fi
    if [ "$("${psql[@]}" -tAc "SELECT 1 FROM pg_roles WHERE rolname='grafana'")" != "1" ]; then
        "${psql[@]}" -c "CREATE USER grafana LOGIN PASSWORD 'grafana'"
    fi
    "${psql[@]}" -c 'GRANT ALL PRIVILEGES ON DATABASE grafana TO grafana'
}

# provision_srv runs the one-time /srv provisioning, guarded so it recovers from
# a partial previous run (see the two-phase guard note above).
provision_srv() {
    if srv_provisioned; then
        log "/srv already provisioned — skipping one-time setup."
        return
    fi
    log "provisioning ${SRV} ..."
    ensure_srv_dirs
    pg_cluster_exists || init_postgres
    provision_databases

    # Dashboards version marker (plugins copied above); mirrors the ansible
    # dashboards role so upgrade detection has a baseline.
    if [ -f /usr/share/percona-dashboards/VERSION ]; then
        cp /usr/share/percona-dashboards/VERSION "${SRV}/grafana/PERCONA_DASHBOARDS_VERSION"
    fi

    printf '%s' "native" > "${DIST_FILE}"   # written LAST — the "done" sentinel
    log "/srv provisioning complete."
}

# generate_nginx_cert provisions the TLS material nginx.conf references
# (certificate.crt/key, ca-certs.pem, dhparam.pem under /srv/nginx). This is the
# native port of the container's generate-ssl-certificate: same logic, minus the
# cloud-init /var/lib/cloud/scripts location. The supporting files ship in the
# RPM at /etc/nginx/ssl; the leaf cert is self-signed here (localhost SAN via
# certificate.conf). Operators can bring their own by dropping certificate.crt/
# key into /srv/nginx — present files are never overwritten.
generate_nginx_cert() {
    local ssl_src=/etc/nginx/ssl
    local ssl_dst="${SRV}/nginx"
    mkdir -p "${ssl_dst}"

    local f
    for f in dhparam.pem ca-certs.pem certificate.conf; do
        if [ ! -f "${ssl_dst}/${f}" ]; then
            if [ ! -f "${ssl_src}/${f}" ]; then
                log "FATAL: ${ssl_src}/${f} missing; the pfm-server RPM must ship the nginx SSL sources." >&2
                exit 1
            fi
            cp "${ssl_src}/${f}" "${ssl_dst}/${f}"
        fi
    done

    if [ ! -f "${ssl_dst}/certificate.key" ] || [ ! -f "${ssl_dst}/certificate.crt" ]; then
        log "generating self-signed nginx certificate ..."
        # umask in a subshell so the private key is created 0600 from the first
        # syscall (no world-readable window before the chmod).
        (
            umask 077
            openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
                -extensions v3_req \
                -keyout "${ssl_dst}/certificate.key" \
                -out "${ssl_dst}/certificate.crt" \
                -config "${ssl_dst}/certificate.conf"
        )
        chmod 600 "${ssl_dst}/certificate.key"
    fi
}

# Steps that are cheap and must run every boot regardless of init state.
prepare_runtime() {
    log "creating nginx temp directories ..."
    mkdir -p "${SRV}"/nginx/tmp/{client,proxy,fastcgi,uwsgi,scgi}

    generate_nginx_cert

    log "validating nginx configuration ..."
    nginx -t

    # Create the pmm-agent config (idempotent) so pfm-agent.service can start.
    # Native keeps it under /srv (pfm-writable, persistent) rather than the
    # container's root-owned /usr/local/percona/pmm/config; pfm-agent.service's
    # --config-file points here to match.
    install -d -m 770 /srv/pmm-agent/config /srv/pmm-agent/tmp /srv/nomad/data
    local agent_cfg=/srv/pmm-agent/config/pmm-agent.yaml
    if [ ! -f "${agent_cfg}" ]; then
        log "creating pmm-agent configuration ..."
        /usr/sbin/pmm-agent setup \
            --config-file="${agent_cfg}" \
            --skip-registration \
            --id=pmm-server \
            --paths-tempdir=/srv/pmm-agent/tmp \
            --paths-nomad-data-dir=/srv/nomad/data \
            --server-address=127.0.0.1:8443 \
            --server-insecure-tls
    fi
}

main() {
    # /srv must be owned/writable by the pfm user (the RPM %post chowns it, T7).
    # Fail loud with guidance rather than dying on the first mkdir's EACCES.
    if [ ! -w "${SRV}" ]; then
        log "FATAL: ${SRV} is not writable by the pfm user (uid $(id -u)/gid $(id -g))." >&2
        log "The RPM %post must run 'chown -R pfm:pfm ${SRV}' before pfm-init starts." >&2
        exit 1
    fi

    provision_srv
    prepare_runtime
    log "done."
}

main "$@"
