%global debug_package   %{nil}
%global __strip         /bin/true

%global repo		        pmm
%global provider	      github.com/percona/%{repo}
%global commit		      ad4af6808bcd361284e8eb8cd1f36b1e98e32bce
%global shortcommit	    %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         27
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

%define clickhouse_datasource_version 4.19.0
%define polystat_panel_version        2.1.16

%ifarch x86_64
%define plugin_platform linux_amd64
%else
%define plugin_platform linux_arm64
%endif

Name:		  pfw-dashboards
Version:	%{version}
Release:	%{rpm_release}
Summary:	PGF WatchTower dashboards for PostgreSQL monitoring

License:	AGPLv3
URL:		  https://%{provider}

BuildRequires:	nodejs
BuildRequires:	unzip
Requires:	pfw-grafana

Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz
Source1:	https://github.com/grafana/clickhouse-datasource/releases/download/v%{clickhouse_datasource_version}/grafana-clickhouse-datasource-%{clickhouse_datasource_version}.%{plugin_platform}.zip
Source2:	https://github.com/grafana/grafana-polystat-panel/releases/download/v%{polystat_panel_version}/grafana-polystat-panel-%{polystat_panel_version}.zip

%description
This package provides a set of PGF WatchTower dashboards for PostgreSQL and system monitoring
using VictoriaMetrics datasource.


%prep
%setup -q -n %{repo}-%{commit}


%build
node -v
npm version
make -C dashboards release


%install
install -d %{buildroot}/opt/postgres1st/watchtower/dashboards/panels/pmm-app

# cp -a ./dashboards/panels %{buildroot}/opt/postgres1st/watchtower/dashboards
cp -a ./dashboards/pmm-app/dist %{buildroot}/opt/postgres1st/watchtower/dashboards/panels/pmm-app
unzip -q %{SOURCE1} -d %{buildroot}/opt/postgres1st/watchtower/dashboards/panels
unzip -q %{SOURCE2} -d %{buildroot}/opt/postgres1st/watchtower/dashboards/panels
echo %{version} > %{buildroot}/opt/postgres1st/watchtower/dashboards/VERSION


%files
%license ./dashboards/LICENSE
%doc ./dashboards/README.md
%dir %attr(0755, root, root) /opt/postgres1st
%dir %attr(0755, root, root) /opt/postgres1st/watchtower
%attr(-,pfw,root) /opt/postgres1st/watchtower/dashboards


%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- First PGF WatchTower build of this package, from source.
