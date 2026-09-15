#!/bin/bash
# pfw-cert-expiry-check.sh — writes the nginx TLS certificate's expiry as a
# node_exporter textfile metric, so it shows up in VictoriaMetrics without any
# exporter code changes.
#
# Installed to /usr/share/pfw/pfw-cert-expiry-check.sh, run by
# pfw-cert-expiry-check.service (oneshot) on pfw-cert-expiry-check.timer.
#
# Contract: must be safe to run on every timer fire, and must never fail the
# unit just because the certificate is temporarily unreadable mid-rotation --
# a stale metric is far less harmful than a check that stops reporting.
set -o errexit
set -o nounset
set -o pipefail

readonly CERT=/srv/nginx/certificate.crt
readonly OUT_DIR=/opt/postgres1st/watchtower/collectors/textfile-collector/low-resolution
readonly OUT_FILE="${OUT_DIR}/pfw_nginx_cert_expiry.prom"
readonly TMP_FILE="${OUT_FILE}.$$"

log() { echo "pfw-cert-expiry-check: $*"; }

cleanup() { rm -f "${TMP_FILE}"; }
trap cleanup EXIT

if [ ! -r "${CERT}" ]; then
    log "WARNING: ${CERT} not readable; leaving any existing metric file in place"
    exit 0
fi

end_date=$(openssl x509 -enddate -noout -in "${CERT}" 2>/dev/null | sed 's/^notAfter=//')
if [ -z "${end_date}" ]; then
    log "WARNING: could not read notAfter from ${CERT}; leaving any existing metric file in place"
    exit 0
fi

expiry_epoch=$(date -d "${end_date}" +%s 2>/dev/null) || {
    log "WARNING: could not parse expiry date '${end_date}'; leaving any existing metric file in place"
    exit 0
}

{
    echo "# HELP pfw_nginx_cert_expiry_seconds Unix timestamp when the nginx TLS certificate expires."
    echo "# TYPE pfw_nginx_cert_expiry_seconds gauge"
    echo "pfw_nginx_cert_expiry_seconds ${expiry_epoch}"
} > "${TMP_FILE}"

# node_exporter's textfile collector requires an atomic replace: it polls the
# directory and would otherwise risk reading a half-written file.
mv -f "${TMP_FILE}" "${OUT_FILE}"
chmod 0644 "${OUT_FILE}"
