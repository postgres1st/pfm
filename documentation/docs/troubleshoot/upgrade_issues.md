# Troubleshoot upgrade issues

## PMM Server not updating correctly

If the automatic update process isn't working, you can force an update using the API:
{.power-number}

1. Open your terminal.
2. Run the update command, replacing <username>:<password> with your credentials and <pmm-server-address> with your PMM server address:
```sh
curl -X POST \
  --user <username>:<password> \
  'http://<pmm-server-address>/v1/server/updates:start' \
  -H 'Content-Type: application/json'
```
3. Wait 2-5 minutes and refresh the Postgres1st WatchTower Home page to verify the update.

## Watchtower updater fails with "client version is too old" error

When upgrading PMM Server via the UI, the Watchtower updater may fail with the following error:

```
client version X.XX is too old. Minimum supported API version is X.XX, please upgrade your client to a newer version
```

This occurs when your Docker installation requires a newer API version than the Watchtower updater supports.

**Solution**
To resolve this issue:
{.power-number}

1. Update to the latest Watchtower updater image:
```
docker pull percona/watchtower:latest
```
2. If the error persists, add the `DOCKER_API_VERSION` environment variable matching your Docker's minimum API version:
```
-e DOCKER_API_VERSION=1.45
```