# Get help

Our documentation guides are packed with information, but they can't cover everything you
need to know about Postgres1st WatchTower, and they won't cover every scenario you might come
across. Don't be afraid to try things out and ask questions when you get stuck.

## Contact us

Get in touch through the [Postgres1st contact page](https://www.postgresfirst.com/about/contact){:target="_blank"}.
Tell us the version you are running (`pfw-admin --version`), what you expected to happen,
and what happened instead.

For an installed server, the fastest way to give us something to work with is a diagnostic
archive:

```sh
pfw-admin summary
```

That collects the agent configuration, versions, status and logs into a single zip. Review
it before sending — it contains details of your environment.

## Upstream documentation

Postgres1st WatchTower is derived from Percona Monitoring and Management. Where a component is
unmodified from upstream, Percona's documentation for it may still be the most detailed
reference available, and their community forum may cover questions about it.

Those are **not our support channels**. Percona cannot answer questions about Postgres1st
WatchTower, cannot see our builds, and our packages, paths and service names differ from
theirs. Anything specific to this product should come to us.
