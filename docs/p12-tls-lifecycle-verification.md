# P12 — Production TLS lifecycle: verification guide

Roadmap outcome (as written): "Validate custom certificate installation, renewal,
replacement, expiry monitoring and recovery from invalid or expired certificates."

This is the manual playbook for that validation. Run it against a **real native EL9
host with `pfw-server` installed** — the systemd units, the `pfw` account and
`/usr/share/pfw/pfw-cert-expiry-check.sh` referenced below only exist after a real
`dnf install pfw-server`, not in the devcontainer. See "Devcontainer differences" at
the bottom if you want to try a reduced version there anyway.

What's actually implemented as of this writing: the expiry-monitoring script, its
systemd timer, the alert template, and the documented replace/recovery procedures in
`build/packages/pfw-airgap-INSTALL.md`. The script and alert-template logic have been
verified against a running devcontainer (metric generation, VictoriaMetrics ingestion,
PromQL evaluation, and the alert template loading against the real validation code).
The RPM packaging, SELinux labeling on the new unit files, and `%post` enablement have
**not** been verified — that needs the native VM run this whole document assumes.

---

## Setup — a second, distinguishable test certificate

Generate once; you'll swap it in and out through the guide to prove install/replace/
renewal actually take effect, not just that the files changed on disk.

```bash
cd /tmp
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout test-good.key -out test-good.crt \
  -subj "/CN=test-verification-cert"
```

---

## 1. Custom certificate installation / replacement

```bash
# Note the currently-active cert's serial number first (so you can tell it changed)
sudo openssl x509 -noout -serial -in /srv/nginx/certificate.crt

# Install with `install`, not cp/mv -- this is what gives the file the correct
# SELinux label (pfw_cert_t). See build/packages/pfw-airgap-INSTALL.md.
sudo install -o pfw -g pfw -m 0644 /tmp/test-good.crt /srv/nginx/certificate.crt
sudo install -o pfw -g pfw -m 0600 /tmp/test-good.key /srv/nginx/certificate.key
sudo systemctl restart pfw-nginx

# Confirm nginx picked it up -- serial should now match test-good.crt's
sudo openssl x509 -noout -serial -in /srv/nginx/certificate.crt
curl -sk -v https://127.0.0.1:8443/ 2>&1 | grep -i "subject\|CN="
```

**Pass:** serial changed, curl shows `CN=test-verification-cert`, and
`systemctl is-active pfw-nginx` reports `active`.

## 2. Renewal

There is no automated renewal in this product — it is a bring-your-own-certificate
design. "Renewal" is the same replace procedure, repeated. Verify it is genuinely
repeatable and not a one-time-only path (`pfw-init` only *generates* a cert when the
files are absent; confirm that doesn't somehow block a second manual replace).

```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout /tmp/test-good2.key -out /tmp/test-good2.crt \
  -subj "/CN=test-verification-cert-v2"

sudo install -o pfw -g pfw -m 0644 /tmp/test-good2.crt /srv/nginx/certificate.crt
sudo install -o pfw -g pfw -m 0600 /tmp/test-good2.key /srv/nginx/certificate.key
sudo systemctl restart pfw-nginx

curl -sk -v https://127.0.0.1:8443/ 2>&1 | grep "CN="
```

**Pass:** now shows `CN=test-verification-cert-v2`.

## 3. Expiry monitoring

```bash
# Run the check manually instead of waiting for the hourly timer
sudo -u pfw /usr/share/pfw/pfw-cert-expiry-check.sh
cat /opt/postgres1st/watchtower/collectors/textfile-collector/low-resolution/pfw_nginx_cert_expiry.prom
```

**Pass:** file exists, contains `pfw_nginx_cert_expiry_seconds <epoch>`, matching
`openssl x509 -enddate -noout -in /srv/nginx/certificate.crt`.

```bash
# Confirm it reached VictoriaMetrics (wait up to ~60s for the low-resolution scrape)
curl -sk "http://127.0.0.1:9090/prometheus/api/v1/query?query=pfw_nginx_cert_expiry_seconds"
```

**Pass:** non-empty `result`, value matches the `.prom` file.

```bash
# Confirm the timer itself is enabled and will fire on its own
systemctl status pfw-cert-expiry-check.timer
systemctl list-timers pfw-cert-expiry-check.timer
```

**Pass:** `enabled`, `active (waiting)`, a next-trigger time is shown.

### Force the alert to actually fire

Proves the whole chain end to end, not just that the metric exists.

```bash
# Install a cert expiring in 2 days -- well under the 14-day default threshold
openssl req -x509 -nodes -days 2 -newkey rsa:2048 \
  -keyout /tmp/test-expiring.key -out /tmp/test-expiring.crt \
  -subj "/CN=test-expiring-soon"
sudo install -o pfw -g pfw -m 0644 /tmp/test-expiring.crt /srv/nginx/certificate.crt
sudo install -o pfw -g pfw -m 0600 /tmp/test-expiring.key /srv/nginx/certificate.key
sudo systemctl restart pfw-nginx
sudo -u pfw /usr/share/pfw/pfw-cert-expiry-check.sh
```

Then in the UI: **Alerts → Alert templates**, find "Server TLS certificate about to
expire," create a rule from it with the default 14-day threshold, and check
**Alerts → Alert rules** a few minutes later.

**Pass:** the rule is shown as firing.

## 4. Recovery from an expired certificate

Confirms nginx *tolerates* expiry (doesn't crash) and the operator's browser correctly
warns — this is a different failure mode from an invalid certificate (§5).

```bash
openssl req -x509 -nodes -days -30 -newkey rsa:2048 \
  -keyout /tmp/test-expired.key -out /tmp/test-expired.crt \
  -subj "/CN=test-already-expired"
sudo install -o pfw -g pfw -m 0644 /tmp/test-expired.crt /srv/nginx/certificate.crt
sudo install -o pfw -g pfw -m 0600 /tmp/test-expired.key /srv/nginx/certificate.key
sudo systemctl restart pfw-nginx
systemctl is-active pfw-nginx        # expect: active
curl -sk -v https://127.0.0.1:8443/ 2>&1 | grep -i "expire\|verify"
```

**Pass:** nginx stays `active` (no crash-loop); curl shows an expiry/verification
warning.

## 5. Recovery from an invalid certificate

```bash
echo "not a real certificate" | sudo tee /srv/nginx/certificate.crt
sudo systemctl restart pfw-nginx
sleep 3
systemctl status pfw-nginx           # expect: failed, or restarting rapidly
sudo journalctl -u pfw-nginx -n 20 --no-pager
```

**Pass:** nginx fails to start, and the reason is visible in the log.

Now follow the documented recovery runbook
(`build/packages/pfw-airgap-INSTALL.md`, "Recover from an invalid or expired
certificate"):

```bash
sudo rm -f /srv/nginx/certificate.crt /srv/nginx/certificate.key
sudo systemctl restart pfw-init pfw-nginx
systemctl is-active pfw-nginx        # expect: active
sudo openssl x509 -noout -subject -in /srv/nginx/certificate.crt   # fresh self-signed cert
```

**Pass:** nginx recovers to `active` with a freshly regenerated self-signed
certificate. `pfw-init` is `RemainAfterExit`, so restarting `pfw-nginx` alone will
*not* trigger it — both units must be restarted together, which is exactly what this
step is checking.

---

## Restore afterward

```bash
sudo rm -f /srv/nginx/certificate.crt /srv/nginx/certificate.key
sudo systemctl restart pfw-init pfw-nginx
```

---

## Devcontainer differences

The devcontainer runs everything as user `pmm`, not `pfw` (`id pfw` → no such user),
and has no systemd units at all (supervisord instead) — `pfw-nginx`,
`pfw-cert-expiry-check.timer`, `pfw-init` don't exist there. `/srv/nginx/certificate.crt`
does exist at the same path, and `openssl x509` commands against it work unmodified.

If adapting steps for the devcontainer:

- Drop `sudo`; run via `docker exec pmm-server <command>` (or `docker exec -it
  pmm-server bash` for a shell) — `docker exec` is root by default.
- Replace `-o pfw -g pfw` with `-o pmm -g pmm` (or drop ownership flags entirely).
- There is no `pfw-cert-expiry-check.sh` at `/usr/share/pfw/` unless you've manually
  copied it in — the RPM install step that places it there hasn't run.
- The alert-templates directory is `/usr/local/percona/alerting-templates`, not
  `/opt/postgres1st/watchtower/alerting-templates`.
- The textfile-collector directory is
  `/usr/local/percona/pmm/collectors/textfile-collector/`, not
  `/opt/postgres1st/watchtower/collectors/textfile-collector/`.
- Certificate replacement and the expiry-monitoring *script logic* can be verified
  this way; the systemd timer, SELinux labeling, and crash-loop/recovery behavior
  cannot — those only exist under a real RPM install.
