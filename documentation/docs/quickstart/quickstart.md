# Get started with PFMM

To get up and running with Postgres1st (PFMM) in no time, install PFMM on Bare Metal/Virtual using the Easy-install script for Docker.

This is the simplest and most efficient way to install PFMM with Docker.

??? info "Alternative installation options"
     For alternative setups or if you're not using Docker, explore the additional installation options detailed in the **Setting up** chapter:

    - [Deploy on Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md)
    - [Deploy based on a Docker image](../install-pmm/install-pmm-server/deployment-options/docker/index.md)
    - [Deploy on Virtual Appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)
    - [Deploy on Kubernetes/OpenShift via Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md)
    - [Run a PFMM instance hosted at AWS Marketplace](../install-pmm/install-pmm-server/deployment-options/aws/deploy_aws.md)

#### Prerequisites

Before you start installing PFMM, verify that your system meets the compatibility requirements:

??? info "Verify system compatibility"
    - System: Linux-compatible system with `sudo` privileges or `root` access
    - Network: Internet connectivity to download PFMM components
    - Ports: Your system's firewall should allow TCP traffic on port `443`

## Install PFMM

The Easy-install script only runs on Linux-compatible systems. To use it, run the command with `sudo` privileges or as `root`:
{ .power-number }

1. Download and install PFMM using `cURL` or `wget`:

    === "cURL"

        ```sh
        curl -fsSL https://raw.githubusercontent.com/percona/pmm/refs/heads/main/get-pmm.sh | /bin/bash
        ```

    === "wget"

        ```sh
        wget -qO - https://raw.githubusercontent.com/percona/pmm/refs/heads/main/get-pmm.sh | /bin/bash    
        ```

2. After the installation is complete, log into PFMM with the default `admin:admin` credentials.

??? info "What's happening under the hood?"
     This script does the following:

     * Installs Docker if it is not installed on your system.
     * Stops and renames any currently running PFMM Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old `pmm-server` container is not a recoverable backup.
     * Pulls and runs the latest PFMM Docker image.

## Connect database

Once PFMM is set up, choose the database or the application that you want it to monitor:

=== ":simple-postgresql: PostgreSQL"

    To connect a PostgreSQL database: 
    { .power-number}

    1. Create a PMM-specific user for monitoring:
        
        ```
        CREATE USER pmm WITH SUPERUSER ENCRYPTED PASSWORD '<your_password>';
        ```

    2. Ensure that PFMM can log in locally as this user to the PostgreSQL instance. To enable this, edit the `pg_hba.conf` file. If  not already enabled by an existing rule, add:

        ```conf
        local   all             pmm                                md5
        # TYPE  DATABASE        USER        ADDRESS                METHOD
        ```

    3. Set up the `pg_stat_monitor` database extension and configure your database server accordingly. 
    
        If you need to use the `pg_stat_statements` extension instead, see [Adding a PostgreSQL database](../install-pmm/install-pmm-client/connect-database/postgresql.md) and the [`pg_stat_monitor` online documentation](https://docs.percona.com/pg-stat-monitor/configuration.html) for details about available parameters.

    4. Set or change the value for `shared_preload_library` in your `postgresql.conf` file:

        ```ini
        shared_preload_libraries = 'pg_stat_monitor'
        ```

    5. Set up configuration values in your `postgresql.conf` file:

        ```ini
        pg_stat_monitor.pgsm_query_max_len = 2048
        ```

    6. In a `psql` session, run the following command to create the view where you can access the collected statistics. We recommend that you create the extension for the `postgres` database so that you can receive access to statistics from each database.

        ```
        CREATE EXTENSION pg_stat_monitor;
        ```

    7. To optimize server-side resources, install PMM Client via Package Manager on the database node:  
        
        === ":material-debian: Debian-based"

            Install the following with `root` permission: 
            { .power-number} 

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"

            Install the following with `root` permission: 
            { .power-number}   

            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```
            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    8. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    9. Add the PostgreSQL database:

        ```sh 
        pmm-admin add postgresql --username=pmm --password=<your_password>
        ```
            
    For detailed instructions and advanced installation options, see [Adding a PostgreSQL database](../install-pmm/install-pmm-client/connect-database/postgresql.md).

=== ":material-shuffle: HAProxy"
    To connect an HAProxy service:
    { .power-number}

    1. [Set up an HAproxy instance](https://www.haproxy.com/blog/haproxy-exposes-a-prometheus-metrics-endpoint). 
    2. Add the instance to PFMM (default address is <http://localhost:8404/metrics>), and use the `haproxy` alias to enable HAProxy metrics monitoring.
    3. To optimize server-side resources, install PMM Client via Package Manager on the database node: 
        
        === ":material-debian: Debian-based"
            Install the following with `root` permission: 
            { .power-number} 
                                     
            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
                dpkg -i percona-release_latest.generic_all.deb
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                apt update
                apt install -y pmm-client
                ```

        === ":material-redhat: Red Hat-based"
            Install the following with `root` permission: 
            { .power-number} 
         
            1. Install [percona-release](https://docs.percona.com/percona-software-repositories/installing.html) tool.  If this is already installed, [update percona-release](https://docs.percona.com/percona-software-repositories/updating.html) to the latest version:

                ```sh
                yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
                ```

            2. Enable the PMM client repository:

                ```sh
                percona-release enable pmm3-client release
                ```
            3. Install the PMM Client package:

                ```sh
                yum install -y pmm-client
                ```

    4. Register PMM Client:
        
        ```sh
        pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
        ```

    5. Run the command below, specifying the `listen-port`` as the port number where HAProxy is running. (This flag is mandatory.)

        ```sh
        pmm-admin add haproxy --listen-port=8404
        ```

    For detailed instructions and more information on the command arguments, see the [HAProxy topic](../install-pmm/install-pmm-client/connect-database/haproxy.md).

## Check database monitoring results

After installing PFMM and connecting the database, go to the database's Instance Summary dashboard. This shows essential information about your database performance and an overview of your environment.

For more information, see [PFMM Dashboards](../use/dashboards-panels/index.md).

## Next steps

- [Configure PFMM via the interface](../configure-pmm/configure.md)
- [Manage users in PFMM](../admin/manage-users/index.md)
- [Set up roles and permissions](../admin/roles/index.md)
