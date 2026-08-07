# pfm-server — native RHEL assembly of the pfm monitoring stack.
#
# Ships the systemd units, first-boot provisioning, runtime config and account
# that turn the individual component packages into a native (no-container)
# server, run as `systemctl start pfm.target` under journald as the non-root
# `pfm` account. This is pfm's value-add: upstream PMM ships only a container,
# so there is no server meta-package to base this on.
#
# Binaries/packages are pfm-* (ExecStart=/usr/sbin/pfm-managed). The client owns
# the pfm-agent.service name, so this package's self-monitoring unit is
# pfm-server-agent.service; %post masks the client's unit so the two agents don't
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

# Tracks the upstream PMM release this assembly is built from, so pfm-server,
# pfm-managed and pfm-client all read the same version. It was 3.0.0 -- an
# independent number chosen when this spec was written -- which made the
# metapackage the customer actually installs look OLDER than the 3.9.0 components
# inside it. 3.9.0 > 3.0.0, so this is a normal upgrade for anyone already on the
# old number and needs no Epoch.
# `~beta1` (tilde, not hyphen): rpm reads it as a pre-release, so this sorts BELOW a
# plain 3.9.0 and upgrades to the release cleanly. Keep in step with PFMM_PMM_VERSION in
# build/scripts/pfmm-airgap-vars, which carries the same value for the sibling packages.
%define full_pfm_version 3.9.0~beta1

# Release carries a build timestamp and the source commit, matching the sibling
# specs (pfm-managed, percona-dashboards, grafana). Without it every build
# produced the identical NEVRA 3.0.0-1, so `dnf upgrade` had nothing to compare
# and silently skipped the package -- and this is the package that owns the
# systemd units, pfm-init.sh, the grafana/nginx config and the polkit rule, so
# none of those could ever be updated on an existing install without a manual
# `dnf reinstall`.
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:           pfm-server
Version:        %{full_pfm_version}
Release:        %{rpm_release}
Summary:        Native systemd assembly of the pfm (PMM-derived) monitoring stack

License:        AGPLv3
URL:            https://github.com/postgres1st/pfm
Source0:        %{repo}-%{shortcommit}.tar.gz
BuildArch:      noarch

BuildRequires:  systemd-rpm-macros
# Builds pfm_nginx.pp. No matching runtime Requires: semodule lives in
# policycoreutils, which a real RHEL host always has but our test containers do not
# (verified: no semodule, no policy store). The scriptlets below are guarded so a
# host without SELinux installs cleanly and simply runs unconfined, rather than the
# transaction failing on a machine that never needed the policy.
BuildRequires:  selinux-policy-devel

Requires(pre):    shadow-utils
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

# Component packages built from this monorepo (pmm-managed carries the systemd
# process-management backend). NOTE: names/versions to be pinned on first VM
# install against the actual built RPMs + the dep-availability grid.
Requires:       pfm-managed
Requires:       pfm-client
Requires:       percona-victoriametrics
Requires:       vmproxy
Requires:       percona-qan-api2
Requires:       percona-dashboards
Requires:       pfm-grafana

# Third-party data/proxy tier (PGDG PostgreSQL / ClickHouse / distro repos).
# postgresql18: -server has initdb/pg_ctl; -contrib has pg_stat_statements
# (postgres won't start without it); the base package has psql (pfm-init.sh).
Requires:       postgresql18-server
Requires:       postgresql18-contrib
Requires:       postgresql18
Requires:       clickhouse-server
Requires:       nginx
Requires:       openssl
# polkit evaluates the shipped rule that lets non-root pmm-managed drive systemctl
# against pfm-* units; without polkitd running, those calls fail "Access denied".
Requires:       polkit

%description
pfm-server assembles the pfm monitoring stack (PostgreSQL, ClickHouse,
VictoriaMetrics, vmalert, vmproxy, qan-api2, nginx, Grafana, pfm-managed and
pfm-agent) into a native systemd service set. It provides one unit per service
plus a pfm.target that starts the whole stack, idempotent first-boot
provisioning, zero-egress defaults, and a non-root pfm service account —
without a container.

%prep
%setup -q -n %{repo}-%{commit}

%build
# The SELinux module is the only thing compiled here; everything else is config,
# units and scripts. See build/packages/selinux/pfm_nginx.te for why we ship policy
# at all -- in short, /srv is site-specific so no vendor can label it for us.
(cd build/packages/selinux && make -f %{_datadir}/selinux/devel/Makefile pfm_nginx.pp)

%install
install -d -p %{buildroot}%{_datadir}/selinux/packages
install -p -m 0644 build/packages/selinux/pfm_nginx.pp \
    %{buildroot}%{_datadir}/selinux/packages/pfm_nginx.pp
install -d -p %{buildroot}%{_unitdir}
install -d -p %{buildroot}%{_tmpfilesdir}
install -d -p %{buildroot}%{_sysusersdir}
install -d -p %{buildroot}%{_datadir}/pfm
install -d -p %{buildroot}%{_prefix}/lib/pfm/defaults

cd build/packages/config/pfm

install -p -m 0644 pfm.target %{buildroot}%{_unitdir}/pfm.target
for u in pfm-init pfm-postgresql pfm-clickhouse pfm-nginx pfm-grafana \
         pfm-victoriametrics pfm-vmalert pfm-vmproxy pfm-qan-api2 \
         pfm-managed pfm-server-agent pfm-clickhouse-perms; do
    install -p -m 0644 ${u}.service %{buildroot}%{_unitdir}/${u}.service
done

install -p -m 0644 pfm-tmpfiles.conf  %{buildroot}%{_tmpfilesdir}/pfm.conf
install -p -m 0644 pfm-sysusers.conf  %{buildroot}%{_sysusersdir}/pfm.conf
install -p -m 0755 pfm-init.sh        %{buildroot}%{_datadir}/pfm/pfm-init.sh

# ClickHouse custom config (data under /srv/clickhouse to match pfm-clickhouse's
# ReadWritePaths). Staged here; %post deploys it to /etc/clickhouse-server and
# symlinks config.xml/users.xml -> default-*. Single source of truth with the
# container image build (build/ansible/roles/clickhouse/files).
install -d -p %{buildroot}%{_datadir}/pfm/clickhouse
for f in default-config.xml default-users.xml low-memory-config.xml \
         low-memory-users.xml dhparam.pem switch-config.sh; do
    install -p -m 0644 ../../../ansible/roles/clickhouse/files/${f} \
        %{buildroot}%{_datadir}/pfm/clickhouse/${f}
done

# Grafana config (postgres backend + /srv/grafana paths). The percona-grafana RPM
# ships a stock grafana.ini (sqlite under the homepath, unwritable by the pfm
# user); %post deploys this one over it. Single source of truth with the image
# build (build/ansible/roles/grafana/files).
install -d -p %{buildroot}%{_datadir}/pfm/grafana
# Ships as /etc/grafana/pfm.ini and is selected via `grafana server --config`
# (same pattern as nginx/pfm.conf above). pfm-grafana owns grafana.ini, so a
# copy written over it is reverted the next time that package is upgraded --
# which silently returns grafana to its stock sqlite-in-homepath config and the
# server then fails to start ("mkdir /usr/share/grafana/data: read-only").
install -d -p %{buildroot}%{_sysconfdir}/grafana
install -p -m 0644 ../../../ansible/roles/grafana/files/grafana.ini \
    %{buildroot}%{_sysconfdir}/grafana/pfm.ini

# Grafana provisioning: datasources (VictoriaMetrics/ClickHouse/PTSummary),
# dashboards (from the percona-dashboards pmm-app plugin), plugins. Without these
# grafana boots but has no datasources/dashboards (empty UI). Grafana reads these
# in place -- pfm.ini sets paths.provisioning here -- so there is no copy into
# grafana's own conf dir to go stale or be reverted by a pfm-grafana upgrade.
for p in datasources dashboards plugins; do
    install -d -p %{buildroot}%{_datadir}/pfm/grafana/provisioning/${p}
done
install -p -m 0644 ../../../ansible/roles/grafana/files/datasources.yml \
    %{buildroot}%{_datadir}/pfm/grafana/provisioning/datasources/default.yml
install -p -m 0644 ../../../ansible/roles/grafana/files/dashboards.yml \
    %{buildroot}%{_datadir}/pfm/grafana/provisioning/dashboards/default.yml
install -p -m 0644 ../../../ansible/roles/grafana/files/plugins.yml \
    %{buildroot}%{_datadir}/pfm/grafana/provisioning/plugins/default.yml

for e in victoriametrics vmalert vmproxy qan-api2 grafana; do
    # 0640 root:pfm — these seeds carry default DB/ClickHouse credentials; only
    # systemd-tmpfiles (root) needs to read them to seed the 0600 /run copies, so
    # they must not be world-readable. (Ownership is enforced by %attr in %files.)
    install -p -m 0640 defaults/${e}.env %{buildroot}%{_prefix}/lib/pfm/defaults/${e}.env
done

# Native nginx config + TLS sources. pfm-nginx.service reads /etc/nginx/nginx.conf
# and pfm-init.sh's generate_nginx_cert requires the SSL sources at /etc/nginx/ssl.
# These are the pfm-adapted copies (non-root pfm user, native paths, zero egress) —
# NOT the container image's versions.
install -d -p %{buildroot}%{_sysconfdir}/nginx/conf.d
install -d -p %{buildroot}%{_sysconfdir}/nginx/ssl
# Ships as pfm.conf (the base nginx package owns nginx.conf); selected via `nginx -c`.
install -p -m 0644 nginx/nginx.conf      %{buildroot}%{_sysconfdir}/nginx/pfm.conf
install -p -m 0644 nginx/local-rss.xml   %{buildroot}%{_sysconfdir}/nginx/local-rss.xml
install -p -m 0644 nginx/conf.d/pmm.conf     %{buildroot}%{_sysconfdir}/nginx/conf.d/pmm.conf
install -p -m 0644 nginx/conf.d/pmm-ssl.conf %{buildroot}%{_sysconfdir}/nginx/conf.d/pmm-ssl.conf
install -p -m 0644 nginx/ssl/dhparam.pem      %{buildroot}%{_sysconfdir}/nginx/ssl/dhparam.pem
install -p -m 0644 nginx/ssl/ca-certs.pem     %{buildroot}%{_sysconfdir}/nginx/ssl/ca-certs.pem
install -p -m 0644 nginx/ssl/certificate.conf %{buildroot}%{_sysconfdir}/nginx/ssl/certificate.conf

# polkit rule: lets the non-root pfm account drive systemctl for pfm-* units.
install -d -p %{buildroot}%{_datadir}/polkit-1/rules.d
install -p -m 0644 pfm-polkit.rules %{buildroot}%{_datadir}/polkit-1/rules.d/49-pfm.rules

%pre
# Create the pfm system account before files are laid down (mirrors the
# declarative pfm-sysusers.conf; kept here so %attr ownership resolves at unpack
# and for EL versions without a sysusers file trigger).
getent group pfm >/dev/null || groupadd -r pfm
getent passwd pfm >/dev/null || \
    useradd -r -g pfm -d /srv -s /usr/sbin/nologin -c "pfm monitoring service" pfm
exit 0

%post
# SELinux policy for our /srv layout, installed on every transaction so an upgraded
# module actually takes effect. Guarded and never fatal: a host with no SELinux
# tooling (our Rocky 9 test containers have neither semodule nor a policy store)
# must install cleanly and run unconfined rather than fail the transaction.
# -n defers the policy reload so we reload once, explicitly, only when enabled.
if [ -x %{_sbindir}/semodule ]; then
    %{_sbindir}/semodule -n -i %{_datadir}/selinux/packages/pfm_nginx.pp >/dev/null 2>&1 || :
    if %{_sbindir}/selinuxenabled 2>/dev/null; then
        %{_sbindir}/load_policy >/dev/null 2>&1 || :
        # /srv is created by this scriptlet below and populated later by pfm-init,
        # so relabel what exists now; pfm-init restorecons what it creates itself.
        restorecon -R /srv >/dev/null 2>&1 || :
    fi
fi
%systemd_post pfm.target
if [ $1 -eq 1 ]; then
    # First install: materialize the account/runtime dirs and enable the target.
    # Not auto-started — the operator runs `systemctl start pfm.target` (or
    # reboots) so a long first-boot provision doesn't block the transaction.
    systemd-sysusers %{_sysusersdir}/pfm.conf >/dev/null 2>&1 || :
    install -d -m 0770 -o pfm -g pfm /srv || :
    systemd-tmpfiles --create %{_tmpfilesdir}/pfm.conf >/dev/null 2>&1 || :
    systemctl enable pfm.target >/dev/null 2>&1 || :
    # pfm-client's %post starts its own pfm-agent.service (different unit + user);
    # the server uses pfm-server-agent.service, driven by pfm-managed. Stop and mask
    # unit so the two agents don't compete.
    systemctl disable --now pfm-agent.service >/dev/null 2>&1 || :
    systemctl mask pfm-agent.service >/dev/null 2>&1 || :

    # First-boot fixes that need root — pfm-init.sh runs as the pfm user and is
    # sandboxed to /srv, so it cannot touch these root-owned /etc paths.

    # (1) pmm-managed's unit binds /etc/victoriametrics-promscrape.yml in
    # ReadWritePaths (mandatory). The file doesn't exist until pmm-managed writes
    # it, but pmm-managed (pfm user) can't create files in root-owned /etc, so the
    # namespace fails to set up (exit 226) and pmm-managed can never start to
    # generate it. Seed an empty pfm-owned file here to break the chicken-and-egg;
    # pmm-managed overwrites it with the real scrape config on first run.
    if [ ! -e %{_sysconfdir}/victoriametrics-promscrape.yml ]; then
        install -m 0644 -o pfm -g pfm /dev/null %{_sysconfdir}/victoriametrics-promscrape.yml || :
    fi

    # (2) Deploy the pfm ClickHouse config. The stock clickhouse-server RPM ships a
    # config.xml with data under /var/lib/clickhouse, but pfm-clickhouse.service
    # runs as pfm and its ReadWritePaths only expose /srv/clickhouse — so the stock
    # config can't write its data dir. Install our custom config (data in
    # /srv/clickhouse) exactly as Percona's image does: drop the files in, replace
    # config.xml/users.xml with symlinks to the default-* variants, hand the tree
    # to pfm (stock perms are 0700 clickhouse:clickhouse — unreadable by pfm).
    if [ -d %{_datadir}/pfm/clickhouse ]; then
        for f in default-config.xml default-users.xml low-memory-config.xml \
                 low-memory-users.xml dhparam.pem switch-config.sh; do
            [ -e %{_datadir}/pfm/clickhouse/${f} ] && \
                install -m 0644 %{_datadir}/pfm/clickhouse/${f} %{_sysconfdir}/clickhouse-server/${f} || :
        done
        chmod 0755 %{_sysconfdir}/clickhouse-server/switch-config.sh 2>/dev/null || :
        # config.xml/users.xml -> default-* symlinks (remove stock regular files first)
        [ -L %{_sysconfdir}/clickhouse-server/config.xml ] || rm -f %{_sysconfdir}/clickhouse-server/config.xml
        [ -L %{_sysconfdir}/clickhouse-server/users.xml ]  || rm -f %{_sysconfdir}/clickhouse-server/users.xml
        ln -sf default-config.xml %{_sysconfdir}/clickhouse-server/config.xml || :
        ln -sf default-users.xml  %{_sysconfdir}/clickhouse-server/users.xml  || :
    fi
    for d in %{_sysconfdir}/clickhouse-server /var/lib/clickhouse /var/log/clickhouse-server; do
        [ -d "$d" ] && chown -R pfm:pfm "$d" && chmod -R u+rwX "$d" || :
    done

    # (3) Nothing to deploy for grafana any more, deliberately. Both its config
    # and its provisioning are now read from pfm-owned paths that this package
    # ships directly (/etc/grafana/pfm.ini via --config, and
    # %{_datadir}/pfm/grafana/provisioning via the ini's paths.provisioning), and
    # plugins are read from the package that owns them. Copying into
    # percona-grafana's own directories is what made upgrades of that package
    # revert our settings, and copying dashboards into /srv on first boot is what
    # made dashboard updates never reach a running server.

    echo "pfm-server installed. Start it with: systemctl start pfm.target"
fi

%preun
%systemd_preun pfm.target

%postun
%systemd_postun pfm.target
if [ $1 -eq 0 ]; then
    # Full removal (not upgrade, where $1 >= 1): undo the mask %post applied to
    # pfm-client's pfm-agent.service, otherwise it stays masked forever and
    # pfm-client can never run its own agent again.
    systemctl unmask pfm-agent.service >/dev/null 2>&1 || :
    # Same condition: only on full removal. Removing the module during an upgrade
    # would leave the host unconfined between %postun and the new %post.
    if [ -x %{_sbindir}/semodule ]; then
        %{_sbindir}/semodule -n -r pfm_nginx >/dev/null 2>&1 || :
        %{_sbindir}/selinuxenabled 2>/dev/null && %{_sbindir}/load_policy >/dev/null 2>&1 || :
    fi
fi

%files
%{_unitdir}/pfm.target
%{_unitdir}/pfm-init.service
%{_unitdir}/pfm-postgresql.service
%{_unitdir}/pfm-clickhouse.service
%{_unitdir}/pfm-clickhouse-perms.service
%{_unitdir}/pfm-nginx.service
%{_unitdir}/pfm-grafana.service
%{_unitdir}/pfm-victoriametrics.service
%{_unitdir}/pfm-vmalert.service
%{_unitdir}/pfm-vmproxy.service
%{_unitdir}/pfm-qan-api2.service
%{_unitdir}/pfm-managed.service
%{_unitdir}/pfm-server-agent.service
%{_tmpfilesdir}/pfm.conf
%{_sysusersdir}/pfm.conf
%dir %{_datadir}/pfm
# Not %dir on selinux/packages: /usr/share/selinux/packages is owned by
# selinux-policy, which is present wherever the module is usable. Claiming it here
# would conflict.
%{_datadir}/selinux/packages/pfm_nginx.pp
%attr(0755, root, root) %{_datadir}/pfm/pfm-init.sh
%{_datadir}/pfm/clickhouse
%{_datadir}/pfm/grafana
%dir %{_prefix}/lib/pfm
%dir %{_prefix}/lib/pfm/defaults
# Secret-bearing credential seeds: root-owned, group pfm, not world-readable.
%attr(0640, root, pfm) %{_prefix}/lib/pfm/defaults/*.env
# Native nginx config (/etc/nginx and conf.d are owned by the base nginx package,
# so they are not %dir'd here; only the ssl subdir is ours). Operator-editable
# config is %config(noreplace).
# Our grafana config, read via `grafana server --config` from pfm-grafana.service.
# Deliberately NOT /etc/grafana/grafana.ini: that path belongs to pfm-grafana, and
# writing over it means every upgrade of that package silently reverts us to the
# stock sqlite-in-homepath config. /etc/grafana itself is pfm-grafana's, so only
# the file is listed here.
# 0640 root:pfm — carries the grafana DB password, so it must not be
# world-readable; grafana runs as pfm and reads it via group.
%config(noreplace) %attr(0640, root, pfm) %{_sysconfdir}/grafana/pfm.ini
%config(noreplace) %{_sysconfdir}/nginx/pfm.conf
%config(noreplace) %{_sysconfdir}/nginx/conf.d/pmm.conf
%config(noreplace) %{_sysconfdir}/nginx/conf.d/pmm-ssl.conf
%{_sysconfdir}/nginx/local-rss.xml
%dir %{_sysconfdir}/nginx/ssl
%{_sysconfdir}/nginx/ssl/dhparam.pem
%{_sysconfdir}/nginx/ssl/ca-certs.pem
%{_sysconfdir}/nginx/ssl/certificate.conf
%{_datadir}/polkit-1/rules.d/49-pfm.rules

%changelog
* Wed Aug 05 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- Track the upstream PMM version so pfm-server matches pfm-managed/pfm-client
  instead of carrying an independent 3.0.0.

* Sat Jul 04 2026 Postgre First <asheshvashi@gmail.com> - 3.0.0-1
- Initial pfm-server assembly: native systemd unit set, first-boot provisioning,
  zero-egress defaults, and the pfm service account.
