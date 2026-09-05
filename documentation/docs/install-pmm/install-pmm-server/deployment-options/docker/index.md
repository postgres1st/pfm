# Install PMM Server with Docker

Deploy PMM Server as a Docker container for a fast, flexible and isolated setup. 

While PMM Server runs independently, we highly recommend that you streamline [upgrades via the PGF WatchTower user interface](../../../../pmm-upgrade/ui_upgrade.md) by installing the [Watchtower updater](https://containrrr.dev/watchtower/) alongside PMM Server. 

!!! info "The Watchtower updater is a third-party tool"
    Watchtower is an independent open-source container updater
    ([containrrr.dev](https://containrrr.dev/watchtower/)); the image installed below,
    `percona/watchtower`, is a downstream fork of it. Neither is a Postgres1st component, and
    neither is related to any Postgres1st product with a similar name. The updater applies
    only to container deployments — native RPM installations upgrade with `dnf` and never
    use it.

With the Watchtower updater installed, you can easily update PMM Server directly from the **Upgrade** page or by clicking the **Upgrade Now** button on the **Home** dashboard.

## Prerequisites
Before installation, ensure you have:

- Docker version 17.03 or higher
- CPU with `x86-64-v2` support
- [Sufficient system resources](../../../plan-pmm-installation/hardware_and_system.md) (recommended: 2+ CPU cores, 4+ GB RAM, 100+ GB disk space)

### Watchtower updater security requirements

The Watchtower updater requires access to the Docker socket to monitor and update containers. Since the Docker socket provides root-level access to the host system, it's critical to limit the Watchtower updater's exposure to prevent potential security vulnerabilities.

To ensure a secure setup when using the Watchtower updater:
 
 - limit the Watchtower updater's access to Docker network or localhost to prevent unauthorized external connections. See [Container network isolation guide](https://docs.docker.com/network/drivers/bridge/#use-user-defined-bridge-networks).
 - configure network to ensure only PMM Server is exposed externally. See [Docker networking best practices](https://docs.docker.com/network/bridge/#manage-a-user-defined-bridge).
 - secure Docker socket access for the Watchtower updater. See [Docker socket security](https://docs.docker.com/engine/security/security/#docker-daemon-attack-surface).
 - place both the Watchtower updater and PMM Server on the same Docker network. See [Watchtower updater network configuration](https://containrrr.dev/watchtower/usage-overview/#docker_host).

## Installation options

### Container setup summary

??? info "Container setup at a glance"
    - **Pull the Docker image**: `docker pull percona/pmm-server:3`
    - **Choose storage**: Docker volumes (recommended) or host directory
    - **Run the container**: Using the appropriate `docker run` command
    - **Access the UI**: Navigate to `https://SERVER_IP_ADDRESS` in your browser
    - **Log in**: Default credentials `admin` / `admin`

### Install PMM Server + Watchtower updater

You can install PMM Server with the Watchtower updater using one of two methods:


=== "Easy-install script (Recommended for simplicity)"

    The [Easy-install script](../docker/easy-install.md) simplifies setup by including Watchtower updater commands, enabling a one-step installation of PGF WatchTower with the Watchtower updater. Run the following command:

      ```sh
      curl -fsSL https://www.percona.com/get/pmm | /bin/bash
      ```

=== "Manual installation (For customization)"
    For a more customizable setup, follow these steps:
    {.power-number}
    
    1.  Create a Docker network for PGF WatchTower and the Watchtower updater:
         ```sh
         docker network create pmm-network
         ``` 

    2. (Optional but recommended) Install the Watchtower updater to enable PMM Server upgrades via the UI:

        - Create a user-defined token to secure the Watchtower updater's HTTP API. You can use any value or generate a secure token using `openssl` or another method. Ensure the same token is used in both the Watchtower updater and PMM Server configurations:

            ```sh   
            openssl rand -hex 16
            # Example output:
            e09541c81e672bf0e48dbc72d4f92790
            ```
        
        - Install the Watchtower updater using your token: 

            ```sh  
            docker run --detach \
            --restart always \
            --network=pmm-network \
            -e WATCHTOWER_HTTP_API_TOKEN=your_token \
            -e WATCHTOWER_HTTP_API_UPDATE=1 \
            --volume /var/run/docker.sock:/var/run/docker.sock \
            --name watchtower \
            percona/watchtower:latest
            ```

    3. Run PMM Server with Docker based on your preferred data storage method:
         - [Run Docker with host directory](../docker/run_with_host_dir.md)
         - [Run Docker with volume](../docker/run_with_vol.md)

    ## Configuration options

    ### Storage configuration

    You can choose either of two storage options offered by PMM Server:

    | Option | Suitable for | Docker parameter |
    |--------|-------------|---------|
    | [Docker volumes](../docker/run_with_vol.md) (Recommended) | Production environments | `--volume pmm-data:/srv` |
    | [Host directory](../docker/run_with_host_dir.md) | Development/testing | `--volume /path/on/host:/srv` |


    ### Environment variables

    Configure PMM Server's behavior using environment variables:

    ```sh
    docker run -e PMM_DATA_RETENTION=720h -e PMM_DEBUG=true percona/pmm-server:3
    ```

    Common variables:

    | Variable | Default | Description |
    |----------|---------|-------------|
    | `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data |
    | `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval |
    | `PMM_ENABLE_UPDATES` | `true` | Allow version checks and UI updates |
    | `PMM_ENABLE_TELEMETRY` | `true` | Send usage statistics |

    For a complete list, see the [environment variables](../docker/env_var.md).

## Access PMM Server

After installation:
{.power-number}

1. Access the PGF WatchTower interface in your browser: `https://SERVER_IP_ADDRESS` (replace with your server's address)

2. Log in with default credentials: `admin` / `admin`. 

3. Change the default password on first login.

## Advanced configuration
After basic installation, you may want to customize your PMM Server setup:

### Security options
- Configure a [trusted SSL certificate](../../../../admin/security/ssl_encryption.md) to remove browser warnings.
- Disable updates if needed:

    - **via Docker**:  add `-e PMM_ENABLE_UPDATES=0` to the `docker run` command (for the life of the container)
    - **via UI**: go to **Configuration > Settings > Advanced settings** and disable **Check for Updates** (can be turned back on by any admin in the UI)

- Enable HTTP (insecure, NOT recommended): add `--publish 80:8080` to the `docker run` command.

!!! info "Warning"
    PMM Client requires a secure (TLS-encrypted) connection and will only communicate with PMM Server over HTTPS.

## Next steps
- [Install PMM Client on hosts you want to monitor](../../../install-pmm-client/index.md)
- [Connect databases for monitoring](../../../install-pmm-client/connect-database/index.md)
