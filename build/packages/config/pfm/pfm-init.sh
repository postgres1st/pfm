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

    log "enabling pg_stat_statements ..."
    "${PG_BIN}/pg_ctl" start -D "${POSTGRES_DATA_DIR}" -o "-c logging_collector=off"
    PGPASSWORD="${pgpw}" /usr/bin/psql -U postgres -h /run/postgresql -d postgres \
        -c 'CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public'
    "${PG_BIN}/pg_ctl" stop -D "${POSTGRES_DATA_DIR}"

    printf '%s' "native" > "${DIST_FILE}"
    log "/srv initialization complete."
}

# Steps that are cheap and must run every boot regardless of init state.
prepare_runtime() {
    log "creating nginx temp directories ..."
    mkdir -p "${SRV}"/nginx/tmp/{client,proxy,fastcgi,uwsgi,scgi}

    log "generating self-signed nginx certificate ..."
    bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate >/dev/null 2>&1

    log "validating nginx configuration ..."
    nginx -t
}

main() {
    if pfm_srv_initialized; then
        log "/srv already initialized — skipping one-time provisioning."
    else
        init_srv
    fi
    prepare_runtime
    log "done."
}

main "$@"
