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
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif

LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm/version.ProjectName=vmproxy' \
			-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build vmproxy release binary
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/vmproxy .

install:                        ## Install vmproxy binary
	go build -v $(LD_FLAGS) -o $(GOBIN)/vmproxy .

install-race:                   ## Install vmproxy binary with race detector
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/vmproxy .

build:                          ## Build vmproxy binary to current directory
	go build -v $(LD_FLAGS) -o vmproxy .

TEST_FLAGS ?= -timeout=20s

test:                           ## Run tests
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(TEST_FLAGS) -race -coverprofile=cover.out -covermode=atomic ./...
