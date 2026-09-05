# Connect HAProxy databases to PGF WatchTower
Monitor your HAProxy load balancer performance with Postgres1st (PGF WatchTower). PGF WatchTower collects metrics from HAProxy's built-in Prometheus endpoint to provide insights into proxy performance, backend health, and traffic patterns.

## Prerequisites
Before adding HAProxy to PGF WatchTower, ensure:
{.power-number}

1. HAProxy configured with metrics endpoint. 
    - HAProxy must expose Prometheus metrics. See [How to configure HAProxy](https://www.haproxy.com/blog/haproxy-exposes-a-prometheus-metrics-endpoint).
    - Default metrics endpoint: `http://localhost:8404/metrics`
    - Verify metrics are accessible: `curl http://localhost:8404/metrics`

2. PMM Client installed and configured
  - PMM Client (pfw-agent) running on the same host as HAProxy
  - Node registered with PMM Server using pfw-admin config

## Add HAProxy service

Add HAProxy monitoring with the required port specification:

```sh
pfw-admin add haproxy --listen-port=8404
```

where `listen-port` is the port number where HAProxy is running. This is the only required flag.

??? example "Successful output"
    ```txt
    HAProxy Service added.
    Service ID  : c481183f-70a2-443f-91e5-cae5cecd06a2
    Service name: Ubuntu-haproxy
    ```
### Advanced configuration options
Customize the HAProxy service with additional parameters:

```sh
# With authentication
pfw-admin add haproxy --listen-port=8404 --username=pmm --password=pmm MyHAProxy

# With custom metrics path and HTTPS
pfw-admin add haproxy --listen-port=8404 --metrics-path=/prom-metrics --scheme=https

# With custom service name
pfw-admin add haproxy --listen-port=8404 Production-HAProxy
```

#### Available options

- `--listen-port`: HAProxy metrics port (required)
- `--username`: Basic authentication username
- `--password`: Basic authentication password
- `--metrics-path`: Metrics endpoint path (default: /metrics)
- `--scheme`: Connection protocol (http or https)
- `--skip-connection-check`: Skip connectivity validation

### Via web UI
To add HAProxy through the PGF WatchTower web interface:
{.power-number}

1. Go to **Inventory > Add service**.
2. Select HAProxy from the service types.
3. Configure the connection parameters then click **Add Service**.

## Verify the connection
Check that HAProxy monitoring is active:

```sh
pfw-admin status
```

HAProxy data is visible in the **Advanced Data Exploration** dashboard:

![!](../../../images/PMM_Advanced_Data_Exploration_HAProxy.png)
