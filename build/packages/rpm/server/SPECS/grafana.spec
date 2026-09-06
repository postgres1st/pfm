%global debug_package   %{nil}
%global commit          95667356a928f669b147e0af5f5338f9b618e7b2
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         117
%define grafana_version 12.4.5
%define full_pmm_version 3.0.0
%define full_version    v%{grafana_version}-%{full_pmm_version}
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# Map RPM arch -> Go arch for locating Grafana's build output (bin/linux/<go_arch>/).
# Keeps amd64 building unchanged while adding aarch64 -> arm64.
%ifarch aarch64
%global go_arch arm64
%else
%global go_arch amd64
%endif

%if ! 0%{?gobuild:1}
%define gobuild(o:) go build -ldflags "${LDFLAGS:-} -B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \\n')" -a -v -x %{?**};
%endif

Name:           pfw-grafana
Version:        %{grafana_version}
Release:        %{rpm_release}
Summary:        Grafana is an open source, feature rich metrics dashboard and graph editor
License:        AGPLv3
# Fork lineage: grafana/grafana -> percona/grafana -> postgres1st/grafana.
# %{commit} must resolve in the postgres1st fork; bump it to the branding HEAD
# once the Postgres1st rebrand commits land on top of the percona/grafana base.
URL:            https://github.com/postgres1st/grafana
Source0:        https://github.com/postgres1st/grafana/archive/%{commit}.tar.gz
ExclusiveArch:  %{ix86} x86_64 %{arm} aarch64

BuildRequires: fontconfig

%description
Grafana is an open source, feature rich metrics dashboard and graph editor for
Graphite, InfluxDB & OpenTSDB.

%prep
%setup -q -n grafana-%{commit}
rm -rf Godeps
sudo npm install -g grunt-cli

%build
mkdir -p _build/src
export GOPATH="$(pwd)/_build"

make build-go

make deps-js
make build-js

%install
install -d -p %{buildroot}%{_datadir}/grafana
cp -rpav conf %{buildroot}%{_datadir}/grafana
cp -rpav public %{buildroot}%{_datadir}/grafana
cp -rpav tools %{buildroot}%{_datadir}/grafana

install -d -p %{buildroot}%{_sbindir}
cp bin/linux/%{go_arch}/grafana-server %{buildroot}%{_sbindir}/
cp bin/linux/%{go_arch}/grafana %{buildroot}%{_sbindir}/
install -d -p %{buildroot}%{_bindir}
cp bin/linux/%{go_arch}/grafana-cli %{buildroot}%{_bindir}/

install -d -p %{buildroot}%{_sysconfdir}/grafana
cp conf/sample.ini %{buildroot}%{_sysconfdir}/grafana/grafana.ini
mv conf/ldap.toml %{buildroot}%{_sysconfdir}/grafana/
install -d -p %{buildroot}%{_sharedstatedir}/grafana

%files
%defattr(-, pfw, root, -)
%{_datadir}/grafana
%doc CHANGELOG.md README.md
%license LICENSE
%attr(0755, pfw, root) %{_sbindir}/grafana
%attr(0755, pfw, root) %{_sbindir}/grafana-server
%attr(0755, pfw, root) %{_bindir}/grafana-cli
%{_sysconfdir}/grafana/grafana.ini
%{_sysconfdir}/grafana/ldap.toml
%dir %{_sharedstatedir}/grafana

%pre
getent group pfw >/dev/null || echo "Group pfw does not exist. Please create it manually."
getent passwd pfw >/dev/null || echo "User pfw does not exist. Please create it manually."
exit 0

%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 12.4.5-1
- First Postgres1st WatchTower build of this package, from source.
