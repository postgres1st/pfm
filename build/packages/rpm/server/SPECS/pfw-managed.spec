%global debug_package   %{nil}
%undefine _missing_build_ids_terminate_build
%global _dwz_low_mem_die_limit 0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          8f3d007617941033867aea6a134c48b39142427f
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         20
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:		pfw-managed
Version:	%{version}
Release:	%{rpm_release}
Summary:	PGF WatchTower management daemon

License:	AGPLv3
URL:		  https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
pfw-managed manages configuration of PGF WatchTower server components (VictoriaMetrics,
Grafana, etc.) and exposes an API for that. Those APIs are used by the pfw-admin tool.
See https://github.com/postgres1st/pfm for more information.


%prep
%setup -q -n pmm-%{commit}
mkdir -p src/github.com/percona
ln -s $(pwd) src/%{provider}


%build

export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/github.com/percona/pmm/managed
make release

cd ../ui
make release

%install
install -d -p %{buildroot}%{_bindir}
install -d -p %{buildroot}%{_sbindir}
# Pinned, not %{name}: nginx serves swagger from /usr/share/pfw-managed.
install -d -p %{buildroot}%{_datadir}/pfw-managed
install -d -p %{buildroot}%{_datadir}/pfw-ui
install -d -p %{buildroot}/opt/postgres1st/watchtower/dashboards/panels/pmm-compat-app
install -d -p %{buildroot}/opt/postgres1st/watchtower/{advisors,checks,alerting-templates}

install -p -m 0755 bin/pfw-managed %{buildroot}%{_sbindir}/pfw-managed
install -p -m 0755 bin/pfw-encryption-rotation %{buildroot}%{_sbindir}/pfw-encryption-rotation
install -p -m 0755 bin/pfw-managed-init %{buildroot}%{_sbindir}/pfw-managed-init
install -p -m 0755 bin/pfw-managed-starlark %{buildroot}%{_sbindir}/pfw-managed-starlark


cd src/github.com/percona/pmm
cp -pa ./api/swagger %{buildroot}%{_datadir}/pfw-managed
cp -pa ./ui/apps/pmm/dist/. %{buildroot}%{_datadir}/pfw-ui
cp -pa ./ui/apps/pmm-compat/dist/. %{buildroot}/opt/postgres1st/watchtower/dashboards/panels/pmm-compat-app
cp -pa ./managed/data/advisors/*.yml %{buildroot}/opt/postgres1st/watchtower/advisors/
cp -pa ./managed/data/checks/*.yml %{buildroot}/opt/postgres1st/watchtower/checks/
cp -pa ./managed/data/alerting-templates/*.yml %{buildroot}/opt/postgres1st/watchtower/alerting-templates/

%pre
# Create the pfw system account before files are laid down so the %attr
# ownership below resolves at unpack. pmm-managed is a dependency of
# pfw-server, so it is unpacked BEFORE pfw-server's own %pre runs; without
# this the pfw-owned paths would silently fall back to root.
getent group pfw >/dev/null || groupadd -r pfw
getent passwd pfw >/dev/null || \
    useradd -r -g pfw -d /srv -s /usr/sbin/nologin -c "pfw monitoring service" pfw
exit 0

%files
%license src/%{provider}/LICENSE
%doc src/%{provider}/README.md
%{_sbindir}/pfw-managed
%{_sbindir}/pfw-encryption-rotation
%{_sbindir}/pfw-managed-init
%{_sbindir}/pfw-managed-starlark
%{_datadir}/pfw-managed
%attr(-, pfw, root) %{_datadir}/pfw-ui
%dir %attr(0755, root, root) /opt/postgres1st
%dir %attr(0755, root, root) /opt/postgres1st/watchtower
%attr(-, pfw, root) /opt/postgres1st/watchtower/dashboards/panels/pmm-compat-app
%dir %attr(0755, pfw, pfw) /opt/postgres1st/watchtower/advisors
%dir %attr(0755, pfw, pfw) /opt/postgres1st/watchtower/checks
%dir %attr(0755, pfw, pfw) /opt/postgres1st/watchtower/alerting-templates
%attr(0644, pfw, root) /opt/postgres1st/watchtower/advisors/*.yml
%attr(0644, pfw, root) /opt/postgres1st/watchtower/checks/*.yml
%attr(0644, pfw, root) /opt/postgres1st/watchtower/alerting-templates/*.yml

%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- First PGF WatchTower build of this package, from source.
