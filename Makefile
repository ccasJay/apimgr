# API Manager Makefile

build:
	go build -o apimgr .

install: build
	@echo "✅ 项目目录中的 apimgr 已更新"
	@echo "如需系统安装，请在Terminal中运行: sudo cp apimgr /usr/local/bin/apimgr"

upgrade: build
	@echo "✅ 项目目录中的 apimgr 已更新"
	@echo "如需系统安装，请在Terminal中运行: sudo cp apimgr /usr/local/bin/apimgr"

uninstall:
	sudo rm -f /usr/local/bin/apimgr
	@echo "✅ 已卸载 /usr/local/bin/apimgr"

run: build
	./apimgr

clean:
	rm -f apimgr
	sudo rm -f /usr/local/bin/apimgr 2>/dev/null || true

.PHONY: build install install-local uninstall run clean