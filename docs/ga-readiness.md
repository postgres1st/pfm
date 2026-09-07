# Readiness for GA — the internal TODO

Things deliberately **not** done for beta2, each with the decision that put it here rather
than a vague "later". beta2 is a beta precisely so these can wait; GA is where they cannot.

Blockers for **beta2** live in `docs/beta2-release-readiness.md`. This list is the next
horizon.

---

## 1. Follow the full release process, as beta1 did

beta2 has been built, signed and validated from a working tree rather than through the
release process beta1 went through. The artefacts are sound — every suite passes against
the bundle, and it is signed with the release key — but "we ran the process" and "the
output looks right" are different claims, and only the second is currently true.

Do the whole process for GA: the release tag, the recorded provenance, and the archived
artefacts, so the build is reproducible from a tag rather than from whatever was in the
tree that afternoon.

## 2. Rename the internal database and role

The server's own PostgreSQL database and its role are still literally `pmm-managed`:

```
build/packages/config/pfw/pfw-init.sh:356
    provision_app_db pmm-managed pmm-managed \
        "$(read_secret "${MANAGED_DB_SECRET}" PMM_POSTGRES_DBPASSWORD)"
```

The **password is not** — it is generated per install and read from a secret file. Only
the database name and the role remain. `LEGACY_MANAGED_DB_PASSWORD=pmm-managed`
(`pfw-init.sh:62`) survives solely to adopt an installation created before credentials
were generated.

Note `docs/pfw-identifier-map.md` §C.1 still says "database, role **and password** — all
literally `pmm-managed`". That was true when written and is now stale: correct it when
this item is picked up.

Tier C: stored inside the cluster, so renaming needs `ALTER DATABASE` and `ALTER ROLE`
coordinated across the unit, the init script and Grafana's environment file.

The service name was renamed for beta2 (`watchtower-postgresql`, `watchtower-db`) because
it is visible in Inventory and cost nothing. This one is not visible and does cost
something, so it waits — but it should not survive GA, and the free window closes the
moment a customer has an installation to migrate.

## 3. Negative-control `test-pfw-client`

`build/scripts/test-pfw-client` is the only suite that executes the documented customer
install — `pfw-admin config` then `pfw-admin add` with no `--server-url`, on a separate
host. It passes 9/0.

It has also never failed, and unlike `test-pfw-airgap` and `test-pfw-upgrade` its
assertions have not been through `test-pfw-negative-control`. A suite that has only ever
been green has not yet shown it can go red, and two of its assertions were wrong on the
first run in a way that looked like product failure. Break each one deliberately —
uninstall the agent, feed a bad password, point it at the wrong host — and require each to
notice, before relying on it as the evidence that a customer's first contact works.

## 4. Move the credential store out of `/srv`

beta2 keeps the bootstrap credentials at `/srv/.pfw-secrets` and gives them their own
SELinux type, `pfw_secret_t`, because PID 1 reads them through `EnvironmentFile=` as
`init_t` and cannot open generic `var_t`. That works, and it is guarded.

It is still a label the product has to *maintain*. Three things must all hold on every
host, forever: the policy module loads, the `.fc` rule matches, and the
`ExecStartPost` relabel runs. `pfw-server.spec` loads the module with `semodule -n -i
… || :`, so a failed load is silent — the assertion in `pfw-secrets-label.sh` exists
precisely because that silence once cost a day. An admin running `restorecon -R /srv`
on a host where the module did not load will relabel the store back to `var_t` and stop
the server, with the cause three units away from the symptom.

Putting the store under `/etc/pfw/` removes the whole class: `etc_t` comes from default
labelling, no policy module is involved, and any `restorecon` anywhere produces the
right answer. It is also where systemd `EnvironmentFile=` material conventionally lives.

Not done for beta2 because it moves a path named in `INSTALL.md` (which documents
reading the initial admin password from `/srv/.pfw-secrets/grafana.env`), five unit
files, `pfw-init.sh`, `docs/pfw-identifier-map.md` and the backup guidance — a rename
sweep, to fix a bug that one `.fc` line already fixes. The right trade at GA, not
mid-cycle.

Considered and rejected: labelling the store `etc_t` in place. It boots, but `etc_t` is
readable by a long list of confined domains and these files are generated passwords, so
a private type read by `init_t` alone is strictly tighter.

## 5. Guard every spec's version, not just `pfw-server`'s

`PFW_VERSION` in `build/scripts/pfw-airgap-vars` is the single source of truth, and
`test-frozen-identifiers` asserts that `pfw-server.spec` agrees with it. Nothing asserts
it for the other eight specs.

Four of them carry a **placeholder** version in the tree — `pfw-managed.spec` and
`vmproxy.spec` say `2.0.0`, `grafana.spec` and `pfw-qan-api2.spec` say `3.0.0` — which
`build-pfw-airgap` rewrites at build time:

```python
s = re.sub(r'%define full_pmm_version 3\.0\.0',
           '%define full_pmm_version ' + version, s, count=1)
```

That regex is pinned to the literal placeholder, uses `count=1`, and **asserts nothing**.
Change a placeholder, reformat the line, or add a fifth spec with a different literal, and
the substitution silently does not match: the package ships versioned `2.0.0`, sorts below
every real release, and `dnf upgrade` skips it forever. The failure is invisible at build
time and shows up as a customer who cannot upgrade.

This is the same shape as every defect this cycle — two lists that must agree with nothing
comparing them — and the same shape as the SELinux label defect that stopped the server
booting. It is not a beta2 blocker because the built artefacts were checked by hand: all
of ours came out `3.9.0~beta2`, and the third-party packages correctly kept their upstream
versions. "Checked by hand once" is exactly what a guard replaces.

The guard should compare each spec's **resolved** version against `PFW_VERSION` — resolved,
because reading the placeholder out of the tree would assert the wrong thing. Deriving the
spec list from `build/packages/rpm/**` rather than naming them keeps a newly added spec
from being silently uncovered.

## 6. Rewrite the documentation site

Covered in full in `docs/deferred-documentation-rebrand.md`. Its conclusion is the part
worth repeating here: it is a **rewrite, not a rename**. The install section documents five
deployment paths this product does not ship and omits the one it does, so renaming
binaries inside those pages would make wrong instructions look more authoritative.

Shipping beta2 rather than GA is what buys the time for this.

---

## Recorded as done elsewhere

- **x86_64 bundle** — built in a separate session. Not verified from this working tree;
  `docs/beta2-release-readiness.md` still lists it under hardware-gated items and should be
  reconciled against that build rather than against this note.
