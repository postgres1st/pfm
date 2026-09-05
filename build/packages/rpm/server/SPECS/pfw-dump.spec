%undefine _missing_build_ids_terminate_build

%global repo            pmm-dump
%global provider        github.com/percona/%{repo}
%global commit          8353b46afd09746c07a6d1c001dd1ef72e6c4761
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:		pfw-dump
Version:	0.8.0-ga
Release:	%{rpm_release}
Summary:	PGF WatchTower dump tool: exports and imports monitoring metrics and query analytics.

License:	AGPLv3
URL:		https://%{provider}
Source0:	https://%{provider}/archive/%{commit}.tar.gz

%description
%{summary}

%prep
%setup -q -n %{repo}-%{commit}

%build
make build BRANCH="main" COMMIT="%{shortcommit}" VERSION="%{version}"

%install
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 pmm-dump %{buildroot}%{_sbindir}/pfw-dump

%files
%license LICENSE
%doc README.md
%{_sbindir}/pfw-dump


%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 0.8.0-1
- First PGF WatchTower build of this package, from source.
