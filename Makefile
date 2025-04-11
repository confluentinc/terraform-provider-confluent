### BEGIN HEADERS ###
# This block is managed by ServiceBot plugin - Make. The content in this block is created using a common
# template and configurations in service.yml.
# Modifications in this block will be overwritten by generated content in the nightly run.
# For more information, please refer to the page:
# https://confluentinc.atlassian.net/wiki/spaces/Foundations/pages/2871328913/Add+Make
SERVICE_NAME := terraform-provider-confluent-internal
SERVICE_DEPLOY_NAME := terraform-provider-confluent-internal

### END HEADERS ###
### BEGIN MK-INCLUDE UPDATE ###
### This section is managed by service-bot and should not be edited here.
### You can make changes upstream in https://github.com/confluentinc/cc-service-bot

CURL ?= curl
FIND ?= find
TAR ?= tar

# Mount netrc so curl can work from inside a container
DOCKER_NETRC_MOUNT ?= 1

GITHUB_API = api.github.com
GITHUB_MK_INCLUDE_OWNER := confluentinc
GITHUB_MK_INCLUDE_REPO := cc-mk-include
GITHUB_API_CC_MK_INCLUDE := https://$(GITHUB_API)/repos/$(GITHUB_MK_INCLUDE_OWNER)/$(GITHUB_MK_INCLUDE_REPO)
GITHUB_API_CC_MK_INCLUDE_TARBALL := $(GITHUB_API_CC_MK_INCLUDE)/tarball
GITHUB_API_CC_MK_INCLUDE_VERSION ?= $(GITHUB_API_CC_MK_INCLUDE_TARBALL)/$(MK_INCLUDE_VERSION)

MK_INCLUDE_DIR := mk-include
MK_INCLUDE_LOCKFILE := .mk-include-lockfile
MK_INCLUDE_TIMESTAMP_FILE := .mk-include-timestamp
# For optimum performance, you should override MK_INCLUDE_TIMEOUT_MINS above the managed section headers to be
# a little longer than the worst case cold build time for this repo.
MK_INCLUDE_TIMEOUT_MINS ?= 240
# If this latest validated release is breaking you, please file a ticket with DevProd describing the issue, and
# if necessary you can temporarily override MK_INCLUDE_VERSION above the managed section headers until the bad
# release is yanked.
MK_INCLUDE_VERSION ?= v0.1343.0

# Make sure we always have a copy of the latest cc-mk-include release less than $(MK_INCLUDE_TIMEOUT_MINS) old:
# Note: The simply-expanded make variable makes sure this is run once per make invocation.
UPDATE_MK_INCLUDE := $(shell \
	func_fatal() { echo "$$*" >&2; echo output here triggers error below; exit 1; } ; \
	test -z "`git ls-files $(MK_INCLUDE_DIR)`" || { \
		func_fatal 'fatal: checked in $(MK_INCLUDE_DIR)/ directory is preventing make from fetching recent cc-mk-include releases for CI'; \
	} ; \
	trap "rm -f $(MK_INCLUDE_LOCKFILE); exit" 0 2 3 15; \
	waitlock=0; while ! ( set -o noclobber; echo > $(MK_INCLUDE_LOCKFILE) ); do \
	   sleep $$waitlock; waitlock=`expr $$waitlock + 1`; \
	   test 14 -lt $$waitlock && { \
	      echo 'stealing stale lock after 105s' >&2; \
	      break; \
	   } \
	done; \
	test -s $(MK_INCLUDE_TIMESTAMP_FILE) || rm -f $(MK_INCLUDE_TIMESTAMP_FILE); \
	{ test -d $(MK_INCLUDE_DIR) && test -d /proc && test -z "$(cat /proc/1/sched 2>&1 |head -n 1 |grep init)"; } || \
	test -z "`$(FIND) $(MK_INCLUDE_TIMESTAMP_FILE) -mmin +$(MK_INCLUDE_TIMEOUT_MINS) 2>&1`" || { \
	   GHAUTH=$$(grep -sq 'machine $(GITHUB_API)' ~/.netrc && echo netrc || \
	     ( command -v gh > /dev/null && gh auth status -h github.com > /dev/null && echo gh )); \
	   test -n "$$GHAUTH" || \
	     func_fatal 'error: no GitHub token available via "~/.netrc" or "gh auth status".\nFollow https://confluentinc.atlassian.net/l/cp/0WXXRLDh to setup GitHub authentication.\n'; \
	   echo "downloading $(GITHUB_MK_INCLUDE_OWNER)/$(GITHUB_MK_INCLUDE_REPO) $(MK_INCLUDE_VERSION) using $$GHAUTH" >&2; \
	   if [ "netrc" = "$$GHAUTH" ]; then \
	     $(CURL) --fail --silent --netrc --location "$(GITHUB_API_CC_MK_INCLUDE_VERSION)" --output $(MK_INCLUDE_TIMESTAMP_FILE)T --write-out '$(GITHUB_API_CC_MK_INCLUDE_VERSION): %{errormsg}\n' >&2; \
	   else \
	     gh release download --clobber --repo=$(GITHUB_MK_INCLUDE_OWNER)/$(GITHUB_MK_INCLUDE_REPO) \
	        --archive=tar.gz --output $(MK_INCLUDE_TIMESTAMP_FILE)T $(MK_INCLUDE_VERSION) >&2; \
	   fi \
	   && TMP_MK_INCLUDE_DIR=$$(mktemp -d -t cc-mk-include.XXXXXXXXXX) \
	   && $(TAR) -C $$TMP_MK_INCLUDE_DIR --strip-components=1 -zxf $(MK_INCLUDE_TIMESTAMP_FILE)T \
	   && rm -rf $$TMP_MK_INCLUDE_DIR/tests \
	   && rm -rf $(MK_INCLUDE_DIR) \
	   && mv $$TMP_MK_INCLUDE_DIR $(MK_INCLUDE_DIR) \
	   && mv -f $(MK_INCLUDE_TIMESTAMP_FILE)T $(MK_INCLUDE_TIMESTAMP_FILE) \
	   && echo 'installed cc-mk-include $(MK_INCLUDE_VERSION) from $(GITHUB_MK_INCLUDE_OWNER)/$(GITHUB_MK_INCLUDE_REPO)' >&2 \
	   || func_fatal unable to install cc-mk-include $(MK_INCLUDE_VERSION) from $(GITHUB_MK_INCLUDE_OWNER)/$(GITHUB_MK_INCLUDE_REPO) \
	   ; \
	} || { \
	   rm -f $(MK_INCLUDE_TIMESTAMP_FILE)T; \
	   if test -f $(MK_INCLUDE_TIMESTAMP_FILE); then \
	      touch $(MK_INCLUDE_TIMESTAMP_FILE); \
	      func_fatal 'unable to access $(GITHUB_MK_INCLUDE_REPO) fetch API to check for latest release; next try in $(MK_INCLUDE_TIMEOUT_MINS) minutes'; \
	   else \
	      func_fatal 'unable to access $(GITHUB_MK_INCLUDE_REPO) fetch API to bootstrap mk-include subdirectory'; \
	   fi; \
	} \
)
ifneq ($(UPDATE_MK_INCLUDE),)
    $(error mk-include update failed)
endif

# Export the (empty) .mk-include-check-FORCE target to allow users to trigger the mk-include
# download code above via make but without having to run any of the other targets, e.g. build.
.PHONY: .mk-include-check-FORCE
.mk-include-check-FORCE:
	@echo -n ""
### END MK-INCLUDE UPDATE ###
### BEGIN INCLUDES ###
# This block is managed by ServiceBot plugin - Make. The content in this block is created using a common
# template and configurations in service.yml.
# Modifications in this block will be overwritten by generated content in the nightly run. To include
# additional mk files, please add them before or after this generated block.
# For more information, please refer to the page:
# https://confluentinc.atlassian.net/wiki/spaces/Foundations/pages/2871328913/Add+Make
include ./mk-include/cc-begin.mk
include ./mk-include/cc-semver.mk
include ./mk-include/cc-semaphore.mk
include ./mk-include/cc-go.mk
include ./mk-include/cc-testbreak.mk
include ./mk-include/cc-vault.mk
include ./mk-include/cc-sonarqube.mk
include ./mk-include/cc-end.mk
### END INCLUDES ###
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
	TF_LOG=debug TF_ACC=1 $(GOCMD) test $(TEST) -v $(TESTARGS) -timeout 120m -failfast
	@echo "finished testacc"

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
