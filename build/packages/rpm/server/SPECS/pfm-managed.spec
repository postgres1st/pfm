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

Name:		pfm-managed
Version:	%{version}
Release:	%{rpm_release}
Summary:	Postgres1st Monitoring and Management management daemon

License:	AGPLv3
URL:		  https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
pmm-managed manages configuration of PMM server components (VictoriaMetrics,
Grafana, etc.) and exposes API for that. Those APIs are used by pmm-admin tool.
See PMM docs for more information.


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
install -d -p %{buildroot}%{_datadir}/%{name}
install -d -p %{buildroot}%{_datadir}/pfm-ui
install -d -p %{buildroot}/opt/postgres1st/pfmm/dashboards/panels/pmm-compat-app
install -d -p %{buildroot}/opt/postgres1st/pfmm/{advisors,checks,alerting-templates}
install -p -m 0755 bin/pfm-managed %{buildroot}%{_sbindir}/pfm-managed
install -p -m 0755 bin/pfm-encryption-rotation %{buildroot}%{_sbindir}/pfm-encryption-rotation
install -p -m 0755 bin/pfm-managed-init %{buildroot}%{_sbindir}/pfm-managed-init
install -p -m 0755 bin/pfm-managed-starlark %{buildroot}%{_sbindir}/pfm-managed-starlark

cd src/github.com/percona/pmm
cp -pa ./api/swagger %{buildroot}%{_datadir}/%{name}
cp -pa ./ui/apps/pmm/dist/. %{buildroot}%{_datadir}/pfm-ui
cp -pa ./ui/apps/pmm-compat/dist/. %{buildroot}/opt/postgres1st/pfmm/dashboards/panels/pmm-compat-app
cp -pa ./managed/data/advisors/*.yml %{buildroot}/opt/postgres1st/pfmm/advisors/
cp -pa ./managed/data/checks/*.yml %{buildroot}/opt/postgres1st/pfmm/checks/
cp -pa ./managed/data/alerting-templates/*.yml %{buildroot}/opt/postgres1st/pfmm/alerting-templates/

%pre
# Create the pfm system account before files are laid down so the %attr
# ownership below resolves at unpack. pmm-managed is a dependency of
# pfm-server, so it is unpacked BEFORE pfm-server's own %pre runs; without
# this the pfm-owned paths would silently fall back to root.
getent group pfm >/dev/null || groupadd -r pfm
getent passwd pfm >/dev/null || \
    useradd -r -g pfm -d /srv -s /usr/sbin/nologin -c "pfm monitoring service" pfm
exit 0

%files
%license src/%{provider}/LICENSE
%doc src/%{provider}/README.md
%{_sbindir}/pfm-managed
%{_sbindir}/pfm-encryption-rotation
%{_sbindir}/pfm-managed-init
%{_sbindir}/pfm-managed-starlark
%{_datadir}/%{name}
%attr(-, pfm, root) %{_datadir}/pfm-ui
%attr(-, pfm, root) /opt/postgres1st/pfmm/dashboards/panels/pmm-compat-app
%dir %attr(0755, pfm, pfm) /opt/postgres1st/pfmm/advisors
%dir %attr(0755, pfm, pfm) /opt/postgres1st/pfmm/checks
%dir %attr(0755, pfm, pfm) /opt/postgres1st/pfmm/alerting-templates
%attr(0644, pfm, root) /opt/postgres1st/pfmm/advisors/*.yml
%attr(0644, pfm, root) /opt/postgres1st/pfmm/checks/*.yml
%attr(0644, pfm, root) /opt/postgres1st/pfmm/alerting-templates/*.yml

%changelog
* Thu Sep 4 2025 Michael Okoko <michael.okoko@percona.com> - 3.4.0-1
- PMM-14013 bundle alerting templates with PMM.

* Wed Jun 11 2025 Michael Okoko <michael.okoko@percona.com> - 3.4.0-1
- PMM-14009 bundle advisors with PMM.

* Thu Apr 24 2025 Matej Kubinec <matej.kubinec@ext.percona.com> - 3.2.0-1
- PMM-13722 add pmm compat plugin

* Mon Sep 23 2024 Jiri Ctvrtka <jiri.ctvrtka@ext.percona.com> - 3.0.0-1
- PMM-13132 add PMM encryption rotation tool

* Fri Mar 22 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 3.0.0-1
- PMM-11231 add pmm ui

* Thu Jul 28 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 2.30.0-1
- PMM-10036 migrate to monorepo

* Fri Jun 17 2022 Anton Bystrov <anton.bystrov@simbirsoft.com> - 2.0.0-17
- PMM-10206 merge pmm-managed to monorepo pmm

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-17
- PMM-5645 built using Golang 1.14

* Tue May 12 2020 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-16
- added pmm-managed-starlark

* Tue Feb 11 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-14
- added pmm-managed-init

* Thu Sep  5 2019 Viacheslav Sarzhan <slava.sarzhan@percona.com> - 2.0.0-10
- init version
