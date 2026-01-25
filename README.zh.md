# API Manager (apimgr)

[English version](README.md)

## 目录
- [项目简介](#项目简介)
- [特性](#-特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [配置文件](#配置文件)
- [命令详解](#命令详解)
- [环境变量](#环境变量)
- [使用示例](#使用示例)
- [Shell 集成](#shell-集成)
- [从旧版本迁移](#从旧版本迁移)
- [故障排查](#故障排查)
- [技术架构](#技术架构)
- [文档](#文档)
- [贡献](#贡献)
- [许可证](#许可证)
- [支持](#支持)

## 项目简介

一个现代化、功能丰富的命令行工具，用于管理 API 配置和测试连通性。apimgr 通过集中式配置管理、安全存储和无缝 shell 集成，简化了多 API 提供商的使用。

**核心亮点：**
- 🎨 **精美的 TUI**: 功能完整的终端用户界面，支持键盘快捷键
- 🔄 **多提供商**: 支持 Anthropic、OpenAI 和自定义 API 提供商
- 🔐 **安全**: 加密密钥存储和文件权限控制
- 🚀 **快速**: 通过 shell 集成实现即时配置切换
- 🧪 **测试**: 内置连通性和兼容性测试功能
- 📦 **跨平台**: 原生支持 macOS、Linux 和 Windows

> **注意**: 本项目目前专注于 Claude Code 集成，未来计划支持更多 AI 编码工具。

## ✨ 特性

### 核心功能

- 📁 **多配置管理**: 使用 JSON 文件存储多组 API 配置
- ⚡ **快速切换**: 使用 `apimgr switch <alias>` 快速切换配置
- 🔄 **自动应用**: 配置切换后自动生成环境变量脚本
- 💾 **持久化**: 配置自动保存，新终端自动加载活动配置
- 🔒 **安全显示**: API 密钥脱敏显示，保护敏感信息
- ✅ **输入验证**: URL 格式验证和必填字段检查
- 🛡️ **错误处理**: 完整的错误处理和用户友好提示
- 🌐 **跨平台**: 支持 macOS、Linux 和 Windows

### 高级特性

- 🖥️ **交互式 TUI**: 功能完整的终端用户界面，支持键盘导航（直接运行 `apimgr` 即可启动）
- 📦 **多提供商支持**: Anthropic、OpenAI 及自定义提供商
- 📡 **连通性测试**: 使用 `apimgr ping` 测试 API 配置的连通性
- 🔄 **配置同步**: 同步配置到 Claude Code 等工具
- 🎯 **本地切换**: 使用 `-l/--local` 参数仅在当前 shell 生效
- 📝 **交互式编辑**: 支持交互式添加和编辑配置
- 📊 **状态检查**: 查看全局和当前 shell 的配置状态
- 📂 **XDG 规范支持**: 遵循 Linux 上的 XDG Base Directory Specification

## 安装

### 方法 1: Go install (推荐)

```bash
go install github.com/ccasJay/apimgr@latest
```

### 方法 2: 二进制下载

从 [GitHub Releases](https://github.com/ccasJay/apimgr/releases) 下载适合您系统的预编译二进制文件。

### 方法 3: 从源码构建

```bash
git clone https://github.com/ccasJay/apimgr.git
cd apimgr
go build -o apimgr .

# 可选：安装到系统路径
sudo mv apimgr /usr/local/bin/
```

### 方法 4: 使用 Makefile

```bash
# 构建
make build

# 安装到系统路径
make install
```

## 快速开始

### TUI 模式（推荐）

直接运行 `apimgr` 即可启动交互式 TUI 界面：

```bash
apimgr
```

TUI 快捷键：
| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 上下移动 |
| `g/G` | 跳到顶部/底部 |
| `Enter` | 查看详情 |
| `s` | 本地切换配置 (Claude Code) |
| `S` | 全局切换配置 |
| `a` | 添加配置 |
| `e` | 编辑配置 |
| `d` | 删除配置 |
| `p` | 连接测试 |
| `t` | 兼容性测试 |
| `m` | 切换模型 |
| `?` | 帮助 |
| `q` | 退出 |

### 命令行模式

```bash
# 1. 初始化配置目录和 shell 集成
apimgr enable

# 2. 添加 API 配置
apimgr add my-config --sk sk-xxxxxxxx --url https://api.anthropic.com --model claude-3

# 3. 切换到新配置
apimgr switch my-config

# 4. 验证配置
apimgr status

# 5. 测试连通性
apimgr ping              # 基本连通性测试
apimgr ping -T           # 兼容性测试（自动检测 provider）
apimgr ping -T --stream  # 测试流式响应兼容性

# 6. 列出所有配置
apimgr list
```

### 基本命令

```bash
# 初始化配置目录（首次使用必须）
apimgr enable

# 添加配置
apimgr add --alias my-config --sk sk-xxxxxxxx --url https://api.anthropic.com --model claude-3
# 或使用 auth token
apimgr add --alias my-config --ak <auth-token> --url https://api.anthropic.com --model claude-3

# 列出所有配置（* 表示当前活动配置）
apimgr list

# 切换配置
apimgr switch <别名>

# 显示当前配置
apimgr status

# 编辑配置
apimgr edit <别名> [--sk <new-key>] [--ak <new-token>] [--url <new-url>] [--model <new-model>]

# 删除配置
apimgr remove <别名>
```

### 交互式添加

```bash
# 完全交互式模式
apimgr add

# API 密钥预设交互式
apimgr add --sk <your-api-key>

# 认证令牌预设交互式
apimgr add --ak <your-auth-token>
```

### 配置文件

#### 配置路径

- **默认路径**: `~/.config/apimgr/config.json` (遵循 XDG 规范)
- **旧版本兼容**: `~/.apimgr.json` (会自动迁移到新路径)
- **自定义路径**: 可以通过 `XDG_CONFIG_HOME` 环境变量自定义配置目录：

  ```bash
  XDG_CONFIG_HOME=~/.myconfig apimgr add my-config --sk sk-xxx...
  ```

格式如下：

```json
{
  "active": "my-config",
  "configs": [
    {
      "alias": "my-config",
      "api_key": "sk-xxxxxxxx",
      "auth_token": "",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3",
      "provider": "anthropic"
    }
  ]
}
```

### Provider 自动检测

当配置中未显式设置 `provider` 字段时，apimgr 会根据 base URL 自动检测 provider 类型：

| URL 模式 | 检测到的 Provider |
|----------|-------------------|
| `*api.anthropic.com*` | anthropic |
| `*api.openai.com*` | openai |
| 其他 URL | anthropic (默认) |

这意味着在添加使用标准 API URL 的配置时，可以省略 `provider` 字段：
```bash
# Provider 将自动检测为 "anthropic"
apimgr add my-anthropic --sk sk-ant-... --url https://api.anthropic.com

# Provider 将自动检测为 "openai"
apimgr add my-openai --sk sk-... --url https://api.openai.com
```
```

### 环境变量

切换配置时会生成 `active.env` 文件，包含以下环境变量：

- `ANTHROPIC_API_KEY`: API 密钥
- `ANTHROPIC_AUTH_TOKEN`: 认证令牌（二选一）
- `ANTHROPIC_BASE_URL`: API 基础 URL（可选）
- `ANTHROPIC_MODEL`: 模型名称（可选）
- `APIMGR_ACTIVE`: 当前活动配置别名

### 使用示例

```bash
# 1. 首次安装
apimgr enable
# 输出：
# 📁 Creating XDG-compliant directory structure...
# ✅ Configuration ready at ~/.config/apimgr/config.json
#
# 📝 Checking shell configuration...
# ⚠️  Shell integration not configured. Add this line to your shell config:
#
#     [[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env

# 2. 添加 shell 集成并重载
echo '[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env' >> ~/.zshrc
source ~/.zshrc

# 3. 添加开发环境配置
apimgr add --alias dev --sk sk-dev123 --model claude-3-opus
# 输出：
# ✅ Configuration updated - active.env regenerated
# 已添加配置: dev

# 4. 添加生产环境配置
apimgr add --alias prod --sk sk-prod456 --model claude-3
# 输出：
# ✅ Configuration updated - active.env regenerated
# 已添加配置: prod

# 5. 查看所有配置
apimgr list
# 输出：
#   dev: API Key: sk-d****123 (URL: https://api.anthropic.com, Model: claude-3-opus)
# * prod: API Key: sk-p****456 (URL: https://api.anthropic.com, Model: claude-3)

# 6. 切换到开发环境
apimgr switch dev
# 输出：
# ✅ Configuration updated - active.env regenerated
# 已切换到配置: dev

# 7. 验证当前配置
apimgr status
# 输出：
# 当前激活的配置:
#   别名: dev
#   API Key: sk-d****123
#   Base URL: https://api.anthropic.com
#   Model: claude-3-opus

# 8. 验证环境变量
echo $ANTHROPIC_API_KEY
# 输出: sk-dev123

# 9. 编辑配置
apimgr edit dev --model claude-3.5-sonnet
# 输出：
# ✅ Configuration updated - active.env regenerated
# 配置已更新: dev

# 10. 删除配置
apimgr remove test-config
# 输出：
# ✅ Configuration updated - active.env regenerated
# 已删除配置: test-config
```

## 命令详解

### TUI 模式

```bash
apimgr            # 启动交互式 TUI 界面
```

TUI 提供完整的图形化终端界面，支持：
- 配置列表浏览和详情查看
- 键盘快捷键操作
- 添加、编辑、删除配置
- 连接测试和兼容性测试
- 模型切换
- 帮助面板

### enable

初始化配置目录和 shell 集成

```bash
apimgr enable
```

功能：

- 创建 XDG 标准目录结构 (`~/.config/apimgr/`)
- 从旧版本自动迁移配置文件
- 生成 `active.env` 文件
- 提供 shell 配置指导

### add

添加新的 API 配置

```bash
# 命令行模式
apimgr add <alias> [--sk <api-key>] [--ak <auth-token>] [--url <base-url>] [--model <model>]

# 交互式模式
apimgr add
apimgr add --sk <api-key>
apimgr add --ak <auth-token>
```

### list

列出所有已保存的配置，`*` 表示当前活动配置

```bash
apimgr list
```

### switch

切换到指定配置

```bash
apimgr switch <别名>
```

### status

显示当前激活的配置信息

```bash
apimgr status
```

### edit

编辑指定配置

```bash
apimgr edit <alias> [--sk <new-key>] [--ak <new-token>] [--url <new-url>] [--model <new-model>]
```

### remove

删除指定的配置

```bash
apimgr remove <别名>
```

### ping

测试 API 连通性和兼容性

```bash
# 基本连通性测试
apimgr ping [alias]          # 测试指定或当前活动配置
apimgr ping -u URL           # 测试自定义 URL
apimgr ping -t 30s           # 自定义超时时间
apimgr ping -j               # JSON 格式输出

# 兼容性测试模式 (-T)
apimgr ping -T               # 测试 API 兼容性（自动检测 provider）
apimgr ping -T --stream      # 测试流式响应兼容性
apimgr ping -T -p /custom    # 使用自定义端点路径
apimgr ping -T -v            # 详细输出（显示请求/响应内容）
```

`-T` 标志启用兼容性测试模式，功能包括：
- 发送真实的 chat completion 请求验证 API 格式
- 根据 base URL 自动检测 provider 类型（Anthropic/OpenAI）
- 验证响应结构是否符合 Claude Code 的期望
- 使用 `--stream` 标志测试流式响应支持

## Shell 集成

### 启用

添加以下行到你的 `~/.zshrc` 或 `~/.bashrc`:

```bash
[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env
```

### 工作原理

- `active.env` 文件会在每次配置变更时自动更新
- 只需要在 shell 配置中添加一行引用
- 配置切换后，新终端或重新加载的 shell 会自动使用新配置
- 无需重启终端，只需重新加载 shell 配置或打开新终端

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
go install github.com/ccasJay/apimgr@latest
# 或重新编译
go build -o apimgr .

# 2. 运行启用命令（自动迁移配置）
apimgr enable

# 3. 添加 shell 集成
echo '[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env' >> ~/.zshrc
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
- **配置应用**: 切换配置后自动生成 `active.env` 文件
- **Shell 集成**: 使用简单的 `source` 命令引用 `active.env`

## 故障排查

### 常见错误

#### 超时错误
**症状**: 运行 `apimgr ping` 时连接超时

**解决方案**: 
```bash
# 使用 -t 标志增加超时时间
apimgr ping -t 30s

# 检查网络连接
curl -I https://api.anthropic.com
```

#### 连接被拒绝
**症状**: 出现 "Connection refused" 错误

**解决方案**:
- 检查 API 服务器是否运行且可访问
- 验证 base URL 是否正确
- 检查防火墙设置
- 尝试在浏览器或使用 curl 访问 URL

#### DNS 解析失败
**症状**: 出现 "No such host" 或 DNS 解析错误

**解决方案**:
- 验证域名拼写
- 检查 DNS 设置
- 尝试使用不同的 DNS 服务器
- 测试: `nslookup api.anthropic.com`

#### URL 无效
**症状**: 添加配置时出现 "Invalid URL" 错误

**解决方案**: 确保 URL 包含协议 (http:// 或 https://)
```bash
# ✗ 错误
apimgr add config --url api.anthropic.com

# ✓ 正确
apimgr add config --url https://api.anthropic.com
```

#### 配置切换后没有生效
**症状**: 切换后环境变量未更新

**解决方案**:
```bash
# 1. 检查 active.env 文件是否存在
ls -la ~/.config/apimgr/active.env

# 2. 确认 shell 集成已添加
grep apimgr ~/.zshrc  # 或 ~/.bashrc

# 3. 如果未找到，添加它：
echo '[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env' >> ~/.zshrc

# 4. 重新加载 shell 配置
source ~/.zshrc

# 5. 验证环境变量
echo $ANTHROPIC_API_KEY
```

#### 命令未找到
**症状**: `apimgr: command not found`

**解决方案**:
```bash
# 确认 apimgr 在 PATH 中
which apimgr

# 如果未找到，移动到系统目录
sudo cp apimgr /usr/local/bin/

# 或临时添加当前目录到 PATH
export PATH=$PATH:$(pwd)
```

### 获取帮助

使用详细输出标志：
```bash
apimgr ping -v      # 详细输出，包含请求/响应详情
apimgr ping -j      # JSON 格式输出，便于解析和自动化
apimgr ping -T -v   # 详细的兼容性测试，包含完整 API 详情
```

**示例 JSON 输出：**
```json
{
  "url": "https://api.anthropic.com",
  "statusCode": 200,
  "statusText": "OK",
  "requestMethod": "HEAD",
  "durationMs": 123,
  "timeoutMs": 10000,
  "success": true
}
```

更多帮助：
- 运行 `apimgr --help` 或 `apimgr <命令> --help`
- 查看[快速开始指南](QUICKSTART.zh.md)
- 在 GitHub 上[提交 issue](https://github.com/ccasJay/apimgr/issues)

### 配置文件问题

```bash
# 检查配置文件语法
cat ~/.config/apimgr/config.json | jq .

# 如果损坏，恢复备份或重新创建
mv ~/.config/apimgr/config.json ~/.config/apimgr/config.json.bak
echo '{"active":"","configs":[]}' > ~/.config/apimgr/config.json
```

### 权限问题

```bash
# 修复目录权限
chmod 755 ~/.config/apimgr
chmod 600 ~/.config/apimgr/config.json
```

## 技术架构

- **语言**: Go 1.21+
- **CLI 框架**: Cobra
- **配置格式**: JSON
- **存储位置**: `~/.config/apimgr/` (XDG 规范)
- **配置管理**: 直接文件读写 + 活动环境文件生成

## 开发

```bash
# 构建
go build -o apimgr .

# 安装到系统
sudo cp apimgr /usr/local/bin/apimgr

# 运行测试
go test ./...

# 清理
go clean
```

## 文档

- [快速开始指南](QUICKSTART.zh.md) - 快速上手基本使用
- [架构指南](ARCHITECTURE.md) - 技术架构和设计细节（英文）
- [贡献指南](CONTRIBUTING.zh.md) - 如何为项目做贡献
- [行为准则](CODE_OF_CONDUCT.zh.md) - 社区规范
- [安全政策](SECURITY.md) - 安全实践和漏洞报告（英文）
- [更新日志](CHANGELOG.md) - 版本历史和发布说明（英文）
- [代码审核报告](CODE_AUDIT_REPORT.md) - 详细代码质量分析

## 贡献

欢迎贡献！请阅读[贡献指南](CONTRIBUTING.zh.md)了解开发流程和行为准则。

## 许可证

MIT - 详见 [LICENSE](LICENSE) 文件

## 支持

如有问题、功能请求或疑问，请在 GitHub 上[提交 issue](https://github.com/ccasJay/apimgr/issues)。
