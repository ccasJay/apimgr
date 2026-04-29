#!/usr/bin/env bash
set -euo pipefail

# 构建并安装apimgr到系统路径
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o apimgr .

echo "✅ 编译成功"

# 尝试使用sudo安装到/usr/local/bin
if sudo install -m 0755 apimgr /usr/local/bin/apimgr 2>/dev/null; then
    echo "✅ 已更新 /usr/local/bin/apimgr"
else
    echo "⚠️  无法更新 /usr/local/bin/apimgr (权限不足)"
    echo "请手动运行: sudo install -m 0755 apimgr /usr/local/bin/apimgr"
fi

echo "✅ 项目目录中的 apimgr 已更新"
echo "如需启用 shell 集成，请运行: apimgr shell-install"
