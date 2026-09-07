#!/usr/bin/env bash
# Label the bootstrap credential store so PID 1 can read it, then prove it worked.
#
# Runs as root from pfw-init.service (ExecStartPost=+) because relabelling needs root;
# pfw-init itself is User=pfw. This mirrors pfw-nginx.service's ExecStartPre=-+restorecon
# on /srv/nginx, for the same underlying reason: we own a layout under /srv, so we own
# its labels.
#
# WHY IT IS NOT OPTIONAL. systemd opens EnvironmentFile= as PID 1 -- init_t, a CONFINED
# domain -- before the unit's User= and CapabilityBoundingSet= ever apply to the child.
# /srv is var_t and init_t has no rule permitting open on var_t files, so an unlabelled
# store makes pfw-managed, pfw-grafana and pfw-qan-api2 fail with
#
#     Failed to load environment files: Permission denied
#
# and restart forever with readyz at 500 -- while their own logs say nothing, because
# the process never starts. That the three services are themselves unconfined is
# irrelevant: the reader is systemd, not the service.
#
# WHY THE ASSERTION. pfw-server.spec loads the policy module with
# `semodule -n -i ... || :` -- a deliberate no-fail so the RPM still installs on a host
# with no SELinux tooling. A failed policy load is therefore SILENT, and without this
# check its only symptom is the boot failure above, three units away from the cause.
set -o nounset

readonly SECRETS_DIR=/srv/.pfw-secrets
readonly WANT_TYPE=pfw_secret_t

[ -d "${SECRETS_DIR}" ] || exit 0

# A host without SELinux is not an error -- our own test containers have no policy
# store at all. Exit quietly rather than failing a boot that is fine.
command -v selinuxenabled >/dev/null 2>&1 || exit 0
selinuxenabled || exit 0

if command -v restorecon >/dev/null 2>&1; then
    # -R because the files matter, not just the directory. Files created later inherit
    # the directory's type, so this is the only relabel a normal host ever needs.
    restorecon -R "${SECRETS_DIR}" 2>/dev/null || true
fi

ctx=$(stat -c %C "${SECRETS_DIR}" 2>/dev/null)
case "${ctx}" in
    *:"${WANT_TYPE}":*) exit 0 ;;
esac

if [ "$(getenforce 2>/dev/null)" = "Enforcing" ]; then
    {
        printf 'pfw-init: %s is labelled %s, not %s.\n' \
            "${SECRETS_DIR}" "${ctx:-<unlabelled>}" "${WANT_TYPE}"
        printf 'pfw-init: PID 1 cannot open EnvironmentFile= with that label, so\n'
        printf 'pfw-init: pfw-managed, pfw-grafana and pfw-qan-api2 will not start.\n'
        printf 'pfw-init: the policy module is most likely not loaded. Check:\n'
        printf 'pfw-init:   semodule -l | grep pfw\n'
        printf 'pfw-init: and repair with:\n'
        printf 'pfw-init:   semodule -i %s/pfw_nginx.pp && restorecon -R %s\n' \
            /usr/share/selinux/packages "${SECRETS_DIR}"
    } >&2
    exit 1
fi

# Permissive: the same wrong label, but the host boots regardless. Warn rather than
# fail, so a deliberately permissive install is not blocked by a cosmetic mismatch --
# while still leaving the reason in the journal for whoever turns enforcing back on.
printf 'pfw-init: warning: %s is labelled %s, not %s; SELinux is permissive so this boots anyway.\n' \
    "${SECRETS_DIR}" "${ctx:-<unlabelled>}" "${WANT_TYPE}" >&2
exit 0
