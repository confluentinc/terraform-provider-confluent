TEST?=./...

# Project variables
NAME        := terraform-provider-confluent
# Build variables
BUILD_DIR   := bin
VERSION     ?= $(shell git tag --sort=-creatordate | grep -v ".*deleted" | head -n 1)
# Go variables
GOCMD         := GO111MODULE=on go
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
ifneq (,$(findstring \#major,$(GIT_MESSAGES)))
BUMP := major
else ifneq (,$(findstring \#minor,$(GIT_MESSAGES)))
BUMP := minor
else ifneq (,$(findstring \#patch,$(GIT_MESSAGES)))
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
	rm -rf ./$(BUILD_DIR)

.PHONY: deps
deps: ## Fetch dependencies
	@ $(MAKE) --no-print-directory log-$@
	$(GOCMD) mod vendor

.PHONY: build
build: clean ## Build binary for current OS/ARCH
	@ $(MAKE) --no-print-directory log-$@
	$(GOBUILD) -o ./$(BUILD_DIR)/$(GOOS)-$(GOARCH)/$(NAME)

.PHONY: test
test:
	$(GOCMD) test ./...

.PHONY: testacc
testacc:
	TF_LOG=debug TF_ACC=1 $(GOCMD) test $(TEST) -v $(TESTARGS) -coverprofile=coverage.txt -covermode=atomic -timeout 120m -failfast
	@echo "finished testacc"

# Live integration tests with group filtering support
# Usage: make live-test TF_LIVE_TEST_GROUPS="core,kafka" or make live-test (for all)
.PHONY: live-test
live-test:
	@echo "Running live integration tests against Confluent Cloud..."
	@if [ -z "$(TF_LIVE_TEST_GROUPS)" ]; then \
		echo "Running ALL live tests..."; \
		TF_ACC=1 TF_ACC_PROD=1 $(GOCMD) test ./internal/provider/ -v -run=".*Live$$" -tags="live_test,all" -timeout 1440m; \
	else \
		echo "Running live tests for groups: $(TF_LIVE_TEST_GROUPS)"; \
		TAGS="live_test"; \
		for group in $$(echo "$(TF_LIVE_TEST_GROUPS)" | tr ',' ' '); do \
			TAGS="$$TAGS,$$group"; \
		done; \
		TF_ACC=1 TF_ACC_PROD=1 $(GOCMD) test ./internal/provider/ -v -run=".*Live$$" -tags="$$TAGS" -timeout 1440m; \
	fi
	@echo "Finished running live integration tests against Confluent Cloud"

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

.PHONY: live-test-essential
live-test-essential:
	@$(MAKE) live-test TF_LIVE_TEST_GROUPS="core,kafka"

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
