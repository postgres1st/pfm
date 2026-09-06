#!/usr/bin/env bash
# Hand the secret store to root:pfw so systemd can read EnvironmentFile= for units
# that drop CAP_DAC_OVERRIDE. Run as root via the `+` prefix in pfw-init.service.
#
# Idempotent: safe on every boot, and a no-op when the store is already correct.
set -o errexit
set -o nounset
set -o pipefail

readonly SECRETS_DIR=/srv/.pfw-secrets

[ -d "${SECRETS_DIR}" ] || exit 0

# 0770, not 0700: pfw must still traverse it AND be able to create a secret on a
# later boot (ensure_secrets adds files over time). Owner root so that systemd's
# capability-less child can read the files as their owner. Only root and the pfw
# group can traverse, so the directory is what keeps the store private.
chown root:pfw "${SECRETS_DIR}"
chmod 0770 "${SECRETS_DIR}"

# 0640, not 0600: same reason on the files themselves. No other local user can
# reach them -- the directory denies traversal to everyone but root and pfw.
shopt -s nullglob
for f in "${SECRETS_DIR}"/*.env; do
    chown root:pfw "${f}"
    chmod 0640 "${f}"
done
