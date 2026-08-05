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
readonly POSTGRES_DATA_DIR="${SRV}/postgres18"
readonly POSTGRES_PASSWORD_FILE="${SRV}/.postgres_password"
readonly PG_BIN=/usr/pgsql-18/bin

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
    # No grafana/plugins here on purpose. This used to copy
    # /opt/postgres1st/pfmm/dashboards/panels into ${SRV}/grafana/plugins, and
    # because provisioning only runs once (guarded by srv_provisioned), that copy
    # was never refreshed: upgrading percona-dashboards or pfm-managed updated the
    # packaged plugins while the running server kept serving the first-boot copy,
    # so dashboard changes could never reach an existing install. grafana.ini now
    # points paths.plugins straight at the package directory instead.
    # grafana (without /plugins) is still needed: it is grafana's paths.data.
    mkdir -p "${SRV}"/{backup,clickhouse,grafana,logs,nginx,prometheus/rules,victoriametrics}
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

# provision_app_db <dbname> <role> <password> — idempotently create an
# application database and its login role, then make the role OWN the database.
#
# Ownership, not just GRANT: PostgreSQL 15 revoked CREATE on schema public from
# PUBLIC, and schema public is owned by the pseudo-role pg_database_owner. So a
# database left owned by postgres (the superuser that created it) leaves the app
# role unable to create its own tables — both grafana's and pfm-managed's
# migrators fail on their first CREATE TABLE with "permission denied for schema
# public" (SQLSTATE 42501). Handing the database to the app role makes
# pg_database_owner resolve to it, restoring CREATE on public.
# Requires PGPASSWORD to be exported by the caller.
provision_app_db() {
    local db=$1 role=$2 pw=$3
    # The names below are interpolated into SQL. Every current caller passes a
    # literal, so nothing here is attacker-controlled today -- but this refuses
    # anything that is not a plain identifier so that stays true if a caller ever
    # starts deriving them from config or the environment. Quoting alone would not
    # be enough: an embedded double quote closes the quoted identifier, and an
    # embedded single quote closes the password literal.
    local n
    for n in "${db}" "${role}"; do
        [[ ${n} =~ ^[A-Za-z_][A-Za-z0-9_-]*$ ]] || {
            log "FATAL: refusing unsafe SQL identifier: ${n}" >&2; exit 1; }
    done
    [[ ${pw} =~ ^[A-Za-z0-9_-]+$ ]] || {
        log "FATAL: refusing password with SQL-significant characters" >&2; exit 1; }
    local pg=(/usr/bin/psql -U postgres -h /run/postgresql -d postgres)
    if [ "$("${pg[@]}" -tAc "SELECT 1 FROM pg_database WHERE datname='${db}'")" != "1" ]; then
        "${pg[@]}" -c "CREATE DATABASE \"${db}\""
    fi
    if [ "$("${pg[@]}" -tAc "SELECT 1 FROM pg_roles WHERE rolname='${role}'")" != "1" ]; then
        "${pg[@]}" -c "CREATE USER \"${role}\" LOGIN PASSWORD '${pw}'"
    fi
    "${pg[@]}" -c "GRANT ALL PRIVILEGES ON DATABASE \"${db}\" TO \"${role}\""
    "${pg[@]}" -c "ALTER DATABASE \"${db}\" OWNER TO \"${role}\""
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
    provision_app_db grafana grafana grafana

    # pfm-managed creates its own DB/role on first start, but does so as the
    # postgres superuser, which leaves the database owned by postgres — fatal on
    # PG15+ (see provision_app_db). Pre-create them here with the right owner;
    # pfm-managed skips creation when they already exist.
    log "creating pfm-managed database and user ..."
    provision_app_db pmm-managed pmm-managed pmm-managed
}

# provision_srv runs the one-time /srv provisioning, guarded so it recovers from
# a partial previous run (see the two-phase guard note above).
provision_srv() {
    if srv_provisioned; then
        # A host provisioned against an older PostgreSQL major has the sentinel but
        # no cluster at the current POSTGRES_DATA_DIR (the path carries the major:
        # /srv/postgres14 -> /srv/postgres18). Returning here would skip initdb and
        # leave pfm-postgresql to fail on an empty data directory, which reads like
        # a broken package rather than a migration that was never performed. There
        # is deliberately no automatic migration: crossing a PostgreSQL major needs
        # pg_upgrade or dump/restore, and silently initdb'ing over a host that still
        # holds the old cluster would strand its data.
        if ! pg_cluster_exists; then
            local old
            for old in "${SRV}"/postgres[0-9]*; do
                [ -d "${old}" ] && [ "${old}" != "${POSTGRES_DATA_DIR}" ] || continue
                log "FATAL: ${SRV} was provisioned with $(basename "${old}") but this build expects" >&2
                log "       $(basename "${POSTGRES_DATA_DIR}"). Migrate the cluster (pg_upgrade or" >&2
                log "       dump/restore) into ${POSTGRES_DATA_DIR}, or start from an empty ${SRV}." >&2
                exit 1
            done
        fi
        log "/srv already provisioned — skipping one-time setup."
        return
    fi
    log "provisioning ${SRV} ..."
    ensure_srv_dirs
    pg_cluster_exists || init_postgres
    provision_databases

    # Dashboards version marker (plugins copied above); mirrors the ansible
    # dashboards role so upgrade detection has a baseline.
    if [ -f /opt/postgres1st/pfmm/dashboards/VERSION ]; then
        cp /opt/postgres1st/pfmm/dashboards/VERSION "${SRV}/grafana/PERCONA_DASHBOARDS_VERSION"
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

    # Validate the config pfm-nginx.service actually runs (/etc/nginx/pfm.conf via
    # `nginx -c`), NOT the base nginx package's default /etc/nginx/nginx.conf — and
    # route the startup error log to stderr so the check does not fail trying to open
    # the root-only /var/log/nginx/error.log as the pfm user.
    log "validating nginx configuration ..."
    nginx -t -c /etc/nginx/pfm.conf -e stderr

    # Create the pmm-agent config (idempotent) so pfm-agent.service can start.
    # Native keeps it under /srv (pfm-writable, persistent) rather than the
    # container's root-owned /opt/postgres1st/pfm/config; pfm-agent.service's
    # --config-file points here to match.
    install -d -m 770 /srv/pfm-agent/config /srv/pfm-agent/tmp /srv/nomad/data
    local agent_cfg=/srv/pfm-agent/config/pfm-agent.yaml
    if [ ! -f "${agent_cfg}" ]; then
        log "creating pfm-agent configuration ..."
        # node-address is only autodetected when the host has a routable
        # (non-loopback) address; on an isolated/air-gapped host it resolves to
        # empty and `setup` then hard-fails on the required positional arg. This
        # agent only ever talks to the co-located server over loopback, so fall
        # back to 127.0.0.1 rather than depending on autodetection.
        # Use the first global-scope IPv4 rather than `route get`, which exits 2
        # ("Network is unreachable") on a host with no default route and would
        # trip errexit. Must not be fatal: fall back to loopback.
        local node_addr=""
        node_addr="$(ip -4 -o addr show scope global 2>/dev/null |
                     awk '{split($4,a,"/"); print a[1]; exit}')" || node_addr=""
        [ -n "${node_addr}" ] || node_addr=127.0.0.1
        /usr/sbin/pfm-agent setup \
            --config-file="${agent_cfg}" \
            --skip-registration \
            --id=pmm-server \
            --paths-tempdir=/srv/pfm-agent/tmp \
            --paths-nomad-data-dir=/srv/nomad/data \
            --server-address=127.0.0.1:8443 \
            --server-insecure-tls \
            "${node_addr}"
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
