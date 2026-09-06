# pfw-server — native RHEL assembly of the pfw monitoring stack.
#
# Ships the systemd units, first-boot provisioning, runtime config and account
# that turn the individual component packages into a native (no-container)
# server, run as `systemctl start pfw.target` under journald as the non-root
# `pfw` account. This is pfw's value-add: upstream PMM ships only a container,
# so there is no server meta-package to base this on.
#
# Binaries/packages are pfw-* (ExecStart=/usr/sbin/pfw-managed). The client owns
# the pfw-agent.service name, so this package's self-monitoring unit is
# pfw-server-agent.service; %post masks the client's unit so the two agents don't
# compete. Names kept as pmm-* on purpose: the pmm_managed Prometheus metric
# namespace (dashboards query it), the pmm-managed Postgres role/DB, and the
# PMM_* environment-variable contract.
#
# NOT yet validated on a real host (this was authored off-VM): the exact
# Requires names/versions, the %post enable/tmpfiles/sysusers sequence, and the
# SELinux file contexts (T6) all need a first install on Rocky/RHEL 9. Marked
# inline where relevant.

%global commit      0000000000000000000000000000000000000000
%global shortcommit %(c=%{commit}; echo ${c:0:7})
# Built from the monorepo tarball, which the build script archives with prefix
# "<repo>-<commit>/" where repo_name=pmm — must match the %setup -n below.
%global repo        pmm

# Tracks the upstream PMM release this assembly is built from, so pfw-server,
# pfw-managed and pfw-client all read the same version. It was 3.0.0 -- an
# independent number chosen when this spec was written -- which made the
# metapackage the customer actually installs look OLDER than the 3.9.0 components
# inside it. 3.9.0 > 3.0.0, so this is a normal upgrade for anyone already on the
# old number and needs no Epoch.
# `~beta1` (tilde, not hyphen): rpm reads it as a pre-release, so this sorts BELOW a
# plain 3.9.0 and upgrades to the release cleanly. Keep in step with PFW_VERSION in
# build/scripts/pfw-airgap-vars, which carries the same value for the sibling packages.
%define full_pfw_version 3.9.0~beta2

# Release carries a build timestamp and the source commit, matching the sibling
# specs (pfw-managed, percona-dashboards, grafana). Without it every build
# produced the identical NEVRA 3.0.0-1, so `dnf upgrade` had nothing to compare
# and silently skipped the package -- and this is the package that owns the
# systemd units, pfw-init.sh, the grafana/nginx config and the polkit rule, so
# none of those could ever be updated on an existing install without a manual
# `dnf reinstall`.
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:           pfw-server
Version:        %{full_pfw_version}
Release:        %{rpm_release}
Summary:        Native systemd assembly of the Postgres1st WatchTower monitoring stack

License:        AGPLv3
URL:            https://github.com/postgres1st/pfm
Source0:        %{repo}-%{shortcommit}.tar.gz
BuildArch:      noarch

BuildRequires:  systemd-rpm-macros
# No BuildRequires for the SELinux module: build-pfw-airgap compiles pfw_nginx.pp in
# the builder image and stages the result, so rpmbuild only packages it.
#
# No runtime Requires either. semodule lives in policycoreutils, which a real RHEL
# host always has but our test containers do not (verified: no semodule, no policy
# store, no selinux-policy-targeted). The scriptlets below are guarded so such a host
# installs cleanly and runs unconfined, rather than failing the transaction on a
# machine that never needed the policy.

Requires(pre):    shadow-utils
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

# Component packages built from this monorepo (pmm-managed carries the systemd
# process-management backend). NOTE: names/versions to be pinned on first VM
# install against the actual built RPMs + the dep-availability grid.
Requires:       pfw-managed
Requires:       pfw-agent
Requires:       pfw-victoriametrics
Requires:       pfw-vmproxy
Requires:       pfw-qan-api2
Requires:       pfw-dashboards
Requires:       pfw-grafana

# Third-party data/proxy tier (PGDG PostgreSQL / ClickHouse / distro repos).
# postgresql18: -server has initdb/pg_ctl; -contrib has pg_stat_statements
# (postgres won't start without it); the base package has psql (pfw-init.sh).
Requires:       postgresql18-server
Requires:       postgresql18-contrib
Requires:       postgresql18
Requires:       clickhouse-server
Requires:       nginx
Requires:       openssl
# polkit evaluates the shipped rule that lets non-root pmm-managed drive systemctl
# against pfw-* units; without polkitd running, those calls fail "Access denied".
Requires:       polkit

%description
pfw-server assembles the Postgres1st WatchTower monitoring stack (PostgreSQL, ClickHouse,
VictoriaMetrics, vmalert, vmproxy, qan-api2, nginx, Grafana, pfw-managed and
pfw-agent) into a native systemd service set. It provides one unit per service
plus a pfw.target that starts the whole stack, idempotent first-boot
provisioning, zero-egress defaults, and a non-root pfw service account —
without a container.

%prep
%setup -q -n %{repo}-%{commit}

%build
# Nothing to compile; this package only ships config, units and scripts.
# pfw_nginx.pp included: build-pfw-airgap compiles it and stages the result, because
# that script rewrites %build to a no-op for every spec. Do not add a compile step
# here -- it would be silently discarded.

%install
install -d -p %{buildroot}%{_datadir}/selinux/packages
install -p -m 0644 build/packages/selinux/pfw_nginx.pp \
    %{buildroot}%{_datadir}/selinux/packages/pfw_nginx.pp
install -d -p %{buildroot}%{_unitdir}
install -d -p %{buildroot}%{_tmpfilesdir}
install -d -p %{buildroot}%{_sysusersdir}
install -d -p %{buildroot}%{_datadir}/pfw
install -d -p %{buildroot}%{_prefix}/lib/pfw/defaults

cd build/packages/config/pfw

install -p -m 0644 pfw.target %{buildroot}%{_unitdir}/pfw.target
for u in pfw-init pfw-postgresql pfw-clickhouse pfw-nginx pfw-grafana \
         pfw-victoriametrics pfw-vmalert pfw-vmproxy pfw-qan-api2 \
         pfw-managed pfw-server-agent pfw-clickhouse-perms; do
    install -p -m 0644 ${u}.service %{buildroot}%{_unitdir}/${u}.service
done


install -p -m 0644 pfw-tmpfiles.conf  %{buildroot}%{_tmpfilesdir}/pfw.conf
install -p -m 0644 pfw-sysusers.conf  %{buildroot}%{_sysusersdir}/pfw.conf
install -p -m 0755 pfw-init.sh        %{buildroot}%{_datadir}/pfw/pfw-init.sh
install -p -m 0755 pfw-secrets-perms.sh        %{buildroot}%{_datadir}/pfw/pfw-secrets-perms.sh

# ClickHouse custom config (data under /srv/clickhouse to match pfw-clickhouse's
# ReadWritePaths). Staged here; %post deploys it to /etc/clickhouse-server and
# symlinks config.xml/users.xml -> default-*. Single source of truth with the
# container image build (build/ansible/roles/clickhouse/files).
install -d -p %{buildroot}%{_datadir}/pfw/clickhouse
for f in default-config.xml default-users.xml low-memory-config.xml \
         low-memory-users.xml dhparam.pem switch-config.sh; do
    install -p -m 0644 ../../../ansible/roles/clickhouse/files/${f} \
        %{buildroot}%{_datadir}/pfw/clickhouse/${f}
done

# Grafana config (postgres backend + /srv/grafana paths). The percona-grafana RPM
# ships a stock grafana.ini (sqlite under the homepath, unwritable by the pfw
# user); %post deploys this one over it. Single source of truth with the image
# build (build/ansible/roles/grafana/files).
install -d -p %{buildroot}%{_datadir}/pfw/grafana
# Ships as /etc/grafana/pfw.ini and is selected via `grafana server --config`
# (same pattern as nginx/pfw.conf above). pfw-grafana owns grafana.ini, so a
# copy written over it is reverted the next time that package is upgraded --
# which silently returns grafana to its stock sqlite-in-homepath config and the
# server then fails to start ("mkdir /usr/share/grafana/data: read-only").
install -d -p %{buildroot}%{_sysconfdir}/grafana
install -p -m 0644 ../../../ansible/roles/grafana/files/grafana.ini \
    %{buildroot}%{_sysconfdir}/grafana/pfw.ini
# Blank the [database] password in the RPM's copy only. The shared source keeps
# `password = grafana` because the container image needs it: ansible's
# initialization role creates the container's grafana role with that literal and
# supervisord's [program:grafana] exports no GF_DATABASE_PASSWORD, so emptying it
# at the source would leave container Grafana unable to reach its database. The
# RPM has no such need -- pfw-init.sh generates the password per install and
# pfw-grafana.service supplies it as GF_DATABASE_PASSWORD -- so shipping the
# constant here would only put a working credential in every copy of the package.
# Fail the build if the line is not there: a silent no-op would ship the constant.
# Explicit `|| exit 1` rather than relying on errexit, which rpm does not set for
# %%install in every version. (%% escapes it: rpm expands macros inside comments, so
# a comment line beginning with a bare section macro is parsed as that section.)
grep -q '^password = grafana$' %{buildroot}%{_sysconfdir}/grafana/pfw.ini \
    || { echo "grafana.ini no longer carries 'password = grafana'; the RPM would ship a credential" >&2; exit 1; }

# The ClickHouse credential is replaced at install time by substituting over this
# exact hash (sha256("clickhouse")) in %post. If upstream ever changes the shipped
# value, every one of those sed expressions silently becomes a no-op and each install
# ships the published constant with nothing to notice. Fail the build instead: a
# broken build is recoverable, a silently insecure package is not.
for f in default-users.xml low-memory-users.xml; do
    grep -q '7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3' \
        ../../../ansible/roles/clickhouse/files/${f} \
        || { echo "${f} no longer carries sha256(\"clickhouse\"); the %post substitution would be a silent no-op and the shipped credential would survive" >&2; exit 1; }
done
sed -i 's/^password = grafana$/password =/' %{buildroot}%{_sysconfdir}/grafana/pfw.ini \
    || exit 1

# Grafana provisioning: datasources (VictoriaMetrics/ClickHouse/PTSummary),
# dashboards (from the percona-dashboards pmm-app plugin), plugins. Without these
# grafana boots but has no datasources/dashboards (empty UI). Grafana reads these
# in place -- pfw.ini sets paths.provisioning here -- so there is no copy into
# grafana's own conf dir to go stale or be reverted by a pfw-grafana upgrade.
for p in datasources dashboards plugins; do
    install -d -p %{buildroot}%{_datadir}/pfw/grafana/provisioning/${p}
done
install -p -m 0644 ../../../ansible/roles/grafana/files/datasources.yml \
    %{buildroot}%{_datadir}/pfw/grafana/provisioning/datasources/default.yml
install -p -m 0644 ../../../ansible/roles/grafana/files/dashboards.yml \
    %{buildroot}%{_datadir}/pfw/grafana/provisioning/dashboards/default.yml
install -p -m 0644 ../../../ansible/roles/grafana/files/plugins.yml \
    %{buildroot}%{_datadir}/pfw/grafana/provisioning/plugins/default.yml

for e in victoriametrics vmalert vmproxy qan-api2 grafana; do
    # 0640 root:pfw — these seeds carry default DB/ClickHouse credentials; only
    # systemd-tmpfiles (root) needs to read them to seed the 0600 /run copies, so
    # they must not be world-readable. (Ownership is enforced by %attr in %files.)
    install -p -m 0640 defaults/${e}.env %{buildroot}%{_prefix}/lib/pfw/defaults/${e}.env
done

# Native nginx config + TLS sources. pfw-nginx.service reads /etc/nginx/nginx.conf
# and pfw-init.sh's generate_nginx_cert requires the SSL sources at /etc/nginx/ssl.
# These are the pfw-adapted copies (non-root pfw user, native paths, zero egress) —
# NOT the container image's versions.
install -d -p %{buildroot}%{_sysconfdir}/nginx/conf.d
install -d -p %{buildroot}%{_sysconfdir}/nginx/ssl
# Ships as pfw.conf (the base nginx package owns nginx.conf); selected via `nginx -c`.
install -p -m 0644 nginx/nginx.conf      %{buildroot}%{_sysconfdir}/nginx/pfw.conf
install -p -m 0644 nginx/conf.d/pfw.conf     %{buildroot}%{_sysconfdir}/nginx/conf.d/pfw.conf
install -p -m 0644 nginx/conf.d/pfw-ssl.conf %{buildroot}%{_sysconfdir}/nginx/conf.d/pfw-ssl.conf
install -p -m 0644 nginx/ssl/dhparam.pem      %{buildroot}%{_sysconfdir}/nginx/ssl/dhparam.pem
install -p -m 0644 nginx/ssl/ca-certs.pem     %{buildroot}%{_sysconfdir}/nginx/ssl/ca-certs.pem
install -p -m 0644 nginx/ssl/certificate.conf %{buildroot}%{_sysconfdir}/nginx/ssl/certificate.conf

# polkit rule: lets the non-root pfw account drive systemctl for pfw-* units.
install -d -p %{buildroot}%{_datadir}/polkit-1/rules.d
install -p -m 0644 pfw-polkit.rules %{buildroot}%{_datadir}/polkit-1/rules.d/49-pfw.rules

%pre
# Create the pfw system account before files are laid down (mirrors the
# declarative pfw-sysusers.conf; kept here so %attr ownership resolves at unpack
# and for EL versions without a sysusers file trigger).
getent group pfw >/dev/null || groupadd -r pfw
getent passwd pfw >/dev/null || \
    useradd -r -g pfw -d /srv -s /usr/sbin/nologin -c "pfw monitoring service" pfw
exit 0

%post
# Credential relocation for hosts provisioned before credentials were generated.
# This CANNOT live in pfw-init.sh alone: that unit is a RemainAfterExit oneshot and
# the spec uses %%systemd_postun (line 372), not %%systemd_postun_with_restart, so an
# upgrade only does a daemon-reload. pfw-init stays "active (exited)" and does not
# re-run, Requires= does not re-trigger an already-active unit, and pfw-init has no
# PartOf=pfw.target to be swept up by a target restart. An operator running
# `dnf upgrade && systemctl restart pfw-managed` would hit a non-optional
# EnvironmentFile that nothing had created, and the service would refuse to start.
#
# Upgrades only ($1 -ge 2). On a fresh install pfw-init.sh must GENERATE a random
# value; writing the historical constant here would silently defeat that. The two
# legacy literals below are also duplicated in pfw-init.sh (LEGACY_MANAGED_DB_PASSWORD
# / LEGACY_GRAFANA_DB_PASSWORD): %post runs as root during the transaction and
# cannot source a script the same transaction is installing.
#
# Write-then-rename, mirroring pfw-init.sh's own write_secret(): a transaction
# killed between the echo and the chown/chmod must never leave a permanent,
# wrong-owner file at the final path -- both this block and ensure_secrets() guard
# on the final path's existence alone, so a half-finished file there is never
# repaired. Writing under a .tmp name and renaming into place only after ownership
# and mode are correct means a crash mid-write leaves no file at the final path,
# so the next %post or the next boot's pfw-init retries cleanly instead of wedging.
#
# No `|| :` on the write/chown/chmod/mv steps below (unlike the rest of this
# scriptlet): a failure here must be visible as an rpm scriptlet warning rather
# than silently reporting a successful transaction while pfw-managed is left
# unable to start -- rpm does not abort or roll back the transaction over this,
# it only surfaces the failure.
#
# FIRST in %post, ahead of the SELinux section: that section's `restorecon -R /srv`
# exists to label what this scriptlet creates, and it only labels what already exists
# when it runs. With this block below it, /srv/.pfw-secrets and the two files in it
# were created after the only relabel in the transaction and left unlabelled.
#
# /srv/pfw-secrets-generated (written by pfw-init.sh, deliberately visible and outside
# the hidden secrets directory) records that this host's credentials were GENERATED and
# exist nowhere else. "Upgrading and no secret file" then no longer implies "predates the
# secret store" -- it can equally mean the store was lost, and /srv/.pfw-secrets is easy
# to lose: 0700 and hidden, so `cp -r /srv/*` and tar/rsync without '.[!.]*' skip it.
# Writing the shipped constant in that case would not just break the connection; pfw-managed's
# initWithRoot ALTERs the role to whatever it is handed, rotating a generated credential
# back to the public constant. Warn and leave it to the operator instead.
if [ $1 -ge 2 ] && { [ -f /srv/pfw-distribution ] || [ -f /srv/pmm-distribution ]; }; then
    if [ -f /srv/pfw-secrets-generated ] &&
       { [ ! -f /srv/.pfw-secrets/managed-db.env ] || [ ! -f /srv/.pfw-secrets/grafana.env ]; }; then
        echo "pfw-server: /srv/pfw-secrets-generated says this host's bootstrap credentials were" >&2
        echo "pfw-server: generated, but /srv/.pfw-secrets is missing or incomplete. NOT writing the" >&2
        echo "pfw-server: shipped constant over a generated password. Restore /srv/.pfw-secrets from" >&2
        echo "pfw-server: backup, or reset the PostgreSQL roles by hand and write the new values back." >&2
    else
        install -d -m 0700 -o pfw -g pfw /srv/.pfw-secrets || :
        if [ ! -f /srv/.pfw-secrets/managed-db.env ]; then
            ( umask 077; echo 'PMM_POSTGRES_DBPASSWORD=pmm-managed' > /srv/.pfw-secrets/managed-db.env.tmp )
            chown pfw:pfw /srv/.pfw-secrets/managed-db.env.tmp
            chmod 0600 /srv/.pfw-secrets/managed-db.env.tmp
            mv -f /srv/.pfw-secrets/managed-db.env.tmp /srv/.pfw-secrets/managed-db.env
        fi
        if [ ! -f /srv/.pfw-secrets/grafana.env ]; then
            ( umask 077; echo 'GF_DATABASE_PASSWORD=grafana' > /srv/.pfw-secrets/grafana.env.tmp )
            chown pfw:pfw /srv/.pfw-secrets/grafana.env.tmp
            chmod 0600 /srv/.pfw-secrets/grafana.env.tmp
            mv -f /srv/.pfw-secrets/grafana.env.tmp /srv/.pfw-secrets/grafana.env
        fi
    fi

    # ClickHouse relocation, deliberately OUTSIDE the PostgreSQL store-integrity
    # branch above. Its correctness does not depend on that store: the deployed
    # users XML says which credential the server accepts, so this can be decided on
    # evidence. Nesting it meant a host that lost /srv/.pfw-secrets took the warn
    # branch and never got clickhouse.env, and the operator who restored the two
    # PostgreSQL files then hit a second failure on a file the message never named.
    #
    # Only write the constant when the XML still carries sha256("clickhouse"). A host
    # whose password was already generated has a different hash, and writing the
    # constant would leave pfw-managed and qan-api2 unable to authenticate; that case
    # is left for pfw-init.sh, which fails loudly. A scriptlet must not abort an
    # upgrade transaction, so nothing here is fatal.
    #
    # This must happen in %post and not only in pfw-init.sh: three units now read the
    # file as a non-optional EnvironmentFile, and an upgrade does not re-run pfw-init
    # (RemainAfterExit oneshot, %%systemd_postun restarts nothing), so
    # `dnf upgrade && systemctl restart pfw-managed` would otherwise fail.
    if [ ! -f /srv/.pfw-secrets/clickhouse.env ] &&
       grep -q 7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3 \
           %{_sysconfdir}/clickhouse-server/default-users.xml 2>/dev/null; then
        install -d -m 0700 -o pfw -g pfw /srv/.pfw-secrets || :
        ( umask 077; echo 'PMM_CLICKHOUSE_PASSWORD=clickhouse' > /srv/.pfw-secrets/clickhouse.env.tmp )
        chown pfw:pfw /srv/.pfw-secrets/clickhouse.env.tmp
        chmod 0600 /srv/.pfw-secrets/clickhouse.env.tmp
        mv -f /srv/.pfw-secrets/clickhouse.env.tmp /srv/.pfw-secrets/clickhouse.env
    fi
fi

# SELinux policy for our /srv layout, installed on every transaction so an upgraded
# module actually takes effect. Guarded and never fatal: a host with no SELinux
# tooling (our Rocky 9 test containers have neither semodule nor a policy store)
# must install cleanly and run unconfined rather than fail the transaction.
# -n defers the policy reload so we reload once, explicitly, only when enabled.
if [ -x %{_sbindir}/semodule ]; then
    %{_sbindir}/semodule -n -i %{_datadir}/selinux/packages/pfw_nginx.pp >/dev/null 2>&1 || :
    if %{_sbindir}/selinuxenabled 2>/dev/null; then
        %{_sbindir}/load_policy >/dev/null 2>&1 || :
        # /srv is created by this scriptlet (below on a fresh install, and the
        # credential relocation above may have added /srv/.pfw-secrets on an upgrade)
        # and populated later by pfw-init, so relabel what exists now; pfw-init
        # restorecons what it creates itself.
        restorecon -R /srv >/dev/null 2>&1 || :
    fi
fi
%systemd_post pfw.target

if [ $1 -eq 1 ]; then
    # First install: materialize the account/runtime dirs and enable the target.
    # Not auto-started — the operator runs `systemctl start pfw.target` (or
    # reboots) so a long first-boot provision doesn't block the transaction.
    systemd-sysusers %{_sysusersdir}/pfw.conf >/dev/null 2>&1 || :
    install -d -m 0770 -o pfw -g pfw /srv || :
    systemd-tmpfiles --create %{_tmpfilesdir}/pfw.conf >/dev/null 2>&1 || :
    systemctl enable pfw.target >/dev/null 2>&1 || :
    # pfw-client's %post starts its own pfw-agent.service (different unit + user);
    # the server uses pfw-server-agent.service, driven by pfw-managed. Stop and mask
    # unit so the two agents don't compete.
    systemctl disable --now pfw-agent.service >/dev/null 2>&1 || :
    systemctl mask pfw-agent.service >/dev/null 2>&1 || :

    # First-boot fixes that need root — pfw-init.sh runs as the pfw user and is
    # sandboxed to /srv, so it cannot touch these root-owned /etc paths.

    # (1) pmm-managed's unit binds /etc/victoriametrics-promscrape.yml in
    # ReadWritePaths (mandatory). The file doesn't exist until pmm-managed writes
    # it, but pmm-managed (pfw user) can't create files in root-owned /etc, so the
    # namespace fails to set up (exit 226) and pmm-managed can never start to
    # generate it. Seed an empty pfw-owned file here to break the chicken-and-egg;
    # pmm-managed overwrites it with the real scrape config on first run.
    if [ ! -e %{_sysconfdir}/victoriametrics-promscrape.yml ]; then
        install -m 0644 -o pfw -g pfw /dev/null %{_sysconfdir}/victoriametrics-promscrape.yml || :
    fi

    # (2) Deploy the pfw ClickHouse config. The stock clickhouse-server RPM ships a
    # config.xml with data under /var/lib/clickhouse, but pfw-clickhouse.service
    # runs as pfw and its ReadWritePaths only expose /srv/clickhouse — so the stock
    # config can't write its data dir. Install our custom config (data in
    # /srv/clickhouse) exactly as Percona's image does: drop the files in, replace
    # config.xml/users.xml with symlinks to the default-* variants, hand the tree
    # to pfw (stock perms are 0700 clickhouse:clickhouse — unreadable by pfw).
    if [ -d %{_datadir}/pfw/clickhouse ]; then
        for f in default-config.xml default-users.xml low-memory-config.xml \
                 low-memory-users.xml dhparam.pem switch-config.sh; do
            [ -e %{_datadir}/pfw/clickhouse/${f} ] && \
                install -m 0644 %{_datadir}/pfw/clickhouse/${f} %{_sysconfdir}/clickhouse-server/${f} || :
        done
        chmod 0755 %{_sysconfdir}/clickhouse-server/switch-config.sh 2>/dev/null || :
        # config.xml/users.xml -> default-* symlinks (remove stock regular files first)
        [ -L %{_sysconfdir}/clickhouse-server/config.xml ] || rm -f %{_sysconfdir}/clickhouse-server/config.xml
        [ -L %{_sysconfdir}/clickhouse-server/users.xml ]  || rm -f %{_sysconfdir}/clickhouse-server/users.xml
        ln -sf default-config.xml %{_sysconfdir}/clickhouse-server/config.xml || :
        ln -sf default-users.xml  %{_sysconfdir}/clickhouse-server/users.xml  || :

        # Generate this host's ClickHouse credential, replacing the shipped
        # constant. The packaged XMLs carry sha256("clickhouse") -- a value
        # published in this source tree -- and ClickHouse listens on loopback only,
        # so this is local disclosure of the same class as the pmm-managed one.
        #
        # It runs here rather than in pfw-init.sh because the users XML is
        # root-owned under /etc and pfw-init runs as the pfw user under
        # ProtectSystem=strict. It is inside the first-install branch because this
        # whole ClickHouse block is: an upgrade never re-copies these files, so the
        # substitution survives, and pfw-init.sh relocates the constant for hosts
        # that predate this release.
        #
        # Nothing here is suppressed with `|| :`, but be clear about what that buys:
        # rpm does not abort or roll back the transaction over a failing scriptlet, it
        # only surfaces the failure. So the real protection is the validation below,
        # not the exit status.
        #
        # An unvalidated empty ch_pw would be catastrophic rather than merely broken:
        # sha256("") is a well-known constant, so the deployed XML would accept an
        # EMPTY password from anything on loopback -- worse than the published
        # constant this replaces. Refuse to touch the XML unless the value is exactly
        # what openssl should have produced.
        ch_pw=$(openssl rand -hex 16 2>/dev/null) || ch_pw=""
        case "${ch_pw}" in
            [0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]\
[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]\
[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]\
[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) ;;
            *)
                echo "pfw-server: FAILED to generate a ClickHouse password (openssl)." >&2
                echo "pfw-server: leaving the shipped credential in place rather than" >&2
                echo "pfw-server: installing an empty one. Fix openssl and reinstall." >&2
                ch_pw=""
                ;;
        esac
        if [ -n "${ch_pw}" ]; then
        ch_hash=$(printf '%s' "${ch_pw}" | sha256sum | cut -d' ' -f1)
        # Both copies or neither. A half-applied substitution is the one outcome
        # ensure_clickhouse_secret cannot detect in the direction that matters: if
        # default-users.xml keeps the constant while low-memory-users.xml does not,
        # the evidence check sees the legacy hash, writes the constant, and the stack
        # comes up working AND insecure, with no signal at all.
        ch_substituted=0
        for f in default-users.xml low-memory-users.xml; do
            if [ -f %{_sysconfdir}/clickhouse-server/${f} ]; then
                sed -i "s/7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3/${ch_hash}/g" \
                    %{_sysconfdir}/clickhouse-server/${f}
                if grep -q "${ch_hash}" %{_sysconfdir}/clickhouse-server/${f} &&
                   ! grep -q 7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3 \
                       %{_sysconfdir}/clickhouse-server/${f}; then
                    ch_substituted=$((ch_substituted + 1))
                else
                    echo "pfw-server: FAILED to replace the ClickHouse credential in ${f}." >&2
                    ch_substituted=-99
                fi
            fi
        done
        if [ "${ch_substituted}" -lt 0 ]; then
            echo "pfw-server: the ClickHouse credential is now inconsistent between the two" >&2
            echo "pfw-server: users XMLs. Reinstall pfw-server, or set the same" >&2
            echo "pfw-server: password_sha256_hex in both files by hand and write the matching" >&2
            echo "pfw-server: plaintext into /srv/.pfw-secrets/clickhouse.env." >&2
        fi
        install -d -m 0700 -o pfw -g pfw /srv/.pfw-secrets
        ( umask 077; printf 'PMM_CLICKHOUSE_PASSWORD=%s\n' "${ch_pw}" \
            > /srv/.pfw-secrets/clickhouse.env.tmp )
        chown pfw:pfw /srv/.pfw-secrets/clickhouse.env.tmp
        chmod 0600 /srv/.pfw-secrets/clickhouse.env.tmp
        mv -f /srv/.pfw-secrets/clickhouse.env.tmp /srv/.pfw-secrets/clickhouse.env
        fi
        unset ch_pw ch_hash
    fi
    for d in %{_sysconfdir}/clickhouse-server /var/lib/clickhouse /var/log/clickhouse-server; do
        [ -d "$d" ] && chown -R pfw:pfw "$d" && chmod -R u+rwX "$d" || :
    done

    # (3) Nothing to deploy for grafana any more, deliberately. Both its config
    # and its provisioning are now read from pfw-owned paths that this package
    # ships directly (/etc/grafana/pfw.ini via --config, and
    # %{_datadir}/pfw/grafana/provisioning via the ini's paths.provisioning), and
    # plugins are read from the package that owns them. Copying into
    # percona-grafana's own directories is what made upgrades of that package
    # revert our settings, and copying dashboards into /srv on first boot is what
    # made dashboard updates never reach a running server.

    echo "pfw-server installed. Start it with: systemctl start pfw.target"
fi

%preun
%systemd_preun pfw.target

%postun
%systemd_postun pfw.target
if [ $1 -eq 0 ]; then
    # Full removal (not upgrade, where $1 >= 1): undo the mask %post applied to
    # pfw-client's pfw-agent.service, otherwise it stays masked forever and
    # pfw-client can never run its own agent again.
    systemctl unmask pfw-agent.service >/dev/null 2>&1 || :
    # Same condition: only on full removal. Removing the module during an upgrade
    # would leave the host unconfined between %postun and the new %post.
    if [ -x %{_sbindir}/semodule ]; then
        %{_sbindir}/semodule -n -r pfw_nginx >/dev/null 2>&1 || :
        %{_sbindir}/selinuxenabled 2>/dev/null && %{_sbindir}/load_policy >/dev/null 2>&1 || :
    fi
fi

%files
%{_unitdir}/pfw.target
%{_unitdir}/pfw-init.service
%{_unitdir}/pfw-postgresql.service
%{_unitdir}/pfw-clickhouse.service
%{_unitdir}/pfw-clickhouse-perms.service
%{_unitdir}/pfw-nginx.service
%{_unitdir}/pfw-grafana.service
%{_unitdir}/pfw-victoriametrics.service
%{_unitdir}/pfw-vmalert.service
%{_unitdir}/pfw-vmproxy.service
%{_unitdir}/pfw-qan-api2.service
%{_unitdir}/pfw-managed.service
%{_unitdir}/pfw-server-agent.service
%{_tmpfilesdir}/pfw.conf
%{_sysusersdir}/pfw.conf
%dir %{_datadir}/pfw
# Not %dir on selinux/packages: /usr/share/selinux/packages is owned by
# selinux-policy, which is present wherever the module is usable. Claiming it here
# would conflict.
%{_datadir}/selinux/packages/pfw_nginx.pp
%attr(0755, root, root) %{_datadir}/pfw/pfw-init.sh
%attr(0755, root, root) %{_datadir}/pfw/pfw-secrets-perms.sh
%{_datadir}/pfw/clickhouse
%{_datadir}/pfw/grafana
%dir %{_prefix}/lib/pfw
%dir %{_prefix}/lib/pfw/defaults
# Secret-bearing credential seeds: root-owned, group pfw, not world-readable.
%attr(0640, root, pfw) %{_prefix}/lib/pfw/defaults/*.env
# Native nginx config (/etc/nginx and conf.d are owned by the base nginx package,
# so they are not %dir'd here; only the ssl subdir is ours). Operator-editable
# config is %config(noreplace).
# Our grafana config, read via `grafana server --config` from pfw-grafana.service.
# Deliberately NOT /etc/grafana/grafana.ini: that path belongs to pfw-grafana, and
# writing over it means every upgrade of that package silently reverts us to the
# stock sqlite-in-homepath config. /etc/grafana itself is pfw-grafana's, so only
# the file is listed here.
# 0640 root:pfw -- no longer carries a password (that moved to
# /srv/.pfw-secrets/grafana.env), but it stays non-world-readable because it
# describes the whole server layout.
%config(noreplace) %attr(0640, root, pfw) %{_sysconfdir}/grafana/pfw.ini
%config(noreplace) %{_sysconfdir}/nginx/pfw.conf
%config(noreplace) %{_sysconfdir}/nginx/conf.d/pfw.conf
%config(noreplace) %{_sysconfdir}/nginx/conf.d/pfw-ssl.conf
%dir %{_sysconfdir}/nginx/ssl
%{_sysconfdir}/nginx/ssl/dhparam.pem
%{_sysconfdir}/nginx/ssl/ca-certs.pem
%{_sysconfdir}/nginx/ssl/certificate.conf
%{_datadir}/polkit-1/rules.d/49-pfw.rules

%changelog
* Wed Aug 05 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- Track the upstream PMM version so pfw-server matches pfw-managed/pfw-client
  instead of carrying an independent 3.0.0.

* Sat Jul 04 2026 Postgre First <asheshvashi@gmail.com> - 3.0.0-1
- Initial pfw-server assembly: native systemd unit set, first-boot provisioning,
  zero-egress defaults, and the pfw service account.
