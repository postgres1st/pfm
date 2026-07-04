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

# ---------------------------------------------------------------------------
# DECISION POINT (yours to implement) — see request in the chat.
#
# pfm_srv_initialized: return 0 (true) if /srv is already provisioned and the
# heavy one-time steps (dir tree, PG initdb, pg_stat_statements) should be
# SKIPPED; return 1 (false) to run them.
#
# The container uses a single top-level sentinel: the mere existence of
# ${DIST_FILE}. That is simple but blind to a half-finished init (e.g. power
# loss between initdb and marking done -> sentinel present, PG data dir
# corrupt/absent). The alternative is per-artifact guards (check the data dir,
# the password file, etc.) which recover from partial failure but are wordier.
#
# Trade-off to weigh: crash-recovery robustness vs simplicity, and the blast
# radius of a wrong answer (re-running initdb over a live data dir must never
# happen — a false "needs init" is far more dangerous than a false "done").
# ---------------------------------------------------------------------------
pfm_srv_initialized() {
    # Key the decision off the one artifact we must never overwrite: the PG
    # data dir's PG_VERSION file, which initdb writes last-ish and which marks
    # a real cluster. Non-empty PG_VERSION => cluster exists => skip init.
    # This directly protects the dangerous op (a separate sentinel can desync
    # from the data dir; PG_VERSION cannot).
    [[ -s "${POSTGRES_DATA_DIR}/PG_VERSION" ]]
}

init_srv() {
    log "initializing ${SRV} ..."
    mkdir -p "${SRV}"/{backup,clickhouse,grafana/plugins,logs,nginx,prometheus/rules,victoriametrics}
    log "copying grafana plugins ..."
    cp -r /usr/share/percona-dashboards/panels/* "${SRV}/grafana/plugins/"

    log "initializing PostgreSQL ..."
    install -d -m 750 "${POSTGRES_DATA_DIR}"
    local pgpw
    pgpw=$(openssl rand -hex 16)
    printf '%s' "${pgpw}" > "${POSTGRES_PASSWORD_FILE}"
    chmod 600 "${POSTGRES_PASSWORD_FILE}"
    "${PG_BIN}/initdb" -D "${POSTGRES_DATA_DIR}" \
        --auth-host=scram-sha-256 --auth-local=trust \
        --username=postgres --pwfile="${POSTGRES_PASSWORD_FILE}"

    # Bring PostgreSQL up briefly (as the pg_ctl-owned instance, before the real
    # pfm-postgresql.service) to do the one-time DB provisioning. pmm-managed
    # self-creates its own DB/role on startup (initWithRoot, via
    # /srv/.postgres_password), so only Grafana's DB is created here.
    log "starting PostgreSQL for first-boot provisioning ..."
    "${PG_BIN}/pg_ctl" start -D "${POSTGRES_DATA_DIR}" -o "-c logging_collector=off"

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

    unset PGPASSWORD
    "${PG_BIN}/pg_ctl" stop -D "${POSTGRES_DATA_DIR}"

    # Dashboards version marker (plugins are copied above); mirrors the ansible
    # dashboards role so upgrade detection has a baseline.
    if [ -f /usr/share/percona-dashboards/VERSION ]; then
        cp /usr/share/percona-dashboards/VERSION "${SRV}/grafana/PERCONA_DASHBOARDS_VERSION"
    fi

    printf '%s' "native" > "${DIST_FILE}"
    log "/srv initialization complete."
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
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -extensions v3_req \
            -keyout "${ssl_dst}/certificate.key" \
            -out "${ssl_dst}/certificate.crt" \
            -config "${ssl_dst}/certificate.conf"
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

    if pfm_srv_initialized; then
        log "/srv already initialized — skipping one-time provisioning."
    else
        init_srv
    fi
    prepare_runtime
    log "done."
}

main "$@"
