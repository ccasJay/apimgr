# 快速开始

[English version](QUICKSTART.md)

## 安装

### 方法1：二进制下载
从 [GitHub Releases](https://github.com/ccasJay/apimgr/releases) 下载适合您系统的二进制文件。

### 方法2：源码编译
```bash
git clone https://github.com/ccasJay/apimgr.git
cd apimgr
go build
sudo mv apimgr /usr/local/bin/
```

### 方法3：Go install
```bash
go install github.com/ccasJay/apimgr@latest
```

## Shell 集成

安装二进制后，如果希望 `apimgr switch` 自动更新当前 shell 环境变量，请启用 shell 集成：

```bash
apimgr shell-install
source ~/.zshrc  # 或 source ~/.bashrc
```

旧命令 `apimgr install` 仍可作为别名使用，但推荐使用 `shell-install`。

## 基本使用

### 1. 添加配置

#### 交互式添加
```bash
apimgr add
```

#### 命令行直接添加
```bash
# 添加Anthropic配置
apimgr add my-anthropic --sk sk-ant-api-key --url https://api.anthropic.com --model claude-3-sonnet

# 添加OpenAI配置
apimgr add my-openai --sk sk-oo-api-key --url https://api.openai.com/v1 --model gpt-4 --provider openai
```

### 2. 查看配置列表

```bash
apimgr list
```

### 3. 切换配置

```bash
apimgr switch my-openai
```

### 4. 测试连通性

```bash
# 测试当前配置
apimgr ping

# 测试特定配置
apimgr ping my-anthropic

# 测试自定义URL
apimgr ping -u https://api.example.com
```

### 5. 查看状态

```bash
apimgr status
```

### 6. 编辑配置

```bash
apimgr edit my-openai
```

### 7. 删除配置

```bash
apimgr remove my-anthropic
```

## Shell 集成

要使配置在所有终端会话中自动生效：

### Bash
```bash
echo 'source ~/.config/apimgr/active.env' >> ~/.bashrc
```

### Zsh
```bash
echo 'source ~/.config/apimgr/active.env' >> ~/.zshrc
```

### Fish
```bash
echo 'source ~/.config/apimgr/active.env' >> ~/.config/fish/config.fish
```

重新加载配置：
```bash
source ~/.bashrc  # 或对应Shell的配置文件
```

## 高级功能

### 配置同步

```bash
# 同步到环境变量
apimgr sync env

# 同步到特定工具
apimgr sync tool-name
```

### 批量操作

```bash
# 批量测试所有配置
apimgr ping --all
```

### JSON 输出

```bash
apimgr list --json
apimgr ping --json
```

## 帮助与支持

```bash
# 查看帮助
apimgr --help

# 查看特定命令帮助
apimgr add --help
```

## 常见问题

### 配置不生效
```bash
# 检查Shell集成
apimgr doctor

# 手动加载配置
source ~/.config/apimgr/active.env
```

### 连接失败
```bash
# 测试连通性
apimgr ping -t 30s  # 增加超时时间

# 检查网络设置
curl https://api.anthropic.com  # 手动测试网络
```
