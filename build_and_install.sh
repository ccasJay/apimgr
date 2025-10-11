#!/bin/bash

# 构建并安装apimgr到系统路径
go build -o apimgr .

if [ $? -eq 0 ]; then
    echo "✅ 编译成功"
    # 尝试使用sudo安装到/usr/local/bin
    if sudo cp apimgr /usr/local/bin/apimgr 2>/dev/null; then
        echo "✅ 已更新 /usr/local/bin/apimgr"
    else
        echo "⚠️  无法更新 /usr/local/bin/apimgr (权限不足)"
        echo "请手动运行: sudo cp apimgr /usr/local/bin/apimgr"
    fi

    # 同时更新项目目录中的副本
    echo "✅ 项目目录中的 apimgr 已更新"
else
    echo "❌ 编译失败"
    exit 1
fi