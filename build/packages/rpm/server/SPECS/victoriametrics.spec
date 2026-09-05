%undefine _missing_build_ids_terminate_build

%global repo            VictoriaMetrics
%global provider        github.com/VictoriaMetrics/%{repo}
%global commit          pmm-6401-v1.147.0

Name:           pfw-victoriametrics
Version:        1.147.0
Release:        1%{?dist}
Summary:        PGF WatchTower metrics storage (VictoriaMetrics and vmalert)
License:        Apache-2.0
# %{provider} is the upstream source we build FROM; the package is ours.
URL:            https://github.com/postgres1st/pfm
Source0:        https://%{provider}/archive/%{commit}.tar.gz


%description
%{summary}


%prep
%setup -q -n %{repo}-%{commit}


%build
export PKG_TAG=%{commit}
export BUILDINFO_TAG=%{commit}
export USER=builder

make victoria-metrics-pure
make vmalert-pure


%install
install -D -p -m 0755 ./bin/victoria-metrics-pure %{buildroot}%{_sbindir}/pfw-victoriametrics
install -D -p -m 0755 ./bin/vmalert-pure %{buildroot}%{_sbindir}/pfw-vmalert


%files
%license LICENSE
%doc README.md
%{_sbindir}/pfw-victoriametrics
%{_sbindir}/pfw-vmalert


%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 1.147.0-1
- First PGF WatchTower build of this package, from source.
