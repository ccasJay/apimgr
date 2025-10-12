# API Manager (apimgr)

一个用 Go 语言开发的命令行工具，用于管理 Anthropic API 密钥和模型配置的快速切换。

## ⚠️ 重要提示

**切换配置后需要重启应用才能生效！**

- 如果你在使用 Claude Code (Factory)，切换配置后需要**重启 Factory** 才能使用新配置
- 已运行的进程在启动时读取环境变量，之后修改不会影响它们
- 新打开的终端会自动加载活动配置

## 功能特性

- 📁 **配置管理**: 使用 JSON 文件存储多组 API 配置
- ⚡ **快速切换**: 安装后直接使用 `apimgr switch <alias>` 切换配置
- 🔄 **持久化**: 配置自动保存，新终端自动加载活动配置
- 🔒 **安全显示**: API 密钥脱敏显示，保护敏感信息
- ✅ **输入验证**: URL 格式验证和必填字段检查
- 🛡️ **错误处理**: 完整的错误处理和用户友好提示
- 📦 **跨平台**: 支持 Windows、macOS 和 Linux

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
# 1. 安装 shell 集成（推荐）
apimgr install

# 2. 使配置生效
source ~/.zshrc  # 或 source ~/.bashrc

# 3. 添加配置
apimgr add --alias my-config --key sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 4. 切换配置（直接使用）
apimgr switch my-config

# 5. 列出所有配置
apimgr list

# 6. 显示当前配置
apimgr status
```

### 基本命令

```bash
# 添加配置
apimgr add --alias my-config --key sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 列出所有配置（* 表示当前活动配置）
apimgr list

# 切换配置（两种方式）
apimgr switch my-config              # 推荐：需要先运行 apimgr install
eval "$(apimgr switch my-config)"    # 原始方式

# 显示当前配置
apimgr status

# 删除配置
apimgr remove my-config

# 安装 shell 集成
apimgr install
```

### 配置文件

配置文件位于 `~/.apimgr.json`，格式如下：

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

### 环境变量

切换配置时会输出以下环境变量：

- `ANTHROPIC_API_KEY`: API 密钥
- `ANTHROPIC_AUTH_TOKEN`: 认证令牌（二选一）
- `ANTHROPIC_BASE_URL`: API 基础 URL（可选）
- `ANTHROPIC_MODEL`: 模型名称（可选）
- `APIMGR_ACTIVE`: 当前活动配置别名

### 使用示例

```bash
# 1. 首次安装
apimgr install
source ~/.zshrc  # 使其生效

# 2. 添加开发环境配置
apimgr add --alias dev --key sk-dev123 --url https://api.anthropic.com --model claude-3-opus

# 3. 添加生产环境配置
apimgr add --alias prod --key sk-prod456 --url https://api.anthropic.com --model claude-3

# 4. 查看所有配置
apimgr list
# 输出：
#   dev: API Key: sk-d****123 (URL: https://api.anthropic.com, Model: claude-3-opus)
#   prod: API Key: sk-p****456 (URL: https://api.anthropic.com, Model: claude-3)

# 5. 切换到开发环境
apimgr switch dev

# 6. 验证当前配置
apimgr status
# 输出：
# 当前激活的配置:
#   别名: dev
#   API Key: sk-d****123
#   Base URL: https://api.anthropic.com
#   Model: claude-3-opus

# 7. 切换到生产环境
apimgr switch prod

# 8. 新开终端会自动加载活动配置（prod）
```

## 命令详解

### install
安装 shell 集成，自动包装 `apimgr switch` 命令和自动加载配置

```bash
apimgr install
```

安装后会在 `~/.zshrc` 或 `~/.bashrc` 中添加：
- 自动加载活动配置
- `apimgr switch` 自动应用环境变量（无需 eval）

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
切换到指定配置并输出环境变量设置命令

```bash
# 方式 1：使用简化命令（推荐）
apimgr switch <别名>

# 方式 2：使用 eval
eval "$(apimgr switch <别名>)"
```

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

### load-active
加载活动配置的环境变量（通常在 shell 初始化时自动调用）

```bash
eval "$(apimgr load-active)"
```

## 安全特性

- API 密钥在显示时会进行脱敏处理（如：sk-1234****5678）
- 配置文件权限设置为 0600（仅所有者可读写）
- 支持 URL 格式验证
- 完整的输入验证和错误提示

## 技术架构

- **语言**: Go 1.25+
- **CLI 框架**: Cobra
- **配置格式**: JSON
- **存储位置**: `~/.apimgr.json`

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