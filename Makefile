
# Set path to commands
GO      ?= $(shell command -v go)
GIT     ?= $(shell command -v git)
# Use HOSTCMD to not conflict with Make's $(HOSTNAME)
HOSTCMD ?= $(shell command -v hostname)
INSTALL ?= $(shell command -v install)
SCDOC   ?= $(shell command -v scdoc)
SHELL   ?= /bin/sh

INSTALL_PROGRAM ?= $(INSTALL) -Dm755
INSTALL_DATA    ?= $(INSTALL) -Dm644

IMPORT := github.com/OpenCHAMI/ochami/

prefix      ?= /usr/local
exec_prefix ?= $(prefix)
bindir      ?= $(exec_prefix)/bin
mandir      ?= $(exec_prefix)/man

# Check that commands are present
ifeq ($(GO),)
$(error go command not found.)
endif
ifeq ($(GIT),)
$(error git command not found.)
endif
ifeq ($(HOSTCMD),)
$(error hostname command not found.)
endif
ifeq ($(INSTALL),)
$(error install command not found.)
endif
ifeq ($(SCDOC),)
$(error scdoc command not found.)
endif
ifeq ($(SHELL),)
$(error '$(SHELL)' command not found.)
endif

# Recursive wildcard function, obtained from https://stackoverflow.com/a/18258352
#
# Arg 1: Space-separated list of directories to recurse into
# Arg 2: Space-separated list of patterns to match
rwildcard = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))

NAME      ?= ochami
VERSION   ?= $(shell $(GIT) describe --tags --always --dirty --broken --abbrev=0)
TAG       ?= $(shell $(GIT) describe --tags --always --abbrev=0)
BRANCH    ?= $(shell $(GIT) branch --show-current)
BUILD     ?= $(shell $(GIT) rev-parse HEAD)
GOVER     := $(shell $(GO) env GOVERSION)
GITSTATE  := $(shell if output=$($(GIT) status --porcelain) && [ -n "$output" ]; then echo dirty; else echo clean; fi)
BUILDHOST := $(shell $(HOSTCMD))
BUILDUSER := $(shell whoami)
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

INTERNAL := $(call rwildcard,internal,*.go)
PKG      := $(call rwildcard,pkg,*.go)
MANSRC   := $(wildcard man/*.sc)
MANBIN   := $(subst .sc,,$(MANSRC))
MAN1BIN  := $(filter %.1,$(MANBIN))
MAN5BIN  := $(filter %.5,$(MANBIN))

.PHONY: all
all: binaries

.PHONY: binaries
binaries: $(NAME)

.PHONY: clean
clean:
	$(GO) clean -i -x

.PHONY: clean-man
clean-man:
	rm -f $(MANBIN)

.PHONY: clean-completions
clean-completions:
	rm -rf completions/

completions: $(NAME)
	./scripts/completions.sh

.PHONY: distclean
distclean: clean clean-completions clean-man

.PHONY: install
install: install-prog install-completions install-man

.PHONY: install-prog
install-prog: $(NAME)
	$(INSTALL_PROGRAM) $(NAME) $(DESTDIR)$(bindir)/$(NAME)

.PHONY: install-completions
install-completions: completions
	$(INSTALL_DATA) ./completions/ochami.bash $(DESTDIR)/usr/share/bash-completion/completions/ochami
	$(INSTALL_DATA) ./completions/ochami.fish $(DESTDIR)/usr/share/fish/vendor_completions.d/ochami.fish
	$(INSTALL_DATA) ./completions/ochami.zsh $(DESTDIR)/usr/share/zsh/site-functions/_ochami

.PHONY: install-man
install-man: $(MANBIN)
	mkdir -p $(DESTDIR)$(mandir)/man1
	mkdir -p $(DESTDIR)$(mandir)/man5
	$(INSTALL_DATA) $(MAN1BIN) $(DESTDIR)$(mandir)/man1/
	$(INSTALL_DATA) $(MAN5BIN) $(DESTDIR)$(mandir)/man5/

.PHONY: man
man: $(MANBIN)

man/%: man/%.sc
	$(SCDOC) < $< > $@

.PHONY: uninstall
uninstall: uninstall-prog uninstall-completions uninstall-man

.PHONY: uninstall-prog
uninstall-prog:
	rm -f $(DESTDIR)$(bindir)/$(NAME)

.PHONY: uninstall-completions
uninstall-completions:
	rm -f $(DESTDIR)/usr/share/bash-completion/completions/ochami
	rm -f $(DESTDIR)/usr/share/fish/vendor_completions.d/ochami.fish
	rm -f $(DESTDIR)/usr/share/zsh/site-functions/_ochami

.PHONY: uninstall-man
uninstall-man:
	rm -f $(foreach man1page,$(subst man/,,$(MAN1BIN)),$(DESTDIR)$(mandir)/man1/$(man1page))
	rm -f $(foreach man5page,$(subst man/,,$(MAN5BIN)),$(DESTDIR)$(mandir)/man5/$(man5page))

$(NAME): *.go cmd/*.go $(INTERNAL) $(PKG)
	$(GO) build -v -ldflags="$(LDFLAGS)"
