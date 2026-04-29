# API Manager (apimgr)

[English version](README.md)

一个用 Go 语言开发的命令行工具，用于管理 API 配置（密钥、基础 URL、模型等）并测试连通性，支持多提供商切换。

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

### 方法 2: 从源码构建

```bash
git clone https://github.com/ccasJay/apimgr.git
cd apimgr
go build -o apimgr .

# 可选：安装到系统路径
sudo mv apimgr /usr/local/bin/
```

### 方法 3: 使用 Makefile

```bash
# 构建
make build

# 安装到系统路径
sudo make install

# 或安装到当前用户
make install-local
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
go get -u github.com/yourusername/apimgr
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

### 配置切换后没有生效

```bash
# 检查 active.env 文件是否存在
ls -la ~/.config/apimgr/active.env

# 确认 shell 集成已添加
grep apimgr ~/.zshrc  # 或 ~/.bashrc

# 重新加载 shell 配置
source ~/.zshrc
```

### 命令未找到

```bash
# 确认 apimgr 在 PATH 中
which apimgr

# 如果未找到，添加到 PATH
export PATH=$PATH:/usr/local/bin
# 或将 apimgr 复制到 PATH 中的目录
sudo cp apimgr /usr/local/bin/
```

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

- **语言**: Go 1.24+
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

## 许可证

MIT
