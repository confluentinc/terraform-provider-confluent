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

# Pact variables
export PACT_DO_NOT_TRACK ?= true
# This variable is controlled by the CI environment
# Leaving the name here for documentation purposes
# export PACT_BROKER_URL ?= 
# pact cli hardcodes the path where the binaries are installed as ${CURDIR}/pact
# so let's install the rest of the tooling there too
PACT_BIN_PATH ?= $(CURDIR)/pact/bin
# where to look for pact tests
PACT_TEST_DIR ?= $(CURDIR)/internal/provider
# where to save pact files (contracts)
PACT_FILES_DIR ?= $(CURDIR)/internal/provider/pacts
# add pact tools to path
export PATH := $(PATH):$(PACT_BIN_PATH)
# The name used to publish pacts
PACTICIPANT_NAME := terraform-provider-confluent
# Which environment to release to.
# This means to validate against a provider deployed into that environment
# TODO: can we validate against all of them? Should we?
PACT_RELEASE_ENVIRONMENT := prod
# Whether or not to publish the pact to the pact broker
# there's no point publishing pacts from e.g. patch versions if we're never going to release those
# so this variable will be set to true for major and minor versions only
PACT_SHOULD_PUBLISH ?= false
# Setting this to true will allow deploying a service even if can-i-deploy fails.
# This makes can-i-deploy command always return success, even if verifications failed.
# TODO: set to false once the provider side is ready
PACT_BROKER_CAN_I_DEPLOY_DRY_RUN ?= true
# Set this to a list of pacticipants to ignore during can-i-deploy command.
# Should only be used in either "break-glass" situations or when you're rolling back to a version
# that doesn't have a contract with that specific pacticipant yet (but has contracts with other services)
# Comma-separated list of pacticipant names.
PACT_BROKER_CAN_I_DEPLOY_IGNORE ?=

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
# PACT_SHOULD_PUBLISH determines whether to publish the contract to the pact broker.
PACT_SHOULD_PUBLISH = true
else ifeq ($(BUMP),minor)
bump := $(shell expr $(word 2,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(bump).0
PACT_SHOULD_PUBLISH = true
else ifeq ($(BUMP),patch)
bump := $(shell expr $(word 3,$(split_version)) + 1)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(word 2,$(split_version)).$(bump)
else ifeq ($(BUMP),none)
BUMPED_CLEAN_VERSION := $(word 1,$(split_version)).$(word 2,$(split_version)).$(word 3,$(split_version))
endif

BUMPED_VERSION := v$(BUMPED_CLEAN_VERSION)

RELEASE_SVG := <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="94" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="\#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="94" height="20" rx="3" fill="\#fff"/></clipPath><g clip-path="url(\#a)"><path fill="\#555" d="M0 0h49v20H0z"/><path fill="\#007ec6" d="M49 0h45v20H49z"/><path fill="url(\#b)" d="M0 0h94v20H0z"/></g><g fill="\#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="255" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="390">release</text><text x="255" y="140" transform="scale(.1)" textLength="390">release</text><text x="705" y="150" fill="\#010101" fill-opacity=".3" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text><text x="705" y="140" transform="scale(.1)" textLength="350">$(BUMPED_VERSION)</text></g> </svg>

# which version to publish pacts as.
# since publishing happens in CI in the same job that does the version bump and git tag, 
# BUMPED_VERSION points to the actual resulting tag from the CI job.
PACT_PUBLISH_VERSION := $(BUMPED_VERSION)
# which version to use when recording a release in the pact broker. 
# since release happens in a follow up job, the version has already been bumped by the time we get here,
# so VERSION corresponds to the actual tag being released.
PACT_RELEASE_VERSION := $(VERSION)

.PHONY: all
all: clean pact-install deps test testacc tools build pact-consumer-test

.PHONY: release-ci
release-ci:
ifeq ($(BRANCH_NAME), $(MASTER_BRANCH))
ifeq ($(CI),true)
	make release
endif
endif

.PHONY: release
release: get-release-image commit-release tag-release pact-consumer-publish

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

.PHONY: build-all
build-all: GOOS      = linux darwin
build-all: GOARCH    = amd64
build-all: clean ## Build binary for all OS/ARCH
	@ $(MAKE) --no-print-directory log-$
	@ ./scripts/build/build-all-osarch.sh "$(BUILD_DIR)" "$(NAME)" "$(VERSION)" "$(GOOS)" "$(GOARCH)"

.PHONY: test
test:
	$(GOCMD) test ./...

.PHONY: testacc
testacc:
	TF_LOG=debug TF_ACC=1 $(GOCMD) test $(TEST) -v $(TESTARGS) -timeout 120m
	@echo "skipping testacc"

install: build
	mkdir -p ~/.terraform.d/plugins/darwin_amd64
	cp ./bin/darwin-amd64/terraform-provider-confluent ~/.terraform.d/plugins/darwin_amd64/

.PHONY: gox
gox:
	GO111MODULE=off go get -u github.com/mitchellh/gox

.PHONY: goimports
goimports:
	GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports

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

.PHONY: release-to-public-repo
release-to-public-repo:
	@echo Latest tag version to release is: $(PUBLIC_REPO_TAG_VERSION)
	-@git remote add public https://github.com/confluentinc/terraform-provider-confluent.git
	@echo Preparing to publish the latest commits and the most recent tag to the public repository...
	git push --atomic public master $(PUBLIC_REPO_TAG_VERSION)

# Pact targets
.PHONY: show-pact
show-pact:
	@echo "PACTICIPANT_NAME:                  $(PACTICIPANT_NAME)"
	@echo "PACT_SHOULD_PUBLISH:               $(PACT_SHOULD_PUBLISH)"
	@echo "PACT_PUBLISH_VERSION:              $(PACT_PUBLISH_VERSION)"
	@echo "PACT_RELEASE_VERSION:              $(PACT_RELEASE_VERSION)"
	@echo "PACT_BROKER_URL:                   $(PACT_BROKER_URL)"
	@echo "PACT_BIN_PATH:                     $(PACT_BIN_PATH)"
	@echo "PACT_TEST_DIR:                     $(PACT_TEST_DIR)"
	@echo "PACT_FILES_DIR:                    $(PACT_FILES_DIR)"
	@echo "PACT_DO_NOT_TRACK:                 $(PACT_DO_NOT_TRACK)"
	@echo "PACT_BROKER_CAN_I_DEPLOY_DRY_RUN:  $(PACT_BROKER_CAN_I_DEPLOY_DRY_RUN)"
	@echo "PACT_BROKER_CAN_I_DEPLOY_IGNORE:   $(PACT_BROKER_CAN_I_DEPLOY_IGNORE)"


.PHONY: pact-cli-install
## Installs main pact CLI
pact-cli-install:
	@echo "Installing pact ruby CLI (latest)"
# TODO: this URL will change once we move to no internet access on Semaphore
	curl -fsSL https://raw.githubusercontent.com/pact-foundation/pact-ruby-standalone/master/install.sh | bash

.PHONY: pact-go-install
## Installs pact-go CLI and dylib.
## Requires sudo on ARM Macs.
pact-go-install:
	@PACT_BIN_PATH=$(PACT_BIN_PATH) $(CURDIR)/scripts/pact-go-install.sh

.PHONY: pact-install
## Installs all pact CLI tools
pact-install: pact-cli-install pact-go-install

.PHONY: pact-consumer-test
## Run pact consumer tests. Consumer tests generate pact files, but don't publish them.
pact-consumer-test:
	@echo "--- Running Consumer Pact tests "
	PACT_FILES_DIR=$(PACT_FILES_DIR) $(GOCMD) test -v -count=1 -tags=pact.consumer $(PACT_TEST_DIR)

# We're using `if` condition here to check if there's at least one .json file under $PACT_FILES_DIR
# cause if there are none, `pact-broker publish` would return an error.
# We have no other way of knowing if there are any consumer pacts that require publishing.
.PHONY: pact-consumer-publish
## Publish a consumer pact to the pact broker.
## Pact will be published with $(PACT_PUBLISH_VERSION).
pact-consumer-publish:
ifeq ($(PACT_SHOULD_PUBLISH),true)
	if ls $(PACT_FILES_DIR)/*.json &>/dev/null; then \
		echo "--- Publishing Consumer Pacts to the Pact Broker"; \
		$(PACT_BIN_PATH)/pact-broker publish $(PACT_FILES_DIR) \
			--consumer-app-version $(PACT_PUBLISH_VERSION) \
			--branch $(BRANCH_NAME) \
			--broker-base-url $(PACT_BROKER_URL); \
	else \
		echo "--- No pacts found under $(PACT_FILES_DIR); skip publishing"; \
	fi
else
	@echo "--- PACT_SHOULD_PUBLISH is false, skip publishing pacts"
endif

.PHONY: pact-require-environment
pact-require-environment:
# built-in `ifndef` is evaluated at parse-time
# so if PACT_RELEASE_ENVIRONMENT is defined in a different make file or simply below this target
# it would not see it.
# Shell `if` is evaluated at execution time, after all make files have been parsed,
# so doesn't matter if you define PACT_RELEASE_ENVIRONMENT before or after this target
	@if [ -z $(PACT_RELEASE_ENVIRONMENT) ]; then \
		echo "PACT_RELEASE_ENVIRONMENT is empty or not defined"; \
		exit 1; \
	fi

.PHONY: pact-require-version
pact-require-version:
	@if [ -z $(PACT_RELEASE_VERSION) ]; then \
		echo "PACT_RELEASE_VERSION is empty or not defined"; \
		exit 1; \
	fi

.PHONY: pact-release
## Record release of a $(PACTICIPANT_NAME) to $(PACT_RELEASE_ENVIRONMENT). 
## Requires $(PACT_RELEASE_ENVIRONMENT) and $(PACT_RELEASE_VERSION) to be set.
# See also: https://docs.pact.io/pact_broker/recording_deployments_and_releases
# Release is different from deploy.
# Deploy means that previous version is no longer available.
# Release means that previous version is still available until it's explicitly marked as havint its "support ended".
# In terraform provider we're using release because we officially support multiple versions.
pact-release: pact-require-environment pact-require-version
ifeq ($(CI),true)
	@echo "--- Pact Broker: record release of $(PACTICIPANT_NAME) @ $(PACT_RELEASE_VERSION) to $(PACT_RELEASE_ENVIRONMENT)"
	$(PACT_BIN_PATH)/pact-broker create-or-update-version \
		--pacticipant=$(PACTICIPANT_NAME) \
		--version=$(PACT_RELEASE_VERSION) \
		--broker-base-url=$(PACT_BROKER_URL)
	$(PACT_BIN_PATH)/pact-broker record-release \
		--pacticipant=$(PACTICIPANT_NAME) \
		--version=$(PACT_RELEASE_VERSION) \
		--environment=$(PACT_RELEASE_ENVIRONMENT) \
		--broker-base-url=$(PACT_BROKER_URL)
else
	@echo "--- Can only record deployments from CI"
	@echo "--- Exiting"
endif

# can-i-deploy applies for both deploy and release.
.PHONY: pact-can-i-deploy
## Check if you can deploy a $(PACTICIPANT_NAME) to $(PACT_RELEASE_ENVIRONMENT). 
## Requires $(PACT_RELEASE_ENVIRONMENT) and $(PACT_RELEASE_VERSION) to be set.
# Will retry 30 times with 30 seconds intervals if verification results are not yet available.
# Will allow deployment if the given version does not exist in the broker. 
# This is to allow rollbacks to older service versions.
pact-can-i-deploy: pact-require-environment pact-require-version
	@PACT_BIN_PATH=$(PACT_BIN_PATH) PACTICIPANT_NAME=$(PACTICIPANT_NAME) PACT_DEPLOY_VERSION=$(PACT_RELEASE_VERSION) \
	PACT_BROKER_URL=$(PACT_BROKER_URL) PACT_RELEASE_ENVIRONMENT=$(PACT_RELEASE_ENVIRONMENT) \
	PACT_BROKER_CAN_I_DEPLOY_DRY_RUN=$(PACT_BROKER_CAN_I_DEPLOY_DRY_RUN) \
	PACT_BROKER_CAN_I_DEPLOY_IGNORE=$(PACT_BROKER_CAN_I_DEPLOY_IGNORE) \
	$(CURDIR)/scripts/pact-can-i-deploy.sh
