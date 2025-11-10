# API Manager (apimgr)

一个用 Go 语言开发的命令行工具，用于管理 Anthropic API 密钥和模型配置的快速切换。采用守护进程架构，实现了**无需重启应用即可自动应用配置**的功能。

## ✨ 新架构特性

**v2.0 版本采用守护进程架构，配置切换立即生效，无需重启应用！**

- 🚀 **实时生效**: 配置切换后立即应用到所有新进程，无需重启终端或应用
- 🎯 **守护进程**: 后台守护进程监控配置变化，通过 Unix Socket 提供实时配置
- 📂 **XDG 规范**: 遵循 XDG Base Directory 规范，配置存储在 `~/.config/apimgr/`
- 🔄 **自动迁移**: 从旧版本自动迁移配置到新目录结构
- ⚡ **高性能**: Shell 集成包含智能缓存，减少不必要的查询
- 🛡️ **容错设计**: 多重降级机制，守护进程异常时自动回退到直接读取配置

## 功能特性

- 📁 **配置管理**: 使用 JSON 文件存储多组 API 配置
- ⚡ **快速切换**: 使用 `apimgr switch <alias>` 立即切换配置
- 🔄 **持久化**: 配置自动保存，新终端自动加载活动配置
- 🔒 **安全显示**: API 密钥脱敏显示，保护敏感信息
- ✅ **输入验证**: URL 格式验证和必填字段检查
- 🛡️ **错误处理**: 完整的错误处理和用户友好提示
- 📦 **跨平台**: 支持 macOS 和 Linux（Windows 部分支持）

## 安装

```bash
# 从源码构建
go build -o apimgr .

# 或从 GitHub Release 安装（未来支持）
brew install apimgr
```

## 使用方法

### 快速开始

```bash
# 1. 初始化新架构（首次使用或从旧版本升级）
apimgr enable

# 2. 按提示添加 shell 集成到 ~/.zshrc 或 ~/.bashrc
# 将以下行添加到你的 shell 配置文件：
# source "$HOME/.config/apimgr/shell/integration.sh"

# 3. 重新加载 shell 配置
source ~/.zshrc  # 或 source ~/.bashrc

# 4. 添加配置
apimgr add --alias my-config --key sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 5. 切换配置（立即生效！）
apimgr switch my-config

# 6. 列出所有配置
apimgr list

# 7. 显示当前配置
apimgr status
```

### 基本命令

```bash
# 初始化新架构（首次使用必须）
apimgr enable

# 添加配置
apimgr add --alias my-config --key sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 列出所有配置（* 表示当前活动配置）
apimgr list

# 切换配置（新架构下立即生效）
apimgr switch my-config

# 显示当前配置
apimgr status

# 删除配置
apimgr remove my-config

# 守护进程管理
apimgr daemon start    # 启动守护进程（通常自动启动）
apimgr daemon stop     # 停止守护进程
apimgr daemon status   # 查看守护进程状态
apimgr daemon restart  # 重启守护进程

# 禁用新架构（回退到旧版本行为）
apimgr disable         # 停止守护进程，保留配置
apimgr disable --purge # 完全清理，删除所有配置
```

### 配置文件

配置文件位于 `~/.config/apimgr/config.json`（新架构）或 `~/.apimgr.json`（旧版本兼容），格式如下：

```json
{
  "active": "my-config",
  "configs": [
    {
      "alias": "my-config",
      "api_key": "sk-xxxxxxxx",
      "auth_token": "",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3"
    }
  ]
}
```

新架构目录结构：
```
~/.config/apimgr/
├── config.json           # 配置文件
├── daemon.pid           # 守护进程 PID
├── daemon.sock          # Unix Socket 文件
└── shell/
    └── integration.sh   # Shell 集成脚本
```

### 环境变量

切换配置时会输出以下环境变量：

- `ANTHROPIC_API_KEY`: API 密钥
- `ANTHROPIC_AUTH_TOKEN`: 认证令牌（二选一）
- `ANTHROPIC_BASE_URL`: API 基础 URL（可选）
- `ANTHROPIC_MODEL`: 模型名称（可选）
- `APIMGR_ACTIVE`: 当前活动配置别名

### 使用示例

```bash
# 1. 首次安装（启用新架构）
apimgr enable
# 输出：
# ✓ 创建目录 ~/.config/apimgr
# ✓ 迁移配置文件从 ~/.apimgr.json 到 ~/.config/apimgr/config.json
# ✓ 生成 shell 集成脚本
# 
# 请将以下行添加到你的 ~/.zshrc:
#   source "$HOME/.config/apimgr/shell/integration.sh"

# 2. 添加 shell 集成并重载
echo 'source "$HOME/.config/apimgr/shell/integration.sh"' >> ~/.zshrc
source ~/.zshrc

# 3. 添加开发环境配置
apimgr add --alias dev --key sk-dev123 --url https://api.anthropic.com --model claude-3-opus

# 4. 添加生产环境配置
apimgr add --alias prod --key sk-prod456 --url https://api.anthropic.com --model claude-3

# 5. 查看所有配置
apimgr list
# 输出：
# * dev: API Key: sk-d****123 (URL: https://api.anthropic.com, Model: claude-3-opus)
#   prod: API Key: sk-p****456 (URL: https://api.anthropic.com, Model: claude-3)

# 6. 切换到生产环境（立即生效！）
apimgr switch prod
# 守护进程自动启动并检测到配置变化

# 7. 验证当前配置
apimgr status
# 输出：
# 当前激活的配置:
#   别名: prod
#   API Key: sk-p****456
#   Base URL: https://api.anthropic.com
#   Model: claude-3

# 8. 新开终端或运行新进程，自动使用 prod 配置
echo $ANTHROPIC_API_KEY
# 输出: sk-prod456

# 9. 查看守护进程状态
apimgr daemon status
# 输出: 守护进程正在运行 (PID: 12345)
```

## 命令详解

### enable
启用新的守护进程架构，初始化目录结构并迁移配置

```bash
apimgr enable
```

功能：
- 创建 XDG 标准目录结构 (`~/.config/apimgr/`)
- 从旧版本自动迁移配置文件
- 生成 shell 集成脚本
- 提供 shell 配置指导

### disable
禁用守护进程架构，可选择性清理配置

```bash
apimgr disable         # 仅停止守护进程
apimgr disable --purge # 完全删除配置和目录
```

### add
添加新的 API 配置

```bash
apimgr add --alias <别名> --key <API密钥> [--url <基础URL>] [--model <模型>]
# 或使用 auth token
apimgr add --alias <别名> --ak <认证令牌> --url <基础URL> [--model <模型>]
```

### list
列出所有已保存的配置，`*` 表示当前活动配置

```bash
apimgr list
```

### switch
切换到指定配置（新架构下立即生效）

```bash
apimgr switch <别名>
```

新架构特性：
- 配置切换立即生效，无需重启应用
- 守护进程自动检测配置变化
- 所有新进程自动使用新配置
- 支持多终端同步

### status
显示当前激活的配置信息

```bash
apimgr status
```

### remove
删除指定的配置

```bash
apimgr remove <别名>
```

### daemon
管理后台守护进程

```bash
apimgr daemon start    # 启动守护进程
apimgr daemon stop     # 停止守护进程
apimgr daemon status   # 查看状态
apimgr daemon restart  # 重启守护进程
```

守护进程功能：
- 监控配置文件变化（使用 fsnotify）
- 提供 Unix Socket 服务
- 自动启动（shell 集成检测并启动）
- 信号处理（SIGTERM, SIGINT, SIGHUP）

## 安全特性

- API 密钥在显示时会进行脱敏处理（如：sk-1234****5678）
- 配置文件权限设置为 0600（仅所有者可读写）
- 支持 URL 格式验证
- 完整的输入验证和错误提示

## 从旧版本迁移

如果你正在使用旧版本的 apimgr（v1.x），请按照以下步骤迁移：

### 自动迁移
```bash
# 1. 更新到新版本
go get -u github.com/yourusername/apimgr
# 或重新编译
go build -o apimgr .

# 2. 运行启用命令（自动迁移配置）
apimgr enable

# 3. 添加 shell 集成
echo 'source "$HOME/.config/apimgr/shell/integration.sh"' >> ~/.zshrc
source ~/.zshrc

# 4. 验证迁移成功
apimgr list   # 查看所有配置
apimgr status # 查看当前配置
```

### 手动迁移（如果自动迁移失败）
```bash
# 1. 创建新目录
mkdir -p ~/.config/apimgr

# 2. 复制配置文件
cp ~/.apimgr.json ~/.config/apimgr/config.json

# 3. 运行 enable 命令
apimgr enable

# 4. 更新 shell 配置
# 删除旧的 apimgr 相关配置
# 添加新的集成脚本
```

### 主要变化
- **配置位置**: 从 `~/.apimgr.json` 迁移到 `~/.config/apimgr/config.json`
- **无需重启**: 配置切换立即生效，不再需要重启应用
- **守护进程**: 后台运行守护进程监控配置变化
- **Shell 集成**: 新的集成脚本提供更好的性能和可靠性

## 故障排查

### 守护进程相关问题

**问题：配置切换后没有生效**
```bash
# 检查守护进程状态
apimgr daemon status

# 如果未运行，启动守护进程
apimgr daemon start

# 重启守护进程
apimgr daemon restart
```

**问题：守护进程无法启动**
```bash
# 检查是否有残留的 socket 文件
rm -f ~/.config/apimgr/daemon.sock
rm -f ~/.config/apimgr/daemon.pid

# 重新启动
apimgr daemon start

# 查看错误日志
tail -f /tmp/apimgr-daemon.log  # 如果启用了日志
```

### Shell 集成问题

**问题：环境变量未设置**
```bash
# 确认 shell 集成已添加
grep apimgr ~/.zshrc  # 或 ~/.bashrc

# 手动添加（如果缺失）
echo 'source "$HOME/.config/apimgr/shell/integration.sh"' >> ~/.zshrc

# 重新加载 shell
source ~/.zshrc
```

**问题：命令未找到**
```bash
# 确认 apimgr 在 PATH 中
which apimgr

# 如果未找到，添加到 PATH
export PATH=$PATH:/usr/local/bin
# 或将 apimgr 复制到 PATH 中的目录
sudo cp apimgr /usr/local/bin/
```

### 配置文件问题

**问题：配置文件损坏**
```bash
# 检查配置文件语法
cat ~/.config/apimgr/config.json | jq .

# 如果损坏，恢复备份或重新创建
mv ~/.config/apimgr/config.json ~/.config/apimgr/config.json.bak
echo '{"active":"","configs":[]}' > ~/.config/apimgr/config.json
```

### 权限问题

**问题：无法创建或访问配置文件**
```bash
# 修复目录权限
chmod 755 ~/.config/apimgr
chmod 644 ~/.config/apimgr/config.json
chmod 666 ~/.config/apimgr/daemon.sock  # Socket 需要读写权限
```

### 常见错误信息

- **"守护进程未运行"**: 运行 `apimgr daemon start`
- **"无法连接到 socket"**: 检查守护进程状态，可能需要重启
- **"配置文件不存在"**: 运行 `apimgr enable` 初始化
- **"权限被拒绝"**: 检查文件和目录权限

## 技术架构

- **语言**: Go 1.21+
- **CLI 框架**: Cobra
- **配置格式**: JSON
- **存储位置**: `~/.config/apimgr/` (XDG 规范)
- **进程通信**: Unix Socket
- **文件监控**: fsnotify
- **架构模式**: 守护进程 + Shell 集成

## 开发

```bash
# 构建（推荐使用Makefile）
make install

# 或者手动构建和安装
go build -o apimgr .
sudo cp apimgr /usr/local/bin/apimgr

# 运行测试
go test ./...

# 清理
make clean
```

## 许可证

MIT