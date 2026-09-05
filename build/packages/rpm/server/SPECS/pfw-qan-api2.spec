# Go build id is not supported for now.
# https://github.com/rpm-software-management/rpm/issues/367
# https://bugzilla.redhat.com/show_bug.cgi?id=1295951
%undefine _missing_build_ids_terminate_build

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global import_path     %{provider}
# The commit hash gets sed'ed by build-server-rpm script to set a correct version
# see: https://github.com/percona/pmm/blob/main/build/scripts/build-server-rpm#L58
%global commit          0000000000000000000000000000000000000000
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         17
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 3.0.0

Name:           pfw-qan-api2
Version:        %{version}
Release:        %{rpm_release}
Summary:        Query Analytics API v2 for PGF WatchTower

License:        AGPLv3
# %{provider} stays github.com/percona/pmm: it is the Go import path this
# source unpacks under, not a homepage. The URL is ours.
URL:            https://github.com/postgres1st/pfm
Source0:        https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
Query Analytics (QAN) API v2 is part of PGF WatchTower.
See https://github.com/postgres1st/pfm for more information.


%prep
%setup -T -c -n %{repo}-%{version}
%setup -q -c -a 0 -n %{repo}-%{version}
mkdir -p src/github.com/percona
mv %{repo}-%{commit} src/%{provider}


%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/%{provider}/qan-api2
make release


%install

install -d -p %{buildroot}%{_sbindir}
# Installed path is pinned, NOT %{name}: the binary path is referenced by
# pfw-qan-api2.service, supervisord.go, test fixtures and the dev Makefiles.
# Renaming the package does not rename those; that is a separate change.
install -p -m 0755 src/%{provider}/bin/qan-api2 %{buildroot}%{_sbindir}/pfw-qan-api2


%files
%attr(0755, root, root) %{_sbindir}/pfw-qan-api2
%license src/%{provider}/qan-api2/LICENSE
%doc src/%{provider}/qan-api2/README.md

%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- First PGF WatchTower build of this package, from source.
