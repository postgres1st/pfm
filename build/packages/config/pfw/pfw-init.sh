#!/bin/bash
# pfw-init.sh — first-boot provisioning for the native systemd stack.
# Installed to /usr/share/pfw/pfw-init.sh, run by pfw-init.service (oneshot).
#
# This is the native-systemd port of the first-boot block in
# build/docker/server/entrypoint.sh. Key simplification vs the container:
# systemd runs this as the real `pfw` user, so ALL of the entrypoint's
# NSS-wrapper / arbitrary-UID handling is dropped — there is a real passwd
# entry and /srv is owned by pfw (set by the RPM %post + tmpfiles.d).
#
# Contract: must be safe to run on EVERY boot (RemainAfterExit keeps it from
# re-running within a boot, but it fires again after each reboot).
set -o errexit
set -o nounset
set -o pipefail

readonly SRV=/srv
readonly DIST_FILE="${SRV}/pfw-distribution"
# Where the sentinel lived before the rebrand. srv_provisioned() accepts EITHER, because
# this file is what stops provision_srv() re-running initdb over a real cluster: rename it
# without the fallback and an existing host reads as unprovisioned on its next boot.
readonly LEGACY_DIST_FILE="${SRV}/pmm-distribution"
readonly POSTGRES_DATA_DIR="${SRV}/postgres18"
readonly POSTGRES_PASSWORD_FILE="${SRV}/.postgres_password"
readonly PG_BIN=/usr/pgsql-18/bin

log() { echo "pfw-init: $*"; }

# Two-phase guard. PG_VERSION (written by initdb) protects the DANGEROUS,
# never-repeat op — we must never re-initdb over a real cluster. DIST_FILE is the
# "provisioning complete" sentinel, written LAST, and gates the RECOVERABLE,
# idempotent post-init steps (grafana DB, extension, markers). A crash between
# initdb and DIST_FILE is thus repaired on the next boot rather than skipped
# forever (the earlier single PG_VERSION guard extended "done" to these
# recoverable steps and left Grafana's DB uncreated permanently).
pg_cluster_exists() { [[ -s "${POSTGRES_DATA_DIR}/PG_VERSION" ]]; }
srv_provisioned()   { [[ -f "${DIST_FILE}" || -f "${LEGACY_DIST_FILE}" ]]; }

readonly SECRETS_DIR="${SRV}/.pfw-secrets"
readonly MANAGED_DB_SECRET="${SECRETS_DIR}/managed-db.env"
readonly GRAFANA_SECRET="${SECRETS_DIR}/grafana.env"
readonly CLICKHOUSE_SECRET="${SECRETS_DIR}/clickhouse.env"

# Presence means: at least one credential in SECRETS_DIR is a value generated on
# this host and recorded nowhere else. Deliberately OUTSIDE SECRETS_DIR and NOT
# hidden. A marker inside the directory would disappear with the thing it
# describes, and a dotfile beside it would disappear for the very same reason the
# directory does -- `cp -r /srv/*`, `tar`/`rsync` without `.[!.]*`, and operator
# cleanups all carry the visible entries of /srv and silently drop the hidden ones.
# A plain file at the top of /srv travels with /srv/postgres18, which is what makes
# the mismatch (cluster here, secrets gone) detectable at all.
#
# It is NOT written into DIST_FILE: managed/utils/distribution parses that file's
# contents, so an extra line there would be a different bug.
readonly GENERATED_MARKER="${SRV}/pfw-secrets-generated"

# Values used before credentials were generated. A host provisioned by an older
# build still authenticates with these, so relocation must preserve them exactly
# -- only a fresh install gets a generated value. Rotating an existing role here
# would need PostgreSQL up and an ALTER ROLE, which is not worth breaking first
# boot for; see the design doc's relocation/generation split.
readonly LEGACY_MANAGED_DB_PASSWORD=pmm-managed
readonly LEGACY_GRAFANA_DB_PASSWORD=grafana
readonly LEGACY_CLICKHOUSE_PASSWORD=clickhouse
# sha256("clickhouse") -- the value shipped in default-users.xml:57 and
# low-memory-users.xml:62. Used as EVIDENCE, not as a credential: its presence in
# the deployed XML proves this host still uses the shipped constant.
readonly LEGACY_CLICKHOUSE_HASH=7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3
# The file the running server actually obeys, which is what makes it evidence:
# pfw-clickhouse.service passes --config-file=/etc/clickhouse-server/config.xml, that
# symlink points at default-config.xml, and its <user_directories><users_xml><path>
# names default-users.xml. users.xml is a vestigial symlink nothing reads, and
# switch-config.sh is a deprecated stub. If any link in that chain changes, this
# check stops meaning what it claims and must be revisited.
readonly CLICKHOUSE_USERS_XML=/etc/clickhouse-server/default-users.xml
# The other copy of the same credential. low-memory-config.xml names it, and it is
# the documented remedy for hosts under 16 GB, so a credential left behind here is
# live the moment anyone switches profile.
readonly CLICKHOUSE_LOWMEM_USERS_XML=/etc/clickhouse-server/low-memory-users.xml

# Hex, not base64. The value has to survive three parsers unescaped: the
# identifier/password guard in provision_app_db, systemd's EnvironmentFile parser
# (a trailing backslash is a line continuation that would swallow the next line),
# and a SQL string literal.
gen_password() { openssl rand -hex 16; }

# read_secret <file> <key> -- values are hex or the legacy literals, so none need
# unquoting. Deliberately not `source`: that would execute the file.
read_secret() { sed -n "s/^$2=//p" "$1"; }

# write_secret <file> <line>... -- written to a temp file and renamed so a reader
# never sees a half-written file. UMask=0077 in the unit makes the create 0600
# from the first syscall; the chmod is belt-and-braces for a manual run.
write_secret() {
    local f=$1; shift
    printf '%s\n' "$@" > "${f}.tmp"
    chmod 600 "${f}.tmp"
    mv -f "${f}.tmp" "${f}"
}

# assert_secret_store_intact refuses to write the shipped constant over a generated
# credential. Without it, "provisioned AND no secret file" is read as "this host
# predates the secret store", which stops being true the moment this release ships:
# every host it provisions is provisioned, and losing SECRETS_DIR (see
# GENERATED_MARKER for how easily that happens) would then silently rewrite the
# public constant. pfw-managed makes that worse rather than merely breaking: its
# initWithRoot runs ALTER USER ... WITH PASSWORD with whatever it is handed
# (managed/models/database.go), so the role would be rotated BACK to the constant and
# the disclosure this whole change exists to close would reopen, silently. Grafana
# has no such path and would simply wedge.
assert_secret_store_intact() {
    [ -f "${GENERATED_MARKER}" ] || return 0
    local f
    for f in "${MANAGED_DB_SECRET}" "${GRAFANA_SECRET}"; do
        if [ -f "${f}" ]; then continue; fi
        log "FATAL: ${GENERATED_MARKER} records that this host's bootstrap credentials" >&2
        log "       were generated, but ${f} is missing. The PostgreSQL roles still hold" >&2
        log "       the generated passwords, so this script will NOT write the shipped" >&2
        log "       constant over them." >&2
        log "       Restore ${SECRETS_DIR} from a backup -- it is hidden and 0700, so a" >&2
        log "       copy taken with 'cp -r ${SRV}/*' or a tar/rsync without '.[!.]*' will" >&2
        log "       have skipped it -- or reset the roles by hand as the postgres" >&2
        log "       superuser and write the new values back into ${f}." >&2
        log "       Only delete ${GENERATED_MARKER} if you are certain the roles really" >&2
        log "       do still use the historical constants." >&2
        exit 1
    done
}

# ensure_clickhouse_secret deliberately does NOT join assert_secret_store_intact's
# loop. That guard turns a missing file into a fatal error whenever the marker is
# present, which would break every host upgrading from the release that introduced
# the store: such a host legitimately has the marker and both PostgreSQL secrets,
# and no clickhouse.env at all.
#
# ClickHouse also offers a better signal than inference. The password is stored as a
# sha256 in a root-owned XML that %post deploys once, so the deployed file states
# which credential the server actually accepts. If it still carries the shipped
# constant's hash, writing the constant is correct. If it does not, the password was
# generated and writing the constant would leave pmm-managed and qan-api2
# authenticating with the wrong value -- so refuse, rather than guess.
#
# The XML is 0644 and /etc is readable under ProtectSystem=strict, so the pfw user
# can consult it even though it cannot write there.
ensure_clickhouse_secret() {
    [ -f "${CLICKHOUSE_SECRET}" ] && return 0

    if [ ! -r "${CLICKHOUSE_USERS_XML}" ]; then
        # Reachable only when BOTH the secret and the XML are missing: if %post had
        # written the secret, the guard above would already have returned. So this is
        # not a benign "not deployed yet" -- pfw-managed, pfw-qan-api2 and pfw-grafana
        # all read the secret as a non-optional EnvironmentFile and will refuse to
        # start, with nothing anywhere saying why. Warn rather than exit: the
        # ClickHouse config genuinely may not be deployed on a host where %post's
        # first-install branch was skipped, and killing first boot over it would be
        # worse than the units failing with a message an operator can search for.
        log "WARNING: ${CLICKHOUSE_SECRET} is missing and ${CLICKHOUSE_USERS_XML} is" >&2
        log "         unreadable, so the ClickHouse credential cannot be determined." >&2
        log "         pfw-managed, pfw-qan-api2 and pfw-grafana read that file as a" >&2
        log "         required EnvironmentFile and will not start until it exists." >&2
        log "         Reinstall pfw-server, or write PMM_CLICKHOUSE_PASSWORD=<value>" >&2
        log "         into ${CLICKHOUSE_SECRET} (0600 pfw:pfw) matching the" >&2
        log "         password_sha256_hex in ${CLICKHOUSE_USERS_XML}." >&2
        return 0
    fi

    if grep -q "${LEGACY_CLICKHOUSE_HASH}" "${CLICKHOUSE_USERS_XML}"; then
        log "relocating the existing ClickHouse credential into ${CLICKHOUSE_SECRET}"
        write_secret "${CLICKHOUSE_SECRET}" \
            "PMM_CLICKHOUSE_PASSWORD=${LEGACY_CLICKHOUSE_PASSWORD}"
        return 0
    fi

    log "FATAL: ${CLICKHOUSE_USERS_XML} does not carry the shipped constant's hash, so" >&2
    log "       this host's ClickHouse password was generated -- but ${CLICKHOUSE_SECRET}" >&2
    log "       is missing. Writing the constant would leave pfw-managed, pfw-qan-api2" >&2
    log "       and pfw-grafana unable to authenticate." >&2
    log "       Restore ${SECRETS_DIR} from a backup, or choose a new password and set" >&2
    log "       its sha256 in BOTH of these -- they hold the same credential, and a" >&2
    log "       value left in the second one goes live the moment a low-memory host" >&2
    log "       switches profile:" >&2
    log "         ${CLICKHOUSE_USERS_XML}" >&2
    log "         ${CLICKHOUSE_LOWMEM_USERS_XML}" >&2
    log "       then write the plaintext into ${CLICKHOUSE_SECRET} (0600 pfw:pfw)." >&2
    exit 1
}

# sync_generated_marker derives the marker from the FILES rather than from the branch
# that wrote them. Setting it as a side effect of generation would lose it to a crash
# between the two writes (and could leave it claiming a credential that was never
# written); deriving it means a marker deleted by hand, or missed by an older build,
# is repaired on the next boot. An operator who rotates a legacy host's passwords by
# hand and updates these files gets the protection for free, which is correct: those
# values are no longer the constants either.
sync_generated_marker() {
    if [ -f "${GENERATED_MARKER}" ]; then return 0; fi
    local m g
    m=$(read_secret "${MANAGED_DB_SECRET}" PMM_POSTGRES_DBPASSWORD)
    g=$(read_secret "${GRAFANA_SECRET}" GF_DATABASE_PASSWORD)
    if [ "${m}" = "${LEGACY_MANAGED_DB_PASSWORD}" ] && [ "${g}" = "${LEGACY_GRAFANA_DB_PASSWORD}" ]; then
        return 0   # both still the historical constants: nothing to lose
    fi
    printf '%s\n' \
        "This host's bootstrap credentials in ${SECRETS_DIR} were generated here and" \
        "exist nowhere else. Back that directory up: it is hidden and mode 0700, so" \
        "'cp -r ${SRV}/*' and tar/rsync without '.[!.]*' skip it silently." \
        "pfw-init refuses to start if this file is present and the secret files are" \
        "not, rather than writing the shipped constant over a generated password." \
        > "${GENERATED_MARKER}"
}

# ensure_secrets must run BEFORE provision_srv: provision_databases needs the
# values, and provision_srv returns early on an already-provisioned host, which is
# exactly the host that still needs its credential relocated out of the 0644 unit.
ensure_secrets() {
    install -d -m 0700 "${SECRETS_DIR}"
    assert_secret_store_intact
    ensure_clickhouse_secret

    if [ ! -f "${MANAGED_DB_SECRET}" ]; then
        local pw
        if srv_provisioned; then
            pw=${LEGACY_MANAGED_DB_PASSWORD}
            log "relocating the existing pfw-managed DB credential into ${MANAGED_DB_SECRET}"
        else
            pw=$(gen_password)
            log "generating the pfw-managed DB credential"
        fi
        write_secret "${MANAGED_DB_SECRET}" "PMM_POSTGRES_DBPASSWORD=${pw}"
    fi

    if [ ! -f "${GRAFANA_SECRET}" ]; then
        if srv_provisioned; then
            # Relocation only. The admin user already exists on this host, and
            # Grafana applies admin_password solely when creating it, so writing
            # GF_SECURITY_ADMIN_PASSWORD here would be inert and misleading.
            log "relocating the existing grafana DB credential into ${GRAFANA_SECRET}"
            write_secret "${GRAFANA_SECRET}" \
                "GF_DATABASE_PASSWORD=${LEGACY_GRAFANA_DB_PASSWORD}"
        else
            log "generating the grafana DB and admin credentials"
            write_secret "${GRAFANA_SECRET}" \
                "GF_DATABASE_PASSWORD=$(gen_password)" \
                "GF_SECURITY_ADMIN_PASSWORD=$(gen_password)"
            # The path, never the value. journald persists to /var/log/journal,
            # where anything in adm or systemd-journal could read it -- a weaker
            # form of the disclosure this change exists to close.
            log "initial admin password written to ${GRAFANA_SECRET}; read it with:"
            log "  sudo sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' ${GRAFANA_SECRET}"
        fi
    fi

    sync_generated_marker
}

ensure_srv_dirs() {
    # No grafana/plugins here on purpose. This used to copy
    # /opt/postgres1st/watchtower/dashboards/panels into ${SRV}/grafana/plugins, and
    # because provisioning only runs once (guarded by srv_provisioned), that copy
    # was never refreshed: upgrading percona-dashboards or pfw-managed updated the
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
# role unable to create its own tables — both grafana's and pfw-managed's
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
    # ON_ERROR_STOP because the CREATE USER below is fed on stdin: psql exits 0 on a
    # failed statement from a script, so without it errexit would never see the error.
    local pg=(/usr/bin/psql -v ON_ERROR_STOP=1 -U postgres -h /run/postgresql -d postgres)
    if [ "$("${pg[@]}" -tAc "SELECT 1 FROM pg_database WHERE datname='${db}'")" != "1" ]; then
        "${pg[@]}" -c "CREATE DATABASE \"${db}\""
    fi
    if [ "$("${pg[@]}" -tAc "SELECT 1 FROM pg_roles WHERE rolname='${role}'")" != "1" ]; then
        # On stdin, NOT -c: anything on psql's argv is in /proc/<pid>/cmdline, which
        # is world-readable by default, so `-c "... PASSWORD '<pw>'"` publishes the
        # generated credential to every local user for the life of the process. That
        # is the same local-disclosure class this script exists to close.
        "${pg[@]}" -f - <<EOSQL
CREATE USER "${role}" LOGIN PASSWORD '${pw}';
EOSQL
    fi
    "${pg[@]}" -c "GRANT ALL PRIVILEGES ON DATABASE \"${db}\" TO \"${role}\""
    "${pg[@]}" -c "ALTER DATABASE \"${db}\" OWNER TO \"${role}\""
}

# provision_databases brings PostgreSQL up briefly (the pg_ctl-owned instance,
# before pfw-postgresql.service) for the idempotent one-time DB setup. A RETURN
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
    provision_app_db grafana grafana \
        "$(read_secret "${GRAFANA_SECRET}" GF_DATABASE_PASSWORD)"

    # pfw-managed creates its own DB/role on first start, but does so as the
    # postgres superuser, which leaves the database owned by postgres — fatal on
    # PG15+ (see provision_app_db). Pre-create them here with the right owner;
    # pfw-managed skips creation when they already exist.
    log "creating pfw-managed database and user ..."
    provision_app_db pmm-managed pmm-managed \
        "$(read_secret "${MANAGED_DB_SECRET}" PMM_POSTGRES_DBPASSWORD)"
}

# provision_srv runs the one-time /srv provisioning, guarded so it recovers from
# a partial previous run (see the two-phase guard note above).
provision_srv() {
    if srv_provisioned; then
        # A host provisioned against an older PostgreSQL major has the sentinel but
        # no cluster at the current POSTGRES_DATA_DIR (the path carries the major:
        # /srv/postgres14 -> /srv/postgres18). Returning here would skip initdb and
        # leave pfw-postgresql to fail on an empty data directory, which reads like
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
    if [ -f /opt/postgres1st/watchtower/dashboards/VERSION ]; then
        cp /opt/postgres1st/watchtower/dashboards/VERSION "${SRV}/grafana/PERCONA_DASHBOARDS_VERSION"
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
                log "FATAL: ${ssl_src}/${f} missing; the pfw-server RPM must ship the nginx SSL sources." >&2
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

    # Validate the config pfw-nginx.service actually runs (/etc/nginx/pfw.conf via
    # `nginx -c`), NOT the base nginx package's default /etc/nginx/nginx.conf — and
    # route the startup error log to stderr so the check does not fail trying to open
    # the root-only /var/log/nginx/error.log as the pfw user.
    log "validating nginx configuration ..."
    nginx -t -c /etc/nginx/pfw.conf -e stderr

    # Create the pmm-agent config (idempotent) so pfw-agent.service can start.
    # Native keeps it under /srv (pfw-writable, persistent) rather than the
    # container's root-owned /opt/postgres1st/watchtower/config; pfw-agent.service's
    # --config-file points here to match.
    install -d -m 770 /srv/pfw-agent/config /srv/pfw-agent/tmp /srv/nomad/data
    local agent_cfg=/srv/pfw-agent/config/pfw-agent.yaml
    if [ ! -f "${agent_cfg}" ]; then
        log "creating pfw-agent configuration ..."
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
        /usr/sbin/pfw-agent setup \
            --config-file="${agent_cfg}" \
            --skip-registration \
            --id=watchtower-server \
            --paths-tempdir=/srv/pfw-agent/tmp \
            --paths-nomad-data-dir=/srv/nomad/data \
            --server-address=127.0.0.1:8443 \
            --server-insecure-tls \
            "${node_addr}"
    fi
}

main() {
    # /srv must be owned/writable by the pfw user (the RPM %post chowns it, T7).
    # Fail loud with guidance rather than dying on the first mkdir's EACCES.
    if [ ! -w "${SRV}" ]; then
        log "FATAL: ${SRV} is not writable by the pfw user (uid $(id -u)/gid $(id -g))." >&2
        log "The RPM %post must run 'chown -R pfw:pfw ${SRV}' before pfw-init starts." >&2
        exit 1
    fi

    ensure_secrets
    provision_srv
    prepare_runtime
    log "done."
}

main "$@"
