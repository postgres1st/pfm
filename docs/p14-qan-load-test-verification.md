# P14 — End-to-end Query Analytics: sustained-load verification

Roadmap outcome (as written): "Validate the complete path from a real PostgreSQL
workload through the approved statistics collection path into the WatchTower
interface under sustained load."

Unlike P12, this one has no native-host-only blocker — it's pure software pipeline
(`pg_stat_statements → pmm-agent → pmm-managed → qan-api2 → ClickHouse → interface`),
fully exercisable in the devcontainer. This is both the reproduction steps and the
record of one run's actual results (2026-09-16), with screenshots.

---

## What was already true before any load was generated

`pmm.metrics` in ClickHouse already had 254,172 rows for `pmm-server-postgresql`
(the server's own internal DB), spanning 13 days continuously, with zero recorded
ingestion errors. That proves the pipeline *works* — it does not prove it holds up
under **sustained** load, because that traffic is light, incidental background
usage (~53-60 queries/min from the product's own housekeeping), not a deliberate
throughput test. That gap is what this document closes.

---

## Setup

### Devcontainer gotcha hit first

`make env-up` failed with `pmm-server` reported unhealthy; `qan-api2` was in a
crash-loop (`BACKOFF`). Cause: `dev/clickhouse-config.xml` ships
`settings_constraints_replace_previous=false`, but the image's own
`default-users.xml` `datasource` profile requires it `true` (ClickHouse
25.3.6.56 rejects `changeable_in_readonly` otherwise) — the same known
devcontainer/image drift noted elsewhere for this repo. Fixed locally (not
committed — it's an environment quirk, not a product bug):

```bash
sed -i '' 's#<settings_constraints_replace_previous>false#<settings_constraints_replace_previous>true#' \
  dev/clickhouse-config.xml
docker exec pmm-server supervisorctl restart clickhouse
```

### A separate target database — not the server's own internal Postgres

Load-testing against `pmm-server`'s own metadata DB would be testing the wrong
thing (and is riskier). `docker-compose.yml` already has a `postgres` profile
for exactly this:

```bash
docker compose -f docker-compose.yml --profile postgres up -d
```

This does **not** use the `pmm` profile (which would collide with the
dev container's own `pmm-server` — both compose files name a container
`pmm-server`). The plain `postgres` container lands on the same `pfm_default`
network and is reachable from the devcontainer as host `postgres`.

### Register it as a monitored service, with the approved QAN collector

```bash
docker exec pmm-server pmm-admin add postgresql \
  --server-url=https://admin:admin@127.0.0.1:8443 --server-insecure-tls \
  --username=postgres --password=secret \
  --query-source=pgstatements \
  pgbench-target postgres:5432
```

`--query-source=pgstatements` matters: the default is `pgstatmonitor`, which
needs the `pg_stat_monitor` extension — not present on a vanilla `postgres:18`
image, and `pmm-admin` will silently register the service without a working
QAN agent if the value is wrong. Confirm the agent actually attached:

```bash
docker exec pmm-server pmm-admin list --server-url=https://admin:admin@127.0.0.1:8443 --server-insecure-tls
# expect a `postgresql_pgstatements_agent` row against the new service ID
```

### Generate the workload

```bash
docker exec pmm-server sh -c 'PGPASSWORD=secret /usr/pgsql-18/bin/pgbench -h postgres -U postgres -i -s 10 postgres'
docker exec pmm-server sh -c 'PGPASSWORD=secret /usr/pgsql-18/bin/pgbench -h postgres -U postgres -c 10 -j 4 -T 600 -P 30 postgres'
```

`pgbench` is present in the image at `/usr/pgsql-18/bin/pgbench`, just not on
`$PATH`. TPC-B workload, scale factor 10, 10 clients / 4 threads, 600 seconds.

---

## Results — textual proof

### Load generated

```
number of transactions actually processed: 3266562
number of failed transactions: 0 (0.000%)
latency average = 1.830 ms
tps = 5444.479785 (without initial connection time)
```

### Ingestion — no gaps, no errors

```sql
SELECT count(), min(period_start), max(period_start)
FROM pmm.metrics WHERE service_name='pgbench-target'
-- 523  2026-09-16 15:44:37  2026-09-16 15:54:00
```

Every one-minute bucket present across the full run window, no gap:

```
15:44  57   15:47  44   15:50  44   15:53  50
15:45  49   15:48  59   15:51  44   15:54  44
15:46  44   15:49  44   15:52  44
```

qan-api2's own ingestion-error counter, before and after:

```
qan_api2_data_ingestion_batch_save_seconds_count{error="0"} 15 → 26
qan_api2_data_ingestion_batch_save_seconds_count{error="1"}  0 →  0
```

### Pipeline health under load, not just correctness

```
pmm-managed / pmm-agent / qan-api2 logs: zero ERROR-level entries during the run
ClickHouse: 0 active merges, 17 active parts on `metrics` (no compaction backlog)
qan-api2 process: 90 MB RSS, 0.4% CPU shortly after the run — no leak, no runaway growth
```

### Interface API — same data the UI panel reads

```bash
curl -s -X POST "http://127.0.0.1:9922/v1/qan/metrics:getReport" \
  -H "Content-Type: application/json" \
  -d '{"period_start_from":"2026-09-16T15:44:00Z","period_start_to":"2026-09-16T15:55:00Z",
       "group_by":"queryid","labels":[{"key":"service_name","value":["pgbench-target"]}],
       "columns":["num_queries","load"],"order_by":"-num_queries","limit":10}'
```

Returned 88 distinct query fingerprints, every pgbench statement type present
(`begin`, `UPDATE pgbench_accounts/tellers/branches`, `SELECT abalance`,
`INSERT pgbench_history`, `commit`), ~3,243,850 executions recorded per
statement — 99.3% of pgbench's 3,266,562 actual transactions. The ~0.7% gap is
consistent with bucket-boundary timing at the tail of the run, not dropped data
(the ClickHouse gap-check above shows no missing buckets).

---

## Results — visual proof (WatchTower UI)

Query Analytics (`Dashboards → Query Analytics (QAN)`), filtered to
`service_name=pgbench-target`, time range fixed to the exact run window
(2026-09-16 15:44:00–15:54:30 UTC):

![QAN query list, filtered to pgbench-target for the exact test window](assets/p14-qan-load-test/qan-query-list.jpg)

Real, correct data: every pgbench statement listed with real QPS and load
figures (`UPDATE pgbench_branches` — 1.74 load, 5.15k QPS, 338.74 µs; and so
on), 88 items total, `pgbench-target` at 99.13% of visible load share.

Clicking into a query and opening its **Tables** tab pulls the *live* schema
from the target database — not a cached/stored value, a real-time
introspection call against `pgbench_branches`:

![QAN query detail, Tables tab, showing the live pgbench_branches schema](assets/p14-qan-load-test/qan-table-schema.jpg)

**Note on the Examples tab**: it reported "no examples found," and that's
correct behavior, not a bug — `pg_stat_statements` (the collector used here,
one of the two approved QAN sources) only captures normalized query text and
aggregate stats, never literal parameter values. Only `pg_stat_monitor` (the
other approved collector) captures examples, and it isn't installed on this
target. Worth knowing before assuming "no examples" means something is broken.

---

## Pass/fail

| Check | Result |
|---|---|
| Sustained load generated (10 min, real transactions, 0 failures) | **PASS** |
| No gap in ingested data across the run window | **PASS** |
| Zero ingestion errors (`qan_api2_data_ingestion_batch_save_seconds{error="1"}`) | **PASS** |
| Zero ERROR-level log entries across the pipeline during the run | **PASS** |
| ClickHouse remained healthy under the write load (no merge backlog) | **PASS** |
| qan-api2 process stable (no leak/runaway growth) | **PASS** |
| Correct data via the report API (same path the UI reads) | **PASS** |
| Correct data rendered in the actual UI, with live table introspection | **PASS** |

---

## Cleanup

```bash
docker exec pmm-server pmm-admin remove postgresql --server-url=https://admin:admin@127.0.0.1:8443 --server-insecure-tls pgbench-target
docker compose -f docker-compose.yml --profile postgres down -v
git checkout dev/clickhouse-config.xml   # revert the local-only devcontainer fix if not upstreaming it
```
