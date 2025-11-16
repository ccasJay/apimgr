# API Manager Makefile

build:
	go build -o apimgr .

install: build
	@echo "✅ apimgr in project directory has been updated"
	@echo "To install system-wide, run in Terminal: sudo cp apimgr /usr/local/bin/apimgr"

upgrade: build
	@echo "✅ apimgr in project directory has been updated"
	@echo "To install system-wide, run in Terminal: sudo cp apimgr /usr/local/bin/apimgr"

uninstall:
	sudo rm -f /usr/local/bin/apimgr
	@echo "✅ /usr/local/bin/apimgr has been uninstalled"

run: build
	./apimgr

clean:
	rm -f apimgr
	sudo rm -f /usr/local/bin/apimgr 2>/dev/null || true

.PHONY: build install install-local uninstall run clean