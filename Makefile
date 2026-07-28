# SPDX-FileCopyrightText: 2025 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

# Set path to commands
GO             ?= $(shell command -v go 2>/dev/null)
GOLANGCI_LINT  ?= $(shell command -v golangci-lint 2>/dev/null)
GORELEASER     ?= $(shell command -v goreleaser 2>/dev/null)
GIT            ?= $(shell command -v git 2>/dev/null)
AWK            ?= $(shell command -v awk 2>/dev/null)
REUSE          ?= $(shell command -v reuse 2>/dev/null)
# Use HOSTCMD to not conflict with Make's $(HOSTNAME)
HOSTCMD        ?= $(shell command -v hostname 2>/dev/null)
INSTALL        ?= $(shell command -v install 2>/dev/null)
SCDOC          ?= $(shell command -v scdoc 2>/dev/null)
SHELL          ?= /bin/sh
CONTAINER_PROG ?= $(shell command -v docker 2>/dev/null)

# Allow override for PR builds in goreleaser
IS_PR_BUILD ?= false

INSTALL_PROGRAM ?= $(INSTALL) -Dm755
INSTALL_DATA    ?= $(INSTALL) -Dm644

# Recursive wildcard function, obtained from https://stackoverflow.com/a/18258352
#
# Arg 1: Space-separated list of directories to recurse into
# Arg 2: Space-separated list of patterns to match
rwildcard = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))

# Function to check if a command is available and error if not found
#
# Arg 1: Command path (can be a variable like $(GO) or direct path)
# Arg 2: Command name for error message
# Usage: $(call require-command-shell,$(GO),go)
define require-command-shell
@if [ -z "$(1)" ]; then \
	echo "make: *** $(2) command not found" >&2; \
	exit 1; \
fi
endef

# require-command-shell, but for Makefile-level checks
#
# Arg 1: Command path (can be a variable like $(GO) or direct path)
# Arg 2: Command name for error message
# Usage: $(call require-command-make,$(GO),go)
#
# Note: This function is intended to be used with $(eval $(call ...)).
# The $$ escaping ensures that the ifeq conditional evaluates during $(eval)
# execution (when $(1) has its actual value) rather than during define parsing
# (when $(1) is literally the text "$(1)").
define require-command-make
ifeq ($$(strip $(1)),)
$$(error '$(2)' command not found)
endif
endef

# Check that required commands are present
$(eval $(call require-command-make,$(GIT),git))
$(eval $(call require-command-make,$(HOSTCMD),hostname))

# Ensure shell is defined
ifeq ($(SHELL),)
$(error '$(SHELL)' undefined.)
endif

NAME          ?= ochami
IMPORT        := github.com/OpenCHAMI/$(NAME)/
VERSION       ?= $(shell $(GIT) describe --tags --always --dirty --broken --abbrev=0)
TAG           ?= $(shell $(GIT) describe --tags --always --abbrev=0)
BRANCH        ?= $(shell $(GIT) branch --show-current)
BUILD         ?= $(shell $(GIT) rev-parse HEAD)
GOVER         := $(shell $(GO) env GOVERSION)
GITSTATE      := $(shell if output=$($(GIT) status --porcelain) && [ -n "$output" ]; then echo dirty; else echo clean; fi)
BUILDHOST     := $(shell $(HOSTCMD))
BUILDUSER     := $(shell whoami)
CONTAINER_TAG ?= latest
FQCN          ?= ghcr.io/openchami/$(NAME):$(CONTAINER_TAG)
LDFLAGS := -s \
	   -X '$(IMPORT)internal/version.Version=$(VERSION)' \
	   -X '$(IMPORT)internal/version.Tag=$(TAG)' \
	   -X '$(IMPORT)internal/version.Branch=$(BRANCH)' \
	   -X '$(IMPORT)internal/version.Commit=$(BUILD)' \
	   -X '$(IMPORT)internal/version.Date=$(shell date -Iseconds)' \
	   -X '$(IMPORT)internal/version.GoVersion=$(GOVER)' \
	   -X '$(IMPORT)internal/version.GitState=$(GITSTATE)' \
	   -X '$(IMPORT)internal/version.BuildHost=$(BUILDHOST)' \
	   -X '$(IMPORT)internal/version.BuildUser=$(BUILDUSER)'

CMD      := $(call rwildcard,cmd,*.go)
INTERNAL := $(call rwildcard,internal,*.go)
PKG      := $(call rwildcard,pkg,*.go)
MANSRC   := $(wildcard man/*.sc)
MANBIN   := $(subst .sc,,$(MANSRC))
MAN1BIN  := $(filter %.1,$(MANBIN))
MAN5BIN  := $(filter %.5,$(MANBIN))

HELPERS := extras/scripts/ochami-discovery-old2new.py

prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(exec_prefix)/bin
mandir      ?= $(exec_prefix)/man
libexecdir  ?= $(prefix)/libexec/$(NAME)
sharedir    ?= $(prefix)/share

.PHONY: all
all: binaries ## Build everything

.PHONY: binaries
binaries: $(NAME) ## Build binaries

.PHONY: help
help: ## Show this help
	$(call require-command-shell,$(AWK),awk)
	@$(AWK) 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m[VAR=val]... <target>\033[0m\n\nTargets:\n"} \
	/^[a-zA-Z0-9_\/.-]+:.*##/ { \
	        printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 \
	}' $(MAKEFILE_LIST)

.PHONY: container
container: ## Perform a multi-stage container build (accepts CONTAINER_PROG, CONTAINER_OPTS, CONTAINER_TAG, FQCN)
	$(call require-command-shell,$(CONTAINER_PROG),container program "$(CONTAINER_PROG)")
	$(CONTAINER_PROG) build -t $(FQCN) . $(CONTAINER_OPTS)

.PHONY: goreleaser-build
goreleaser-build: ## Run `goreleaser build` (accepts GORELEASER_OPTS)
	$(call require-command-shell,$(GO),go)
	$(call require-command-shell,$(GORELEASER),goreleaser)
	env \
		GOVERSION=$(GOVER) \
		BUILD_HOST=$(BUILDHOST) \
		BUILD_USER=$(BUILDUSER) \
		IS_PR_BUILD=$(IS_PR_BUILD) \
		$(GORELEASER) build $(GORELEASER_OPTS)

.PHONY: goreleaser-release
goreleaser-release: ## Run `goreleaser release` (accepts GORELEASER_OPTS)
	$(call require-command-shell,$(GO),go)
	$(call require-command-shell,$(GORELEASER),goreleaser)
	env \
		GOVERSION=$(GOVER) \
		BUILD_HOST=$(BUILDHOST) \
		BUILD_USER=$(BUILDUSER) \
		IS_PR_BUILD=$(IS_PR_BUILD) \
		$(GORELEASER) release $(GORELEASER_OPTS)

.PHONY: goreleaser-clean
goreleaser-clean: ## Clean Goreleaser files (remove dist/)
	$(RM) -rf dist/

.PHONY: check-reuse
check-reuse:
	$(call require-command-shell,$(REUSE),reuse)
	reuse lint --lines

.PHONY: lint
lint:
	$(call require-command-shell,$(GOLANGCI_LINT),golangci-lint)
	$(GOLANGCI_LINT) run

.PHONY: test
test: unit-test ## Run all tests

.PHONY: unit-test
unit-test: ## Run unit tests only
	$(call require-command-shell,$(GO),go)
	$(GO) test -cover -v ./...

.PHONY: clean
clean: ## Clean Go build artifacts
	$(call require-command-shell,$(GO),go)
	$(GO) clean -i -x

.PHONY: clean-man
clean-man: ## Clean generated manual pages
	rm -f $(MANBIN)

.PHONY: clean-completions
clean-completions: ## Clean generated shell completions
	rm -rf completions/

completions: $(NAME) ## Generate shell completions
	./scripts/completions.sh

.PHONY: distclean
distclean: clean clean-completions clean-man ## Clean everything (prepare for distribution)

.PHONY: install
install: install-prog install-helper install-completions install-man ## Install everything

.PHONY: install-prog
install-prog: $(NAME) ## Install program
	$(call require-command-shell,$(INSTALL),install)
	$(INSTALL_PROGRAM) $(NAME) $(DESTDIR)$(bindir)/$(NAME)

.PHONY: install-helper
install-helper: $(HELPERS) ## Install helper scripts
	$(call require-command-shell,$(INSTALL),install)
	for h in $(HELPERS); do \
		$(INSTALL_PROGRAM) "$$h" "$(DESTDIR)$(libexecdir)/$$(basename $$h)"; \
	done

.PHONY: install-completions
install-completions: completions ## Install shell completions
	$(call require-command-shell,$(INSTALL),install)
	$(INSTALL_DATA) ./completions/ochami.bash $(DESTDIR)$(sharedir)/bash-completion/completions/ochami
	$(INSTALL_DATA) ./completions/ochami.fish $(DESTDIR)$(sharedir)/fish/vendor_completions.d/ochami.fish
	$(INSTALL_DATA) ./completions/ochami.zsh $(DESTDIR)$(sharedir)/zsh/site-functions/_ochami

.PHONY: install-man
install-man: $(MANBIN) ## Install manual pages
	$(call require-command-shell,$(INSTALL),install)
	mkdir -p $(DESTDIR)$(mandir)/man1
	mkdir -p $(DESTDIR)$(mandir)/man5
	$(INSTALL_DATA) $(MAN1BIN) $(DESTDIR)$(mandir)/man1/
	$(INSTALL_DATA) $(MAN5BIN) $(DESTDIR)$(mandir)/man5/

.PHONY: man
man: $(MANBIN) ## Generate manual pages

man/%: man/%.sc
	$(call require-command-shell,$(SCDOC),scdoc)
	$(SCDOC) < $< > $@

.PHONY: uninstall
uninstall: uninstall-prog uninstall-completions uninstall-man ## Uninstall everything

.PHONY: uninstall-prog
uninstall-prog: ## Uninstall program
	rm -f $(DESTDIR)$(bindir)/$(NAME)

.PHONY: uninstall-completions
uninstall-completions: ## Uninstall shell completions
	rm -f $(DESTDIR)/usr/share/bash-completion/completions/ochami
	rm -f $(DESTDIR)/usr/share/fish/vendor_completions.d/ochami.fish
	rm -f $(DESTDIR)/usr/share/zsh/site-functions/_ochami

.PHONY: uninstall-man
uninstall-man: ## Uninstall manual pages
	rm -f $(foreach man1page,$(subst man/,,$(MAN1BIN)),$(DESTDIR)$(mandir)/man1/$(man1page))
	rm -f $(foreach man5page,$(subst man/,,$(MAN5BIN)),$(DESTDIR)$(mandir)/man5/$(man5page))

$(NAME): *.go $(CMD) $(INTERNAL) $(PKG)
	$(call require-command-shell,$(GO),go)
	$(GO) build -v -ldflags="$(LDFLAGS)"
