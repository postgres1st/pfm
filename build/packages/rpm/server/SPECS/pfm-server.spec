# pfm-server — native RHEL assembly of the pfm monitoring stack.
#
# Ships the systemd units, first-boot provisioning, runtime config and account
# that turn the individual component packages into a native (no-container)
# server, run as `systemctl start pfm.target` under journald as the non-root
# `pfm` account. This is pfm's value-add: upstream PMM ships only a container,
# so there is no server meta-package to base this on.
#
# Service/binary references remain pmm-* (ExecStart=/usr/sbin/pmm-managed, etc.)
# until the rebrand workstream renames the binaries.
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

# the line below is sed'ed by the build script to set a correct version
%define full_pfm_version 3.0.0

Name:           pfm-server
Version:        %{full_pfm_version}
Release:        1%{?dist}
Summary:        Native systemd assembly of the pfm (PMM-derived) monitoring stack

License:        AGPLv3
URL:            https://github.com/postgres1st/pfm
Source0:        %{repo}-%{shortcommit}.tar.gz
BuildArch:      noarch

BuildRequires:  systemd-rpm-macros

Requires(pre):    shadow-utils
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

# Component packages built from this monorepo (pmm-managed carries the systemd
# process-management backend). NOTE: names/versions to be pinned on first VM
# install against the actual built RPMs + the dep-availability grid.
Requires:       pmm-managed
Requires:       pmm-client
Requires:       percona-victoriametrics
Requires:       vmproxy
Requires:       percona-qan-api2
Requires:       percona-dashboards
Requires:       pfm-grafana

# Third-party data/proxy tier (Percona ppg / ClickHouse / distro repos).
# postgresql14: -server has initdb/pg_ctl; -contrib has pg_stat_statements
# (postgres won't start without it); the base package has psql (pfm-init.sh).
Requires:       percona-postgresql14-server
Requires:       percona-postgresql14-contrib
Requires:       percona-postgresql14
Requires:       clickhouse-server
Requires:       nginx
Requires:       openssl
# polkit evaluates the shipped rule that lets non-root pmm-managed drive systemctl
# against pfm-* units; without polkitd running, those calls fail "Access denied".
Requires:       polkit

%description
pfm-server assembles the pfm monitoring stack (PostgreSQL, ClickHouse,
VictoriaMetrics, vmalert, vmproxy, qan-api2, nginx, Grafana, pmm-managed and
pmm-agent) into a native systemd service set. It provides one unit per service
plus a pfm.target that starts the whole stack, idempotent first-boot
provisioning, zero-egress defaults, and a non-root pfm service account —
without a container.

%prep
%setup -q -n %{repo}-%{commit}

%build
# Nothing to compile; this package only ships config, units and scripts.

%install
install -d -p %{buildroot}%{_unitdir}
install -d -p %{buildroot}%{_tmpfilesdir}
install -d -p %{buildroot}%{_sysusersdir}
install -d -p %{buildroot}%{_datadir}/pfm
install -d -p %{buildroot}%{_prefix}/lib/pfm/defaults

cd build/packages/config/pfm

install -p -m 0644 pfm.target %{buildroot}%{_unitdir}/pfm.target
for u in pfm-init pfm-postgresql pfm-clickhouse pfm-nginx pfm-grafana \
         pfm-victoriametrics pfm-vmalert pfm-vmproxy pfm-qan-api2 \
         pfm-managed pfm-agent; do
    install -p -m 0644 ${u}.service %{buildroot}%{_unitdir}/${u}.service
done

install -p -m 0644 pfm-tmpfiles.conf  %{buildroot}%{_tmpfilesdir}/pfm.conf
install -p -m 0644 pfm-sysusers.conf  %{buildroot}%{_sysusersdir}/pfm.conf
install -p -m 0755 pfm-init.sh        %{buildroot}%{_datadir}/pfm/pfm-init.sh

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
%systemd_post pfm.target
if [ $1 -eq 1 ]; then
    # First install: materialize the account/runtime dirs and enable the target.
    # Not auto-started — the operator runs `systemctl start pfm.target` (or
    # reboots) so a long first-boot provision doesn't block the transaction.
    systemd-sysusers %{_sysusersdir}/pfm.conf >/dev/null 2>&1 || :
    install -d -m 0770 -o pfm -g pfm /srv || :
    systemd-tmpfiles --create %{_tmpfilesdir}/pfm.conf >/dev/null 2>&1 || :
    systemctl enable pfm.target >/dev/null 2>&1 || :
    # pmm-client's %post starts its own pmm-agent.service (different unit + user);
    # pfm uses pfm-agent.service, driven by pmm-managed. Stop and mask the client
    # unit so the two agents don't compete.
    systemctl disable --now pmm-agent.service >/dev/null 2>&1 || :
    systemctl mask pmm-agent.service >/dev/null 2>&1 || :
    echo "pfm-server installed. Start it with: systemctl start pfm.target"
fi

%preun
%systemd_preun pfm.target

%postun
%systemd_postun pfm.target
if [ $1 -eq 0 ]; then
    # Full removal (not upgrade, where $1 >= 1): undo the mask %post applied to
    # pmm-client's pmm-agent.service, otherwise it stays masked forever and
    # pmm-client can never run its own agent again.
    systemctl unmask pmm-agent.service >/dev/null 2>&1 || :
fi

%files
%{_unitdir}/pfm.target
%{_unitdir}/pfm-init.service
%{_unitdir}/pfm-postgresql.service
%{_unitdir}/pfm-clickhouse.service
%{_unitdir}/pfm-nginx.service
%{_unitdir}/pfm-grafana.service
%{_unitdir}/pfm-victoriametrics.service
%{_unitdir}/pfm-vmalert.service
%{_unitdir}/pfm-vmproxy.service
%{_unitdir}/pfm-qan-api2.service
%{_unitdir}/pfm-managed.service
%{_unitdir}/pfm-agent.service
%{_tmpfilesdir}/pfm.conf
%{_sysusersdir}/pfm.conf
%dir %{_datadir}/pfm
%attr(0755, root, root) %{_datadir}/pfm/pfm-init.sh
%dir %{_prefix}/lib/pfm
%dir %{_prefix}/lib/pfm/defaults
# Secret-bearing credential seeds: root-owned, group pfm, not world-readable.
%attr(0640, root, pfm) %{_prefix}/lib/pfm/defaults/*.env
# Native nginx config (/etc/nginx and conf.d are owned by the base nginx package,
# so they are not %dir'd here; only the ssl subdir is ours). Operator-editable
# config is %config(noreplace).
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
* Sat Jul 04 2026 Postgre First <asheshvashi@gmail.com> - 3.0.0-1
- Initial pfm-server assembly: native systemd unit set, first-boot provisioning,
  zero-egress defaults, and the pfm service account.
