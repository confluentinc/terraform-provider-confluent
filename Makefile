TEST?=./...

# Project variables
NAME        := terraform-provider-confluent
# Build variables
BUILD_DIR   := bin
VERSION     ?= $(shell git tag --sort=-creatordate | grep -v ".*deleted" | head -n 1)
# Go variables
GOENV         := GO111MODULE=on
GOCMD         := $(GOENV) go
# Pinned rather than @latest: the JUnit XML it emits is a contract with `test-results publish`.
GOTESTSUM_VERSION := v1.13.0
# Skips the install when the pinned version is present. Inlined rather than reached through
# $(MAKE): make force-runs any recipe line containing $(MAKE) even under -n, and a continued recipe
# is one line, so recursing inside live-test's shell block made `make -n` hit production for real.
GOTESTSUM_INSTALL = gotestsum --version 2>/dev/null | grep -qF '$(GOTESTSUM_VERSION)' || go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
# Per-test terraform debug logs under CI. Renaming needs matching edits in
# .semaphore/semaphore.yml and .gitignore, or the artifact push becomes a green no-op.
TF_LOG_DIR    := tflogs
GOBUILD       ?= CGO_ENABLED=0 $(GOCMD) build -mod=vendor
GOOS          ?= $(shell go env GOOS)
GOARCH        ?= $(shell go env GOARCH)
GOFILES       ?= $(shell find . -type f -name '*.go' -not -path "./vendor/*")

BRANCH_NAME ?= $(shell git rev-parse --abbrev-ref HEAD || true)
CLEAN_VERSION := $(shell echo $(VERSION) | grep -Eo '([0-9]+\.){2}[0-9]+')
MASTER_BRANCH := master

# Auto bump by default
BUMP ?= auto
DEFAULT_BUMP ?= patch
GIT_MESSAGES := $(shell git log --pretty='%s' v$(CLEAN_VERSION)...HEAD 2>/dev/null | tr '\n' ' ')

# If auto bump enabled, search git messages for bump hash
ifeq ($(BUMP),auto)
_auto_bump_msg := \(auto\)
ifneq (,$(findstring #major,$(GIT_MESSAGES)))
BUMP := major
else ifneq (,$(findstring #minor,$(GIT_MESSAGES)))
BUMP := minor
else ifneq (,$(findstring #patch,$(GIT_MESSAGES)))
BUMP := patch
else
BUMP := $(DEFAULT_BUMP)
endif
endif

# Figure out what the next version should be
split_version := $(subst ., ,$(CLEAN_VERSION))
ifeq ($(BUMP),major)
bump := $(shell expr $(word 1,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(bump).0.0
else ifeq ($(BUMP),minor)
bump := $(shell expr $(word 2,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(bump).0
else ifeq ($(BUMP),patch)
bump := $(shell expr $(word 3,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(word 2,$(split_version)).$(bump)
else ifeq ($(BUMP),none)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(word 2,$(split_version)).$(word 3,$(split_version))
endif

BUMPED_VERSION := v$(BUMPED_CLEAN_VERSION)

RELEASE_SVG := <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="94" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="\#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="94" height="20" rx="3" fill="\#fff"/></clipPath><g clip-path="url(\#a)"><path fill="\#555" d="M0 0h49v20H0z"/><path fill="\#007ec6" d="M49 0h45v20H49z"/><path fill="url(\#b)" d="M0 0h94v20H0z"/></g><g fill="\#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="255" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="390">release</text><text x="255" y="140" transform="scale(.1)" textLength="390">release</text><text x="705" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text><text x="705" y="140" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text></g> </svg>

.PHONY: all
all: clean deps test testacc tools build

.PHONY: release-ci
release-ci:
ifeq ($(BRANCH_NAME), $(MASTER_BRANCH))
ifeq ($(CI),true)
	make release
endif
endif

.PHONY: release
release: get-release-image commit-release tag-release

.PHONY: checkfmt
checkfmt: RESULT = $(shell goimports -l $(GOFILES) | tee >(if [ "$$(wc -l)" = 0 ]; then echo "OK"; fi))
checkfmt: SHELL := /usr/bin/env bash
checkfmt: ## Check formatting of all go files
	@ echo "$(RESULT)"
	@ if [ "$(RESULT)" != "OK" ]; then exit 1; fi

.PHONY: fmt
fmt: ## Format all go files
	@ $(MAKE) --no-print-directory log-$@
	goimports -w $(GOFILES)

.PHONY: clean
clean: ## Clean workspace
	@ $(MAKE) --no-print-directory log-$@
	rm -rf ./$(BUILD_DIR) ./$(TF_LOG_DIR) ./*-report.xml ./$(TF_LOG_DIR).tar.gz

.PHONY: deps
deps: ## Fetch dependencies
	@ $(MAKE) --no-print-directory log-$@
	$(GOCMD) mod vendor

.PHONY: build
build: clean ## Build binary for current OS/ARCH
	@ $(MAKE) --no-print-directory log-$@
	$(GOBUILD) -o ./$(BUILD_DIR)/$(GOOS)-$(GOARCH)/$(NAME)

# Under CI these run through gotestsum, which writes the JUnit XML Semaphore publishes. CI-only
# because testacc and live-test* are documented local workflows (docs/DEVELOPING.md).
# -v is omitted from every CI invocation: it strips `go test -json`'s framing markers, after which
# stray stderr parses as framing and invents test cases. --format testname keeps the per-test
# progress the default pkgname format drops, while still discarding passing tests' output.
# Unlike live-test*, a failed install fails the build here, since nothing pages on semaphore.yml.
.PHONY: test
test:
ifeq ($(CI),true)
	@$(GOTESTSUM_INSTALL)
	$(GOENV) gotestsum --format testname --junitfile unit-report.xml -- ./...
else
	$(GOCMD) test ./...
endif

# TF_LOG_PATH_MASK gives each test its own debug log, so a failure's captured output is the
# assertion rather than whatever else was logging; semaphore.yml tars $(TF_LOG_DIR) as an artifact.
# The mask must be absolute: go test runs each binary from its own package directory, and the SDK
# log.Fatals on a mask it cannot open.
.PHONY: testacc
testacc:
ifeq ($(CI),true)
	@$(GOTESTSUM_INSTALL)
	mkdir -p $(TF_LOG_DIR)
	TF_LOG=debug TF_LOG_PATH_MASK='$(CURDIR)/$(TF_LOG_DIR)/%s.log' TF_ACC=1 $(GOENV) gotestsum --format testname --junitfile acceptance-report.xml -- $(TEST) $(TESTARGS) -coverprofile=coverage.txt -covermode=atomic -timeout 120m -failfast
else
	TF_LOG=debug TF_ACC=1 $(GOCMD) test $(TEST) -v $(TESTARGS) -coverprofile=coverage.txt -covermode=atomic -timeout 120m -failfast
endif
	@echo "finished testacc"

# Live integration tests with group filtering and concurrency support
# Usage: make live-test TF_LIVE_TEST_GROUPS="core,kafka" or make live-test (for all)
# RTCE tests are excluded here because RTCE prod is only enabled in aws.us-east-1;
# run them via the dedicated `live-test-rtce` target below.
# VERBOSE is empty under CI deliberately: -v breaks gotestsum's report, per the note above `test`.
# A failed gotestsum install degrades to plain `go test` rather than aborting: smoke-tests wraps
# this in a PASS/FAIL that pages, so a module-proxy blip must not look like a Confluent outage.
.PHONY: live-test
live-test:
	@echo "Running live integration tests against Confluent Cloud..."
	@if [ "$(CI)" != "true" ]; then \
		RUNNER="go test"; VERBOSE="-v"; \
	elif $(GOTESTSUM_INSTALL); then \
		RUNNER="gotestsum --format testname --junitfile live-report.xml --"; VERBOSE=""; \
	else \
		echo "gotestsum unavailable, running without a JUnit report"; \
		RUNNER="go test"; VERBOSE="-v"; \
	fi; \
	if [ -z "$(TF_LIVE_TEST_GROUPS)" ]; then \
		echo "Running ALL live tests with parallel execution..."; \
		$(GOENV) TF_ACC=1 TF_ACC_PROD=1 $$RUNNER ./internal/provider/ $$VERBOSE -run=".*Live$$|.*DriftDetection$$" -skip="Rtce" -tags="live_test,all" -timeout 1440m -parallel 10; \
	else \
		echo "Running live tests for groups: $(TF_LIVE_TEST_GROUPS) with parallel execution..."; \
		TAGS="live_test"; \
		for group in $$(echo "$(TF_LIVE_TEST_GROUPS)" | tr ',' ' '); do \
			TAGS="$$TAGS,$$group"; \
		done; \
		$(GOENV) TF_ACC=1 TF_ACC_PROD=1 $$RUNNER ./internal/provider/ $$VERBOSE -run=".*Live$$|.*DriftDetection$$" -skip="Rtce" -tags="$$TAGS" -timeout 1440m -parallel 10; \
	fi
	@echo "Finished running live integration tests against Confluent Cloud"

# RTCE live tests — pinned to aws.us-east-1 because that's the only region
# where RTCE is enabled in prod. Run separately from the main live-test target.
# Degrades on a failed install for the same reason as live-test: live-tests.yml pages on any
# nonzero exit here, so a module-proxy blip must not look like a Confluent outage.
.PHONY: live-test-rtce
live-test-rtce:
	@echo "Running RTCE live integration tests against Confluent Cloud (region=us-east-1)..."
	@if [ "$(CI)" != "true" ]; then \
		RUNNER="go test"; VERBOSE="-v"; \
	elif $(GOTESTSUM_INSTALL); then \
		RUNNER="gotestsum --format testname --junitfile live-rtce-report.xml --"; VERBOSE=""; \
	else \
		echo "gotestsum unavailable, running without a JUnit report"; \
		RUNNER="go test"; VERBOSE="-v"; \
	fi; \
	TF_ACC=1 TF_ACC_PROD=1 TF_ACC_REGION=us-east-1 $(GOENV) $$RUNNER ./internal/provider/ $$VERBOSE -run="Rtce.*Live$$" -tags="live_test,all" -timeout 1440m -parallel 10
	@echo "Finished running RTCE live integration tests"

# Helper targets for common group combinations
.PHONY: live-test-core
live-test-core:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="core"

.PHONY: live-test-kafka
live-test-kafka:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="kafka"

.PHONY: live-test-connect
live-test-connect:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="connect"

.PHONY: live-test-schema-registry
live-test-schema-registry:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="schema_registry"

.PHONY: live-test-networking
live-test-networking:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="networking"

.PHONY: live-test-flink
live-test-flink:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="flink"

.PHONY: live-test-rbac
live-test-rbac:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="rbac"

.PHONY: live-test-data-catalog
live-test-data-catalog:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="data_catalog"

.PHONY: live-test-tableflow
live-test-tableflow:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="tableflow"

.PHONY: live-test-crp
live-test-crp:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="crp"

.PHONY: live-test-drift
live-test-drift:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="drift"

.PHONY: live-test-essential
live-test-essential:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="core,kafka"

.PHONY: live-test-smoke
live-test-smoke:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="smoke"

.PHONY: build-otel-smoke-metric
build-otel-smoke-metric:
	$(GOBUILD) -o ./$(BUILD_DIR)/otel-smoke-metric ./cmd/otel-smoke-metric


install: build
	mkdir -p ~/.terraform.d/plugins/$(GOOS)_$(GOARCH)
	cp ./$(BUILD_DIR)/$(GOOS)-$(GOARCH)/$(NAME) ~/.terraform.d/plugins/$(GOOS)_$(GOARCH)/

.PHONY: gox
gox:
	go install github.com/mitchellh/gox@latest

.PHONY: goimports
goimports:
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY: tools
tools: ## Install required tools
	@ $(MAKE) --no-print-directory log-$@
	@ $(MAKE) --no-print-directory goimports gox

log-%:
	@ grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m==> %s\033[0m\n", $$2}'

.PHONY: commit-release
commit-release:
	git diff --exit-code --cached --name-status || \
	git commit -m "chore: $(BUMP) version bump $(BUMPED_VERSION)"

.PHONY: get-release-image
get-release-image:
	echo '$(RELEASE_SVG)' > release.svg
	git add release.svg

.PHONY: tag-release
tag-release:
	git tag $(BUMPED_VERSION)
	git push origin $(BUMPED_VERSION)
	git push origin $(MASTER_BRANCH) || true

.PHONY: show-version
show-version:
	@echo version: $(VERSION)
	@echo clean version: $(CLEAN_VERSION)
	@echo version bump: $(BUMP) $(_auto_bump_msg)
	@echo bumped version: $(BUMPED_VERSION)
	@echo bumped clean version: $(BUMPED_CLEAN_VERSION)

# Fetch the latest tag from internal repo and save the tag version
# Add the public repo as the remote repo with alias as "public"
# Finally publish the latest tag version to the public repo
PUBLIC_REPO_TAG_VERSION := $(shell git log --tags --simplify-by-decoration --pretty='format:%d' | \
		grep -o 'tag: v[0-9]*\.[0-9]*\.[0-9]*[-a-zA-Z0-9]*' | \
		grep -v '.*deleted' | \
		head -n 1 | \
		sed 's/tag: //')
