# API Manager Makefile

BINARY ?= apimgr
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
LDFLAGS ?= -s -w

.PHONY: build install install-local install-shell upgrade uninstall run clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 0755 "$(BINARY)" "$(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "✅ $(BINARY) installed to $(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "Run '$(BINARY) shell-install' to enable shell integration"

install-local: build
	install -d "$(HOME)/.local/bin"
	install -m 0755 "$(BINARY)" "$(HOME)/.local/bin/$(BINARY)"
	@echo "✅ $(BINARY) installed to $(HOME)/.local/bin/$(BINARY)"

install-shell: build
	./$(BINARY) shell-install

upgrade: install

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "✅ $(DESTDIR)$(BINDIR)/$(BINARY) has been uninstalled"

run: build
	./$(BINARY)

clean:
	rm -f "$(BINARY)"
