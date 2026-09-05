default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

# Strip a leading `v` from `git describe` output IF present. This was `cut -b2-`,
# which drops the first character unconditionally -- correct for upstream's vX.Y.Z
# tags, but a release tag of ours (PFMM_3.9.0_BETA1 at the time) has no leading v,
# so it silently produced FMM_3.9.0_BETA1 in every binary's --version output.
# Stripping only a leading `v` is correct for either tag style.
PMM_RELEASE_PATH ?= ../bin
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | sed 's/^v//')
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)
PMM_DEV_SERVER_PORT ?= 443

# Architecture-dependent image defaults.
#
# The compose file's default MYSQL_IMAGE, percona:5.7, publishes amd64 only, so on an
# arm64 host (Apple silicon, Graviton) it will not start. percona-server:8.0 publishes
# both, so default to it there. Override MYSQL_IMAGE to pin something else.
#
# Not overridden, and why: postgres and percona-server-mongodb already publish arm64.
# aleksi/test_db is amd64-only and CANNOT be fixed by substitution -- but it does not
# need to be. It is a data-only container whose whole job is to seed world.sql into two
# named volumes, and Docker populates a named volume from image content when the
# container is CREATED, before anything executes. So on arm64 it seeds both volumes and
# then exits 255 with an exec-format error. That exit code is expected and harmless;
# check the volumes, not the container.
UNAME_M := $(shell uname -m)
ifneq (,$(filter $(UNAME_M),aarch64 arm64))
MYSQL_IMAGE ?= percona/percona-server:8.0
endif
export MYSQL_IMAGE
export MONGO_IMAGE
export POSTGRES_IMAGE
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif
GOPATH=$(shell go env GOPATH)

VERSION_FLAGS = -X 'github.com/percona/pmm/version.ProjectName=pfw-agent' \
				-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
				-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
				-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
				-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
				-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)'

release:                        ## Build static pmm-agent release binary (Linux only)
	CGO_ENABLED=1 go build -v -ldflags "-extldflags '-static' $(VERSION_FLAGS)" -tags 'osusergo netgo static_build' -o $(PMM_RELEASE_PATH)/pfw-agent
	go build -v -ldflags "-extldflags '-static' $(VERSION_FLAGS)" -tags 'osusergo netgo static_build' -o $(PMM_RELEASE_PATH)/pfw-agent-entrypoint ./cmd/pfw-agent-entrypoint
	$(PMM_RELEASE_PATH)/pfw-agent --version
	ldd $(PMM_RELEASE_PATH)/pfw-agent 2>&1 | grep -Fq 'not a dynamic executable'

release-gssapi:                 ## Build dynamic pmm-agent release binary (for GSSAPI support)
	CGO_ENABLED=1 go build -v -ldflags "$(VERSION_FLAGS)" -tags 'osusergo netgo gssapi' -o $(PMM_RELEASE_PATH)/pfw-agent
	$(PMM_RELEASE_PATH)/pfw-agent --version

release-dev:
	CGO_ENABLED=1 go build -race -v -ldflags "$(VERSION_FLAGS)" -o $(PMM_RELEASE_PATH)/pfw-agent

release-dev-gssapi:
	CGO_ENABLED=1 go build -race -v -ldflags "$(VERSION_FLAGS)" -tags 'gssapi' -o $(PMM_RELEASE_PATH)/pmm-agent-gssapi

gen: clean                      ## Generate files
	go generate ./...

clean:                          ## Remove generated files
	find . -name *_reform.go -delete

# generate stub models for perfschema QAN agent; hidden from help as it is not generally useful
gen-init:
	mkdir tmp-mysql
	go tool reform-db -db-driver=mysql -db-source='root:root-password@tcp(127.0.0.1:3306)/performance_schema' init tmp-mysql

install:                        ## Install pmm-admin binary
	go build -v -ldflags "$(VERSION_FLAGS)" -o $(GOBIN)/pmm-agent

install-race:                   ## Install pmm-admin binary with race detector
	go build -v -ldflags "$(VERSION_FLAGS)" -race -o $(GOBIN)/pmm-agent

# TODO https://jira.percona.com/browse/PMM-4681
# TEST_PARALLEL_PACKAGES ?= foo bar
# go test $(TEST_FLAGS) $(TEST_PARALLEL_PACKAGES) - without `-p 1`

# -timeout is a budget for the whole package binary, not per test, and
# agents/postgres/pgstatstatements alone needs ~90s: TestPGStatStatementsQPS
# issues 14000 sequential round-trips (7000 CREATE TABLE + 7000 INSERT, with
# pgss.max deliberately left above that) plus 7000 DROP TABLE in cleanup. At 60s
# it panicked with "test timed out" on the slower CI runners while passing on the
# faster ones, and a timeout panic skips t.Cleanup, leaving 7000 tables behind to
# break the next run against the same database.
TEST_FLAGS ?= -timeout=180s

test:                           ## Run tests
	go test $(TEST_FLAGS) -p 1 -race ./...

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(TEST_FLAGS) -p 1 -race -coverprofile=cover.out -covermode=atomic ./...

fuzz-slowlog-parser:            ## Run fuzzer for agents/mysql/slowlog/parser package
	# go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p agents/mysql/slowlog/parser/corpus
	cp agents/mysql/slowlog/parser/testdata/*.log agents/mysql/slowlog/parser/corpus/
	cd agents/mysql/slowlog/parser && go-fuzz-build
	cd agents/mysql/slowlog/parser && go-fuzz

fuzz-postgres-parser:   ## Run fuzzer for agents/postgres/parser package
	# FIXME see https://github.com/AlekSi/go-fuzz/pull/1

	# go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p agents/postgres/parser/fuzzing/corpus
	cp agents/postgres/parser/testdata/*.sql agents/postgres/parser/fuzzing/corpus/
	cd agents/postgres/parser && go-fuzz-build
	cd agents/postgres/parser && go-fuzz -workdir fuzzing

bench:                          ## Run benchmarks
	go test -bench=. -benchtime=1s -count=5 -cpu=1 -timeout=30m -failfast github.com/percona/pmm/agent/agents/mysql/slowlog/parser | tee slowlog_parser_new.bench
	go tool benchstat slowlog_parser_old.bench slowlog_parser_new.bench

	go test -bench=. -benchtime=1s -count=5 -cpu=1 -timeout=30m -failfast github.com/percona/pmm/agent/agents/postgres/parser | tee postgres_parser_new.bench
	go tool benchstat postgres_parser_old.bench postgres_parser_new.bench

RUN_FLAGS = --config-file=pmm-agent.yaml

run: install _run               ## Run pmm-agent

run-race: install-race _run     ## Run pmm-agent with race detector

run-race-cover: install-race    ## Run pmm-agent with race detector and collect coverage information
	go test -coverpkg="github.com/percona/pmm/agent/..." \
			-tags maincover \
			-ldflags "$(VERSION_FLAGS)" \
			-race -c -o bin/pmm-agent.test
	bin/pmm-agent.test -test.coverprofile=cover.out -test.run=TestMainCover -- $(RUN_FLAGS)

_run:
	pmm-agent $(RUN_FLAGS)

ENV_UP_FLAGS ?= --force-recreate --renew-anon-volumes --remove-orphans

# The data services the test suites actually dial: agent tests hardcode
# localhost:5432, 127.0.0.1:3306 and localhost:27017, so these must publish on the host
# rather than talk over the compose network. Deliberately excludes pmm-server: that is
# perconalab/pmm-server:3-dev-latest, which is amd64-only AND upstream's image, so our
# renamed paths are not in it -- see the disabled job in .github/workflows/managed.yml.
# Use this for `go test ./agent/...`; use env-up when you also need a server.
env-up-db:                      ## Start only the databases the tests dial
	$(MAKE) _testdata
	docker compose up $(ENV_UP_FLAGS) -d test_db mysql mongo postgres valkey

_testdata:
	rm -fr testdata
	# to make slowlog rotation tests work
	mkdir -p testdata/mysql/slowlogs
	# to make mongolog tests work
	mkdir -p testdata/mongo/var/log/mongodb
	touch testdata/mongo/var/log/mongodb/mongo.log testdata/mongo/var/log/mongodb/mongo_tls.log
	chmod -R 0777 testdata

env-up:                         ## Start development environment
	rm -fr testdata
	# to make slowlog rotation tests work
	mkdir -p testdata/mysql/slowlogs
	# to make mongolog tests work
	mkdir -p testdata/mongo/var/log/mongodb
	touch testdata/mongo/var/log/mongodb/mongo.log testdata/mongo/var/log/mongodb/mongo_tls.log
	chmod -R 0777 testdata

	docker compose up $(ENV_UP_FLAGS) -d

env-down:                       ## Stop development environment
	docker compose down --volumes --remove-orphans

setup-dev: install              ## Run pmm-agent setup in development environment
	pmm-agent setup $(RUN_FLAGS) --server-insecure-tls --server-address=127.0.0.1:${PMM_DEV_SERVER_PORT} --server-username=admin --server-password=admin --paths-exporters_base=$(GOPATH)/bin --force

env-mysql:                      ## Run mysql client
	docker exec -ti pmm-agent_mysql mysql --host=127.0.0.1 --user=root --password=root-password

env-mongo:                      ## Run mongo client
	docker exec -ti pmm-agent_mongo mongo --username=root --password=root-password

env-psql:                       ## Run psql client
	docker exec -ti pmm-agent_postgres env PGPASSWORD=pmm-agent-password psql --username=pmm-agent

env-sysbench-prepare:
	docker compose exec --workdir=/sysbench/sysbench-tpcc sysbench ./tpcc.lua \
		--db-driver=pgsql --pgsql-host=postgres --pgsql-user=pmm-agent --pgsql-password=pmm-agent-password --pgsql-db=pmm-agent \
		--threads=1 --time=0 --report-interval=10 \
		--tables=1 --scale=10  --use_fk=0 --enable_purge=yes prepare

env-sysbench-run:
	docker compose exec --workdir=/sysbench/sysbench-tpcc sysbench ./tpcc.lua \
		--db-driver=pgsql --pgsql-host=postgres --pgsql-user=pmm-agent --pgsql-password=pmm-agent-password --pgsql-db=pmm-agent \
		--threads=4 --time=0 --rate=10 --report-interval=10 --percentile=99 \
		--tables=1 --scale=10  --use_fk=0 --enable_purge=yes run
