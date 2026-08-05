%define debug_package %{nil}

# Release carries a build timestamp so successive builds of the same Version get
# distinct NEVRAs. Without it every build was pfm-client-<version>-1, `dnf
# upgrade` had nothing to compare, and the agent installed on every monitored
# host could never be updated in place. (Same defect fixed in pfm-server.spec;
# that one is stamped with the source commit too, which this package cannot do
# because it is built from a staged tarball rather than the monorepo checkout.)
%define build_timestamp %(date -u +"%y%m%d%H%M")

Name:           pfm-client
Summary:        Postgres1st Monitoring and Management Client (pfm-agent)
Version:        %{version}
Release:        %{release}.%{build_timestamp}%{?dist}
Group:          Applications/Databases
License:        ASL 2.0
Vendor:         Postgres1st
URL:            https://postgresfirst.com
Source:         pfm-client-%{version}.tar.gz
BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root

BuildRequires:    systemd
BuildRequires:    pkgconfig(systemd)
%if 0%{?rhel} && 0%{?rhel} >= 9
Requires:         perl-interpreter
%endif
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

AutoReq:        no

%description
Postgres1st Monitoring and Management (PFMM) is an open-source platform for managing and monitoring
PostgreSQL performance.
PFMM is a free and open-source solution that you can run in your own environment for maximum security and
reliability. It provides thorough time-based analysis for PostgreSQL servers to ensure that your data works
as efficiently as possible.


%prep
%setup -q

%build

%install
install -m 0755 -d $RPM_BUILD_ROOT/usr/sbin
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/bin
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/tools
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/config
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/high-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/high-resolution

install -m 0755 bin/pfm-admin $RPM_BUILD_ROOT/opt/postgres1st/pfm/bin
install -m 0755 bin/pfm-agent $RPM_BUILD_ROOT/opt/postgres1st/pfm/bin
install -m 0755 bin/pfm-agent-entrypoint $RPM_BUILD_ROOT/opt/postgres1st/pfm/bin
install -m 0755 bin/node_exporter $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 bin/postgres_exporter $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 bin/rds_exporter $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 bin/azure_exporter $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 bin/vmagent $RPM_BUILD_ROOT/opt/postgres1st/pfm/exporters
install -m 0755 bin/pt-summary $RPM_BUILD_ROOT/opt/postgres1st/pfm/tools
install -m 0755 bin/pt-pg-summary $RPM_BUILD_ROOT/opt/postgres1st/pfm/tools
install -m 0755 bin/nomad $RPM_BUILD_ROOT/opt/postgres1st/pfm/tools
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/low-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/medium-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/textfile-collector/high-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/low-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-postgres-uptime.yml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-hr.yml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-mr.yaml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 queries-lr.yaml $RPM_BUILD_ROOT/opt/postgres1st/pfm/collectors/custom-queries/postgresql/low-resolution/
install -m 0755 -d $RPM_BUILD_ROOT/%{_unitdir}
install -m 0644 config/pfm-agent.service %{buildroot}/%{_unitdir}/pfm-agent.service


%clean
rm -rf $RPM_BUILD_ROOT

%pre
if [ $1 -eq 1 ]; then
  if ! getent passwd pfm-agent > /dev/null 2>&1; then
    /usr/sbin/groupadd -r pfm-agent
    /usr/sbin/useradd -M -r -g pfm-agent -d /opt/postgres1st/ -s /bin/false -c pfm-agent pfm-agent > /dev/null 2>&1
  fi
fi
if [ $1 -eq 2 ]; then
    /usr/bin/systemctl stop pfm-agent.service >/dev/null 2>&1 ||:
fi

%post
for file in pfm-admin pfm-agent
do
  %{__ln_s} -f /opt/postgres1st/pfm/bin/$file /usr/bin/$file
  %{__ln_s} -f /opt/postgres1st/pfm/bin/$file /usr/sbin/$file
done
%systemd_post pfm-agent.service
if [ $1 -eq 1 ]; then
    if [ ! -f /opt/postgres1st/pfm/config/pfm-agent.yaml ]; then
        install -d -m 0755 /opt/postgres1st/pfm/config
        install -m 0660 -o pfm-agent -g pfm-agent /dev/null /opt/postgres1st/pfm/config/pfm-agent.yaml
    fi
    /usr/bin/systemctl enable pfm-agent >/dev/null 2>&1 || :
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pfm-agent.service
fi

if [ $1 -eq 2 ]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pfm-agent.service
fi

%preun
%systemd_preun pfm-agent.service

if [ -f /opt/postgres1st/pfm/config/pfm-agent.yaml.new ]; then
    rm -f /opt/postgres1st/pfm/config/pfm-agent.yaml.new
fi

%postun
case "$1" in
   1) # This is a dnf upgrade.
      %systemd_postun_with_restart pfm-agent.service
   ;;
esac
if [ $1 -eq 0 ]; then
  %systemd_postun_with_restart pfm-agent.service
  if /usr/bin/id -g pfm-agent > /dev/null 2>&1; then
    /usr/sbin/userdel pfm-agent > /dev/null 2>&1
    /usr/sbin/groupdel pfm-agent > /dev/null 2>&1 || true
    if [ -f /opt/postgres1st/pfm/config/pfm-agent.yaml ]; then
        rm -r /opt/postgres1st/pfm/config/pfm-agent.yaml
    fi
    if [ -f /opt/postgres1st/pfm/config/pfm-agent.yaml.bak ]; then
        rm -r /opt/postgres1st/pfm/config/pfm-agent.yaml.bak
    fi
    if [ -d /opt/postgres1st/pfm/config ] && [ -z "$(ls -A /opt/postgres1st/pfm/config)" ]; then
       rmdir /opt/postgres1st/pfm/config
    fi

    if [ -d /opt/postgres1st/pfm ] && [ -z "$(ls -A /opt/postgres1st/pfm)" ]; then
       rmdir /opt/postgres1st/pfm
    fi

    for file in pfm-admin pfm-agent
    do
      if [ -L /usr/sbin/$file ]; then
        rm -rf /usr/sbin/$file
      fi
      if [ -L /usr/bin/$file ]; then
        rm -rf /usr/bin/$file
      fi
    done
  fi
fi

%files
%config %{_unitdir}/pfm-agent.service
%attr(0660,pfm-agent,pfm-agent) %ghost /opt/postgres1st/pfm/config/pfm-agent.yaml
%attr(-,pfm-agent,pfm-agent) /opt/postgres1st/pfm

%changelog
* Wed May 21 2025 Talha Bin Rizwan <talha.rizwan@percona.com>
- PKG-521 include valkey_exporter into pmm client

* Fri Nov 8 2024 Nurlan Moldomurov <nurlan.moldomurov@percona.com>
- PMM-13399 include nomad into pmm client

* Tue Jun 21 2022 Nikita Beletskii <nikita.beletskii@percona.com>
- PMM-7 remove support for RHEL older then 7

* Tue Aug 24 2021 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-8618 ship default PG queries in PMM.

* Tue Oct 13 2020 Nikolay Khramchikhin <nik@victoriametrics.com>
- PMM-6396 added vmagent binary.

* Tue Aug 25 2020 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-2045 MySQL Group Replication Dashboard.

* Fri Jul 31 2020 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-5701 DB_Uptime in Home Dashboard shows wrong metric.

* Thu Aug 29 2019 Evgeniy Patlan <evgeniy.patlan@percona.com>
- Rework file structure.
