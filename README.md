# API Manager (apimgr)

一个用 Go 语言开发的命令行工具，用于管理 Anthropic API 密钥和模型配置的快速切换。

## 功能特性

- 📁 **配置管理**: 使用 JSON 文件存储多组 API 配置
- ⚡ **快速切换**: 通过 `eval "$(apimgr switch <alias>)"` 实现环境变量即时切换
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

### 基本命令

```bash
# 添加配置
apimgr add --alias my-config --key sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 列出所有配置
apimgr list

# 切换到指定配置（关键功能）
eval "$(apimgr switch my-config)"

# 显示当前配置
apimgr status

# 删除配置
apimgr remove my-config
```

### 配置文件

配置文件位于 `~/.apimgr.json`，格式如下：

```json
[
  {
    "alias": "my-config",
    "api_key": "sk-xxxxxxxx",
    "base_url": "https://api.anthropic.com",
    "model": "claude-3"
  }
]
```

### 环境变量

切换配置时会输出以下环境变量：

- `ANTHROPIC_API_KEY`: API 密钥
- `ANTHROPIC_API_BASE`: API 基础 URL（可选）
- `ANTHROPIC_MODEL`: 模型名称（可选）

### 使用示例

```bash
# 添加开发环境配置
apimgr add --alias dev --key sk-dev123 --url https://api.anthropic.com --model claude-3-opus

# 添加生产环境配置
apimgr add --alias prod --key sk-prod456 --url https://api.anthropic.com --model claude-3

# 查看所有配置
apimgr list

# 切换到开发环境
eval "$(apimgr switch dev)"

# 验证当前配置
apimgr status

# 切换到生产环境
eval "$(apimgr switch prod)"
```

## 命令详解

### add
添加新的 API 配置

```bash
apimgr add --alias <别名> --key <API密钥> [--url <基础URL>] [--model <模型>]
```

### list
列出所有已保存的配置

```bash
apimgr list
```

### switch
切换到指定配置并输出环境变量设置命令

```bash
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
# 构建
go build -o apimgr .

# 运行测试
go test ./...

# 清理
rm apimgr
```

## 许可证

MIT