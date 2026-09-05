%define debug_package %{nil}

# Release carries a build timestamp so successive builds of the same Version get
# distinct NEVRAs. Without it every build was pfw-client-<version>-1, `dnf
# upgrade` had nothing to compare, and the agent installed on every monitored
# host could never be updated in place. (Same defect fixed in pfw-server.spec;
# that one is stamped with the source commit too, which this package cannot do
# because it is built from a staged tarball rather than the monorepo checkout.)
%define build_timestamp %(date -u +"%y%m%d%H%M")

Name:           pfw-agent
Summary:        PGF WatchTower Agent (pfw-agent)
Version:        %{version}
Release:        %{release}.%{build_timestamp}%{?dist}
Group:          Applications/Databases
License:        ASL 2.0
Vendor:         Postgres1st
URL:            https://postgresfirst.com
Source:         pfw-client-%{version}.tar.gz
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
PGF WatchTower is an open-source platform for managing and monitoring
PostgreSQL performance.
PGF WatchTower is a free and open-source solution that you can run in your own environment for maximum security and
reliability. It provides thorough time-based analysis for PostgreSQL servers to ensure that your data works
as efficiently as possible.


%prep
# -n explicitly: the source tarball is staged as pfw-client-%{version} by
# build-pfw-airgap, which is the source layout and does not follow Name:.
%setup -q -n pfw-client-%{version}

%build

%install
install -m 0755 -d $RPM_BUILD_ROOT/usr/sbin
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/bin
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/tools
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/config
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/high-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/high-resolution

install -m 0755 bin/pfw-admin $RPM_BUILD_ROOT/opt/postgres1st/watchtower/bin
install -m 0755 bin/pfw-agent $RPM_BUILD_ROOT/opt/postgres1st/watchtower/bin
install -m 0755 bin/pfw-agent-entrypoint $RPM_BUILD_ROOT/opt/postgres1st/watchtower/bin
install -m 0755 bin/node_exporter $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 bin/postgres_exporter $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 bin/rds_exporter $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 bin/azure_exporter $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 bin/vmagent $RPM_BUILD_ROOT/opt/postgres1st/watchtower/exporters
install -m 0755 bin/pt-summary $RPM_BUILD_ROOT/opt/postgres1st/watchtower/tools
install -m 0755 bin/pt-pg-summary $RPM_BUILD_ROOT/opt/postgres1st/watchtower/tools
install -m 0755 bin/nomad $RPM_BUILD_ROOT/opt/postgres1st/watchtower/tools
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/low-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/medium-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/textfile-collector/high-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/low-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-postgres-uptime.yml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-hr.yml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-mr.yaml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 queries-lr.yaml $RPM_BUILD_ROOT/opt/postgres1st/watchtower/collectors/custom-queries/postgresql/low-resolution/
install -m 0755 -d $RPM_BUILD_ROOT/%{_unitdir}
install -m 0644 config/pfw-agent.service %{buildroot}/%{_unitdir}/pfw-agent.service


%clean
rm -rf $RPM_BUILD_ROOT

%pre
if [ $1 -eq 1 ]; then
  if ! getent passwd pfw-agent > /dev/null 2>&1; then
    /usr/sbin/groupadd -r pfw-agent
    /usr/sbin/useradd -M -r -g pfw-agent -d /opt/postgres1st/ -s /bin/false -c pfw-agent pfw-agent > /dev/null 2>&1
  fi
fi
if [ $1 -eq 2 ]; then
    /usr/bin/systemctl stop pfw-agent.service >/dev/null 2>&1 ||:
fi

%post
for file in pfw-admin pfw-agent
do
  %{__ln_s} -f /opt/postgres1st/watchtower/bin/$file /usr/bin/$file
  %{__ln_s} -f /opt/postgres1st/watchtower/bin/$file /usr/sbin/$file
done
%systemd_post pfw-agent.service
if [ $1 -eq 1 ]; then
    if [ ! -f /opt/postgres1st/watchtower/config/pfw-agent.yaml ]; then
        install -d -m 0755 /opt/postgres1st/watchtower/config
        install -m 0660 -o pfw-agent -g pfw-agent /dev/null /opt/postgres1st/watchtower/config/pfw-agent.yaml
    fi
    /usr/bin/systemctl enable pfw-agent >/dev/null 2>&1 || :
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pfw-agent.service
fi

if [ $1 -eq 2 ]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pfw-agent.service
fi

%preun
%systemd_preun pfw-agent.service

if [ -f /opt/postgres1st/watchtower/config/pfw-agent.yaml.new ]; then
    rm -f /opt/postgres1st/watchtower/config/pfw-agent.yaml.new
fi

%postun
case "$1" in
   1) # This is a dnf upgrade.
      %systemd_postun_with_restart pfw-agent.service
   ;;
esac
if [ $1 -eq 0 ]; then
  %systemd_postun_with_restart pfw-agent.service
  if /usr/bin/id -g pfw-agent > /dev/null 2>&1; then
    /usr/sbin/userdel pfw-agent > /dev/null 2>&1
    /usr/sbin/groupdel pfw-agent > /dev/null 2>&1 || true
    if [ -f /opt/postgres1st/watchtower/config/pfw-agent.yaml ]; then
        rm -r /opt/postgres1st/watchtower/config/pfw-agent.yaml
    fi
    if [ -f /opt/postgres1st/watchtower/config/pfw-agent.yaml.bak ]; then
        rm -r /opt/postgres1st/watchtower/config/pfw-agent.yaml.bak
    fi
    if [ -d /opt/postgres1st/watchtower/config ] && [ -z "$(ls -A /opt/postgres1st/watchtower/config)" ]; then
       rmdir /opt/postgres1st/watchtower/config
    fi

    if [ -d /opt/postgres1st/watchtower ] && [ -z "$(ls -A /opt/postgres1st/watchtower)" ]; then
       rmdir /opt/postgres1st/watchtower
    fi

    for file in pfw-admin pfw-agent
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
%config %{_unitdir}/pfw-agent.service
%attr(0660,pfw-agent,pfw-agent) %ghost /opt/postgres1st/watchtower/config/pfw-agent.yaml
# Shared with the server packages now that both live under one root, so claim only
# the subtrees this package installs. A recursive claim on the root would take
# ownership of the server's advisors/, checks/ and dashboards/ as well.
%dir %attr(0755, root, root) /opt/postgres1st
%dir %attr(0755, root, root) /opt/postgres1st/watchtower
%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/watchtower/bin
%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/watchtower/tools
%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/watchtower/exporters
%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/watchtower/config
%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/watchtower/collectors

%changelog
* Thu Sep 03 2026 Postgre First <asheshvashi@gmail.com> - 3.9.0-1
- First PGF WatchTower build of this package, from source.
