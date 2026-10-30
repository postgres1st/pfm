# Configure Nomad

Postgres1st (PGF WatchTower) includes HashiCorp Nomad to enable future extensibility and enhanced service capabilities.

[Nomad](https://www.nomadproject.io/) is a workload orchestrator designed to deploy and manage containers and non-containerized applications. In PGF WatchTower, Nomad provides the underlying infrastructure to:

- improve resource allocation across monitoring components
- enable future PGF WatchTower extensibility 
- manage distributed monitoring agents more efficiently

Nomad is **disabled by default** in PGF WatchTower and has no impact on system performance when not enabled. 

## Prerequisites

Nomad requires **both** of the following to start:

- `PMM_ENABLE_NOMAD=1` — enables the Nomad feature flag
- `PMM_PUBLIC_ADDRESS=<your-pmm-server-address>` — the publicly reachable address of your PGF WatchTower Server

Setting `PMM_ENABLE_NOMAD=1` alone is not sufficient. If `PMM_PUBLIC_ADDRESS` is not set, PGF WatchTower silently skips starting the Nomad server process with no error or warning.

You can set the public address either as an environment variable at container start, or later via **Configuration > Settings > Advanced settings** in the PGF WatchTower UI (though setting it via the UI requires restarting the container for Nomad to pick it up).

### Enable Nomad

If you're an advanced user who needs Nomad for specific use cases, follow these steps to enable Nomad in PGF WatchTower:
{ .power-number }

1. Start PGF WatchTower Server with the `PMM_ENABLE_NOMAD` and `PMM_PUBLIC_ADDRESS` environment variables:
   ```
   -e PMM_ENABLE_NOMAD=1 \
   -e PMM_PUBLIC_ADDRESS=<your-pmm-server-address>
   ```

2. Expose the Nomad port:
   ```
   -p 4647:4647
   ```

??? info "Docker run command" 

    ```
    docker run -d \
    -e PMM_ENABLE_NOMAD=1 \
    -e PMM_PUBLIC_ADDRESS=<your-pmm-server-address> \
    -p 4647:4647 \
    -p 443:8443 \
    --name pmm-server \
    percona/pmm-server:3
    ```

### Disable Nomad

To disable Nomad:

```
-e PMM_ENABLE_NOMAD=0
```

When Nomad is disabled on the PGF WatchTower Server, the Nomad agent on PGF WatchTower Agents will automatically stop.

## System requirements

When Nomad is enabled, PGF WatchTower Agent nodes have the following additional requirements:

-  `iproute` package must be installed
-  access to cgroup must be available

## Verification

To verify that Nomad is running correctly:
{ .power-number }

1. Check that the Nomad API is available at:
   ```
   https://<PMM_SERVER_URL>/nomad/v1/nodes
   ```

2. Confirm that Nomad agents appear in the node list.

## Internal architecture

When enabled, PGF WatchTower runs the following Nomad components:

- **Nomad server** on PGF WatchTower Server - manages the cluster and schedules workloads
- **Nomad client** on PGF WatchTower Server - executes jobs (workloads) on remote instances
- **Nomad client** on PGF WatchTower Agents - executes distributed workloads

Communication between these components is secured and managed automatically when configured with the proper public address.

## API access
The Nomad API is available through the PGF WatchTower Server's HTTPS port via the `/nomad` prefix. This allows you to access Nomad endpoints without requiring a separate port for the Nomad API.

- Nomad endpoints are only available to users with admin privileges
- All Nomad API endpoints are accessible under the `/nomad` path
- The standard Nomad API documentation applies, but all requests must use the `/nomad`prefix

??? info "Example API request" 

    `https://<PMM_SERVER_URL>/nomad/v1/jobs`

## Future compatibility

Nomad is included in PGF WatchTower to support future extensibility features. Nomad will remain within PGF WatchTower to provide infrastructure for upcoming enhancements and to deliver improved services for existing Percona customers.

## Related links

- [Nomad documentation](https://developer.hashicorp.com/nomad/docs)
