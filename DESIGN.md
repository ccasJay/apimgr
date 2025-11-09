# apimgr 自动应用架构设计方案

## 概述

本文档描述 apimgr 的自动配置应用架构，实现**无需重启终端或应用即可应用新配置**的能力，同时保持 CLI 命令简洁和 shell 配置文件整洁。

## 设计目标

### 用户体验目标
- ✅ **零配置感知**：一次启用，永久生效
- ✅ **命令简洁**：保持现有命令风格（`apimgr switch dev`）
- ✅ **无需重启**：配置切换立即在所有终端生效
- ✅ **Shell 整洁**：`.zshrc` 只有一行引用，所有逻辑隔离

### 技术目标
- ✅ **轻量级**：守护进程内存占用 < 5MB
- ✅ **高性能**：配置查询延迟 < 1ms
- ✅ **跨平台**：支持 macOS、Linux（Windows WSL）
- ✅ **可扩展**：支持未来多服务商管理

---

## 核心架构

```
┌─────────────────────────────────────────────────────────┐
│                    apimgr CLI                           │
│  用户命令: switch/add/list/enable/disable               │
└────────────┬────────────────────────────────────────────┘
             │
             │ 写入配置文件
             ↓
┌─────────────────────────────────────────────────────────┐
│         ~/.config/apimgr/config.json                    │
│  配置存储 (迁移自 ~/.apimgr.json)                       │
└────────────┬────────────────────────────────────────────┘
             │
             │ fsnotify 监听变化
             ↓
┌─────────────────────────────────────────────────────────┐
│              apimgrd (守护进程)                         │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  • 启动方式: apimgr daemon start (自动后台)             │
│  • 监听配置文件变化并更新内存缓存                        │
│  • 通过 Unix Socket 提供环境变量查询服务                │
│  • 管理自身生命周期（PID 文件）                          │
└────────────┬────────────────────────────────────────────┘
             │
             │ Unix Socket 通信
             │ /tmp/apimgr-$UID/apimgr.sock
             ↓
┌─────────────────────────────────────────────────────────┐
│    ~/.config/apimgr/shell-integration.sh                │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  • Shell 启动时通过 Socket 加载初始配置                 │
│  • precmd/PROMPT_COMMAND 钩子动态刷新环境变量           │
│  • 命令包装器（保持 CLI 简洁性）                         │
└─────────────────────────────────────────────────────────┘
             ↑
             │ source 引用（~/.zshrc 中唯一一行）
             │
┌─────────────────────────────────────────────────────────┐
│              ~/.zshrc (用户 Shell 配置)                 │
│  [[ -f ~/.config/apimgr/shell-integration.sh ]] && \    │
│      source ~/.config/apimgr/shell-integration.sh       │
└─────────────────────────────────────────────────────────┘
```

---

## 目录结构设计

### XDG 规范目录布局

```
~/.config/apimgr/                    # 配置目录（XDG_CONFIG_HOME）
├── config.json                      # 主配置文件（迁移自 ~/.apimgr.json）
├── shell-integration.sh             # Shell 集成脚本（自动生成）
└── providers/                       # 多服务商支持（未来扩展）
    ├── anthropic.env
    ├── openai.env
    └── gemini.env

${XDG_RUNTIME_DIR}/apimgr/           # 运行时文件（通常是 /tmp/apimgr-$UID/）
├── apimgr.sock                      # Unix Domain Socket
├── daemon.pid                       # 守护进程 PID
└── daemon.log                       # 守护进程日志（可选）

~/.local/share/apimgr/               # 数据文件（可选，未来使用）
└── history.log                      # 操作历史
```

### 配置文件格式

```json
{
  "active": "dev",
  "configs": [
    {
      "alias": "dev",
      "api_key": "sk-dev-xxx",
      "auth_token": "",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3-opus"
    },
    {
      "alias": "prod",
      "api_key": "sk-prod-xxx",
      "auth_token": "",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3-sonnet"
    }
  ]
}
```

---

## 核心组件设计

### 1. 守护进程 (Daemon) - 优化版

**文件**: `internal/daemon/daemon.go`

**职责**:
- 配置文件监听（fsnotify + 去抖动）
- Unix Socket 服务器（支持多命令）
- 环境变量缓存管理（带版本控制）
- 进程生命周期管理（自动恢复）
- Socket 文件清理

**接口**:
```go
type Daemon interface {
    Start(ctx context.Context) error
    Stop() error
    Reload() error
    GetEnv() map[string]string
    GetVersion() string
}
```

**实现要点**:
```go
type daemon struct {
    configPath   string
    sockPath     string
    pidPath      string
    listener     net.Listener
    watcher      *fsnotify.Watcher
    envCache     map[string]string
    version      string  // 配置版本号（用于缓存优化）
    mu           sync.RWMutex
    debouncer    *Debouncer  // 去抖动器
}

// 去抖动器（防止频繁重载）
type Debouncer struct {
    timer *time.Timer
    mu    sync.Mutex
}

func (d *Debouncer) Debounce(duration time.Duration, fn func()) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.timer != nil {
        d.timer.Stop()
    }
    d.timer = time.AfterFunc(duration, fn)
}

// 启动流程（增强版）
func (d *daemon) Start(ctx context.Context) error {
    // 1. 清理残留Socket文件
    if err := d.cleanupStaleSocket(); err != nil {
        return err
    }
    
    // 2. 检查是否已运行
    if d.isRunning() {
        return ErrAlreadyRunning
    }
    
    // 3. 创建运行时目录
    d.ensureRuntimeDir()
    
    // 4. 加载初始配置到缓存
    d.loadConfig()
    d.updateVersion()  // 生成初始版本号
    
    // 5. 启动 Unix Socket 服务器
    d.startSocketServer()
    
    // 6. 启动配置文件监听器（带去抖动）
    d.watchConfigFile()
    
    // 7. 写入 PID 文件
    d.writePID()
    
    // 8. 注册信号处理（优雅关闭）
    d.handleSignals(ctx)
    
    // 9. 注册清理函数
    defer d.cleanup()
    
    return nil
}

// 清理残留Socket
func (d *daemon) cleanupStaleSocket() error {
    if _, err := os.Stat(d.sockPath); os.IsNotExist(err) {
        return nil  // 文件不存在，无需清理
    }
    
    // 尝试连接，如果失败说明是残留文件
    conn, err := net.Dial("unix", d.sockPath)
    if err != nil {
        // 连接失败，清理残留文件
        return os.Remove(d.sockPath)
    }
    
    // 连接成功，说明有其他守护进程在运行
    conn.Close()
    return ErrAlreadyRunning
}

// Socket 处理器（支持多命令）
func (d *daemon) handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // 读取命令
    decoder := json.NewDecoder(conn)
    encoder := json.NewEncoder(conn)
    
    var cmd string
    if err := decoder.Decode(&cmd); err != nil {
        // 兼容旧的纯文本协议
        scanner := bufio.NewScanner(conn)
        if scanner.Scan() {
            cmd = scanner.Text()
        }
    }
    
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    switch cmd {
    case "GET":
        // 返回环境变量
        encoder.Encode(d.envCache)
    case "VERSION":
        // 返回版本号（用于客户端缓存验证）
        fmt.Fprintf(conn, "%s\n", d.version)
    case "PING":
        // 健康检查
        fmt.Fprintf(conn, "PONG\n")
    case "RELOAD":
        // 强制重载配置
        go d.reloadConfig()
        fmt.Fprintf(conn, "OK\n")
    default:
        fmt.Fprintf(conn, "UNKNOWN_COMMAND\n")
    }
}

// 配置文件监听（带去抖动）
func (d *daemon) watchConfigFile() {
    d.debouncer = &Debouncer{}
    
    go func() {
        for {
            select {
            case event := <-d.watcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    // 100ms 去抖动，避免频繁重载
                    d.debouncer.Debounce(100*time.Millisecond, d.reloadConfig)
                }
            case err := <-d.watcher.Errors:
                log.Printf("watcher error: %v", err)
            }
        }
    }()
}

// 重新加载配置（优化版）
func (d *daemon) reloadConfig() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // 读取最新配置
    cm := config.NewConfigManager()
    activeConfig, err := cm.GetActive()
    if err != nil {
        log.Printf("reload config error: %v", err)
        return  // 保持旧配置
    }
    
    // 更新缓存
    d.envCache = buildEnvMap(activeConfig)
    d.updateVersion()  // 更新版本号
}

// 更新版本号（基于时间戳）
func (d *daemon) updateVersion() {
    d.version = fmt.Sprintf("%d", time.Now().Unix())
}

// 清理函数
func (d *daemon) cleanup() {
    // 移除Socket文件
    os.Remove(d.sockPath)
    // 移除PID文件
    os.Remove(d.pidPath)
}

// 信号处理（优雅关闭）
func (d *daemon) handleSignals(ctx context.Context) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
    
    go func() {
        for {
            select {
            case sig := <-sigChan:
                switch sig {
                case syscall.SIGHUP:
                    // SIGHUP: 重载配置
                    d.reloadConfig()
                case syscall.SIGINT, syscall.SIGTERM:
                    // 优雅关闭
                    d.cleanup()
                    os.Exit(0)
                }
            case <-ctx.Done():
                d.cleanup()
                return
            }
        }
    }()
}
```

### 2. Shell 集成脚本（优化版）

**文件**: `~/.config/apimgr/shell-integration.sh`

**生成器**: `internal/shell/integration.go`

**脚本内容**:
```bash
#!/usr/bin/env bash
# apimgr shell integration v2.0
# Auto-generated by: apimgr enable
# DO NOT EDIT MANUALLY

# 配置
APIMGR_SOCK="${XDG_RUNTIME_DIR:-/tmp}/apimgr-${UID}/apimgr.sock"
APIMGR_CACHE_TTL=10  # 缓存有效期（秒）
APIMGR_CMD_COUNT=0   # 命令计数器
APIMGR_CMD_THRESHOLD=10  # 每N个命令检查一次

# 缓存变量
_APIMGR_LAST_CHECK=0
_APIMGR_VERSION=""
_APIMGR_ERROR_SHOWN=""

# 性能优化：智能缓存机制
_apimgr_should_check() {
  local now=$(date +%s 2>/dev/null || echo 0)
  local elapsed=$((now - _APIMGR_LAST_CHECK))
  
  # 强制刷新标志
  if [[ "$_APIMGR_FORCE_RELOAD" == "1" ]]; then
    _APIMGR_FORCE_RELOAD=""
    return 0
  fi
  
  # 时间检查（超过TTL）
  if [[ $elapsed -gt $APIMGR_CACHE_TTL ]]; then
    return 0
  fi
  
  # 命令计数检查
  APIMGR_CMD_COUNT=$((APIMGR_CMD_COUNT + 1))
  if [[ $APIMGR_CMD_COUNT -ge $APIMGR_CMD_THRESHOLD ]]; then
    APIMGR_CMD_COUNT=0
    return 0
  fi
  
  return 1
}

# 守护进程自动启动
_apimgr_ensure_daemon() {
  if [[ ! -S "$APIMGR_SOCK" ]]; then
    # 仅在交互式shell中尝试启动
    if [[ -t 1 ]] && command -v apimgr &>/dev/null; then
      apimgr daemon start &>/dev/null &
      disown 2>/dev/null
      sleep 0.2  # 等待守护进程就绪
    fi
  fi
}

# 纯Shell JSON解析（降级方案）
_apimgr_parse_json_simple() {
  local json="$1"
  local IFS=$'\n'
  
  # 简单但可靠的JSON解析
  for line in $json; do
    # 匹配 "key": "value" 模式
    if [[ "$line" =~ \"([^\"]+)\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
      local key="${BASH_REMATCH[1]}"
      local value="${BASH_REMATCH[2]}"
      export "$key=$value"
    fi
  done
}

# Socket通信（多方式支持）
_apimgr_socket_read() {
  local cmd="${1:-GET}"
  local response=""
  
  # 方式1: nc (netcat)
  if command -v nc &>/dev/null; then
    response=$(echo "$cmd" | nc -U "$APIMGR_SOCK" 2>/dev/null)
  # 方式2: socat
  elif command -v socat &>/dev/null; then
    response=$(echo "$cmd" | socat - UNIX-CONNECT:"$APIMGR_SOCK" 2>/dev/null)
  # 方式3: 纯bash (仅Linux)
  elif [[ -e /proc/net/unix ]] && [[ -n "$BASH_VERSION" ]]; then
    # 使用bash的内置/dev/tcp需要改造daemon支持TCP
    # 这里作为最后的降级方案，直接读取配置文件
    if [[ -f ~/.config/apimgr/config.json ]]; then
      response=$(cat ~/.config/apimgr/config.json 2>/dev/null | grep -A 10 "\"active\"")
    fi
  fi
  
  echo "$response"
}

# 主加载函数（优化版）
_apimgr_load_env() {
  # 性能优化：检查是否需要更新
  if ! _apimgr_should_check; then
    return 0
  fi
  
  # 更新检查时间
  _APIMGR_LAST_CHECK=$(date +%s 2>/dev/null || echo 0)
  
  # 确保守护进程运行
  _apimgr_ensure_daemon
  
  # 检查socket
  if [[ ! -S "$APIMGR_SOCK" ]]; then
    # 友好的错误提示（仅显示一次）
    if [[ -t 1 ]] && [[ -z "$_APIMGR_ERROR_SHOWN" ]]; then
      echo "apimgr: 守护进程未运行，请运行 'apimgr enable' 启用自动应用模式" >&2
      _APIMGR_ERROR_SHOWN=1
    fi
    return 1
  fi
  
  # 版本检查（减少完整加载）
  local remote_version
  remote_version=$(_apimgr_socket_read "VERSION")
  if [[ -n "$_APIMGR_VERSION" ]] && [[ "$remote_version" == "$_APIMGR_VERSION" ]]; then
    return 0  # 版本未变，跳过更新
  fi
  _APIMGR_VERSION="$remote_version"
  
  # 获取环境变量
  local env_json
  env_json=$(_apimgr_socket_read "GET")
  
  if [[ -z "$env_json" ]]; then
    return 1
  fi
  
  # 解析并设置环境变量
  if command -v jq &>/dev/null; then
    # 优先使用jq（更可靠）
    eval "$(echo "$env_json" | jq -r 'to_entries | map("export \(.key)=\"\(.value)\"") | .[]' 2>/dev/null)"
  else
    # 降级到纯Shell解析
    _apimgr_parse_json_simple "$env_json"
  fi
}

# Shell钩子注册（优化版）
if [[ -n "$ZSH_VERSION" ]]; then
  # Zsh: 使用precmd hook
  autoload -Uz add-zsh-hook 2>/dev/null || true
  if type add-zsh-hook &>/dev/null; then
    add-zsh-hook precmd _apimgr_load_env
  else
    # 降级方案
    precmd_functions+=(_apimgr_load_env)
  fi
elif [[ -n "$BASH_VERSION" ]]; then
  # Bash: 使用PROMPT_COMMAND
  if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="_apimgr_load_env"
  elif [[ "$PROMPT_COMMAND" != *"_apimgr_load_env"* ]]; then
    PROMPT_COMMAND="_apimgr_load_env;$PROMPT_COMMAND"
  fi
fi

# 初始化
_apimgr_ensure_daemon  # 确保守护进程运行
_APIMGR_FORCE_RELOAD=1  # 首次强制加载
_apimgr_load_env

# 命令包装器（智能刷新）
apimgr() {
  command apimgr "$@"
  local ret=$?
  
  # switch/add/remove命令后立即刷新
  case "$1" in
    switch|add|remove)
      if [[ $ret -eq 0 ]]; then
        _APIMGR_FORCE_RELOAD=1
        _apimgr_load_env
      fi
      ;;
  esac
  
  return $ret
}

# 诊断函数（用户可调用）
apimgr_debug() {
  echo "=== apimgr 诊断信息 ==="
  echo "Socket路径: $APIMGR_SOCK"
  echo "Socket存在: $([ -S "$APIMGR_SOCK" ] && echo "是" || echo "否")"
  echo "守护进程PID: $(cat ${XDG_RUNTIME_DIR:-/tmp}/apimgr-${UID}/daemon.pid 2>/dev/null || echo "未知")"
  echo "上次检查: $_APIMGR_LAST_CHECK"
  echo "配置版本: $_APIMGR_VERSION"
  echo "缓存TTL: ${APIMGR_CACHE_TTL}秒"
  echo "命令计数: $APIMGR_CMD_COUNT/$APIMGR_CMD_THRESHOLD"
  echo ""
  echo "依赖检查:"
  echo "  jq: $(command -v jq &>/dev/null && echo "✓ 已安装" || echo "✗ 未安装（使用降级解析）")"
  echo "  nc: $(command -v nc &>/dev/null && echo "✓ 已安装" || echo "✗ 未安装")"
  echo "  socat: $(command -v socat &>/dev/null && echo "✓ 已安装" || echo "✗ 未安装")"
  echo ""
  echo "当前环境变量:"
  env | grep -E "^(ANTHROPIC_|APIMGR_)" | sed 's/^/  /'
}
```

**生成器实现**:
```go
// internal/shell/integration.go
package shell

import (
    "os"
    "path/filepath"
    "text/template"
)

const shellScriptTemplate = `#!/usr/bin/env bash
# ... (上面的脚本内容)
`

func GenerateIntegrationScript(outputPath string) error {
    tmpl := template.Must(template.New("shell").Parse(shellScriptTemplate))
    
    f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
    if err != nil {
        return err
    }
    defer f.Close()
    
    return tmpl.Execute(f, nil)
}

func InstallToShellRC(shellType string) error {
    homeDir, _ := os.UserHomeDir()
    
    var rcFile string
    switch shellType {
    case "zsh":
        rcFile = filepath.Join(homeDir, ".zshrc")
    case "bash":
        rcFile = filepath.Join(homeDir, ".bashrc")
    default:
        return fmt.Errorf("unsupported shell: %s", shellType)
    }
    
    // 检查是否已安装
    content, _ := os.ReadFile(rcFile)
    if strings.Contains(string(content), "apimgr/shell-integration.sh") {
        return nil // Already installed
    }
    
    // 添加 source 行
    scriptPath := filepath.Join(homeDir, ".config", "apimgr", "shell-integration.sh")
    sourceLine := fmt.Sprintf("\n# apimgr - API configuration manager\n[[ -f %s ]] && source %s\n",
        scriptPath, scriptPath)
    
    f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    
    _, err = f.WriteString(sourceLine)
    return err
}
```

### 3. 新增 CLI 命令

#### `apimgr enable`

**职责**: 一键启用自动应用模式

**实现**: `cmd/enable.go`

```go
var EnableCmd = &cobra.Command{
    Use:   "enable",
    Short: "启用自动应用模式",
    Long: `一键启用无需重启的配置自动应用功能

此命令会执行以下操作:
1. 迁移配置文件到 ~/.config/apimgr/
2. 生成 Shell 集成脚本
3. 在 .zshrc/.bashrc 中添加一行引用
4. 启动守护进程

启用后，所有 'apimgr switch' 操作将立即在所有终端生效`,
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("🚀 正在启用 apimgr 自动应用模式...")
        
        // 1. 创建配置目录
        if err := ensureConfigDir(); err != nil {
            fmt.Fprintf(os.Stderr, "错误: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("✓ 配置目录已创建")
        
        // 2. 迁移旧配置文件
        if err := migrateOldConfig(); err != nil {
            fmt.Fprintf(os.Stderr, "警告: 配置迁移失败: %v\n", err)
        } else {
            fmt.Println("✓ 配置文件已迁移")
        }
        
        // 3. 生成 Shell 集成脚本
        if err := generateShellScript(); err != nil {
            fmt.Fprintf(os.Stderr, "错误: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("✓ Shell 集成脚本已生成")
        
        // 4. 安装到 Shell RC
        shellType := detectShell()
        if err := installToShellRC(shellType); err != nil {
            fmt.Fprintf(os.Stderr, "错误: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("✓ 已添加集成到 ~/.%src\n", shellType)
        
        // 5. 启动守护进程
        if err := startDaemon(); err != nil {
            fmt.Fprintf(os.Stderr, "错误: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("✓ 守护进程已启动")
        
        fmt.Println("\n✨ 自动应用模式已启用!")
        fmt.Printf("\n请运行以下命令使其生效:\n")
        fmt.Printf("  source ~/.%src\n\n", shellType)
        fmt.Println("或重新打开终端")
        fmt.Println("\n现在你可以直接使用:")
        fmt.Println("  apimgr switch <配置>  # 立即在所有终端生效")
    },
}

func ensureConfigDir() error {
    homeDir, _ := os.UserHomeDir()
    configDir := filepath.Join(homeDir, ".config", "apimgr")
    return os.MkdirAll(configDir, 0755)
}

func migrateOldConfig() error {
    homeDir, _ := os.UserHomeDir()
    oldPath := filepath.Join(homeDir, ".apimgr.json")
    newPath := filepath.Join(homeDir, ".config", "apimgr", "config.json")
    
    if _, err := os.Stat(oldPath); os.IsNotExist(err) {
        return nil // No old config
    }
    
    // 迁移文件
    return os.Rename(oldPath, newPath)
}

func startDaemon() error {
    // 检查是否已运行
    if daemon.IsRunning() {
        return nil
    }
    
    // 以后台进程启动
    cmd := exec.Command(os.Args[0], "daemon", "start")
    cmd.Stdout = nil
    cmd.Stderr = nil
    cmd.Stdin = nil
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // 等待守护进程就绪
    time.Sleep(500 * time.Millisecond)
    
    if !daemon.IsRunning() {
        return fmt.Errorf("守护进程启动失败")
    }
    
    return nil
}
```

#### `apimgr disable`

**职责**: 禁用自动应用模式

**实现**: `cmd/disable.go`

```go
var DisableCmd = &cobra.Command{
    Use:   "disable",
    Short: "禁用自动应用模式",
    Long:  "停止守护进程并从 Shell 配置中移除集成",
    Run: func(cmd *cobra.Command, args []string) {
        purge, _ := cmd.Flags().GetBool("purge")
        
        fmt.Println("🛑 正在禁用 apimgr 自动应用模式...")
        
        // 1. 停止守护进程
        if err := stopDaemon(); err != nil {
            fmt.Fprintf(os.Stderr, "警告: %v\n", err)
        } else {
            fmt.Println("✓ 守护进程已停止")
        }
        
        // 2. 从 Shell RC 移除集成
        shellType := detectShell()
        if err := removeFromShellRC(shellType); err != nil {
            fmt.Fprintf(os.Stderr, "警告: %v\n", err)
        } else {
            fmt.Printf("✓ 已从 ~/.%src 移除集成\n", shellType)
        }
        
        // 3. 清理文件（可选）
        if purge {
            if err := purgeConfigFiles(); err != nil {
                fmt.Fprintf(os.Stderr, "警告: %v\n", err)
            } else {
                fmt.Println("✓ 配置文件已删除")
            }
        }
        
        fmt.Println("\n✓ 自动应用模式已禁用")
        if !purge {
            fmt.Println("\n提示: 配置文件已保留，使用 --purge 删除所有数据")
        }
    },
}

func init() {
    DisableCmd.Flags().BoolP("purge", "p", false, "同时删除所有配置文件")
}
```

#### `apimgr daemon` (隐藏命令)

**职责**: 守护进程管理（内部命令）

**实现**: `cmd/daemon.go`

```go
var DaemonCmd = &cobra.Command{
    Use:    "daemon",
    Short:  "守护进程管理（内部命令）",
    Hidden: true,
}

var DaemonStartCmd = &cobra.Command{
    Use:   "start",
    Short: "启动守护进程",
    Run: func(cmd *cobra.Command, args []string) {
        d := daemon.New()
        
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        
        if err := d.Start(ctx); err != nil {
            fmt.Fprintf(os.Stderr, "守护进程启动失败: %v\n", err)
            os.Exit(1)
        }
    },
}

var DaemonStopCmd = &cobra.Command{
    Use:   "stop",
    Short: "停止守护进程",
    Run: func(cmd *cobra.Command, args []string) {
        if err := daemon.Stop(); err != nil {
            fmt.Fprintf(os.Stderr, "停止守护进程失败: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("✓ 守护进程已停止")
    },
}

var DaemonStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "查看守护进程状态",
    Run: func(cmd *cobra.Command, args []string) {
        if daemon.IsRunning() {
            pid := daemon.GetPID()
            fmt.Printf("✓ 守护进程运行中 (PID: %d)\n", pid)
        } else {
            fmt.Println("✗ 守护进程未运行")
        }
    },
}

func init() {
    DaemonCmd.AddCommand(DaemonStartCmd)
    DaemonCmd.AddCommand(DaemonStopCmd)
    DaemonCmd.AddCommand(DaemonStatusCmd)
}
```

### 4. 配置管理器更新

**文件**: `config/config.go`

**更新要点**:
```go
// 更新配置路径到 XDG 规范
func NewConfigManager() *ConfigManager {
    homeDir, _ := os.UserHomeDir()
    
    // 优先使用 XDG 路径
    configDir := os.Getenv("XDG_CONFIG_HOME")
    if configDir == "" {
        configDir = filepath.Join(homeDir, ".config")
    }
    
    configPath := filepath.Join(configDir, "apimgr", "config.json")
    
    // 兼容旧路径
    oldPath := filepath.Join(homeDir, ".apimgr.json")
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        if _, err := os.Stat(oldPath); err == nil {
            // 提示用户迁移
            fmt.Fprintf(os.Stderr, "提示: 检测到旧配置文件，请运行 'apimgr enable' 迁移\n")
            configPath = oldPath
        }
    }
    
    return &ConfigManager{
        configPath: configPath,
    }
}
```

---

## 完整工作流程

### 首次启用

```bash
# 1. 用户执行启用命令
$ apimgr enable

🚀 正在启用 apimgr 自动应用模式...
✓ 配置目录已创建
✓ 配置文件已迁移到 ~/.config/apimgr/config.json
✓ Shell 集成脚本已生成
✓ 已添加集成到 ~/.zshrc
✓ 守护进程已启动 (PID: 12345)

✨ 自动应用模式已启用!

请运行以下命令使其生效:
  source ~/.zshrc

或重新打开终端

现在你可以直接使用:
  apimgr switch <配置>  # 立即在所有终端生效

# 2. 使配置生效
$ source ~/.zshrc

# 3. 验证守护进程
$ apimgr daemon status
✓ 守护进程运行中 (PID: 12345)
```

### 日常使用

```bash
# 终端 A
$ apimgr switch dev
✓ 已切换到配置: dev

$ echo $ANTHROPIC_API_KEY
sk-dev-xxx  # ✅ 立即生效

# 终端 B（同时打开）
$ echo $ANTHROPIC_API_KEY
sk-dev-xxx  # ✅ 自动同步（下次命令执行时更新）

# 终端 A 切换到另一个配置
$ apimgr switch prod
✓ 已切换到配置: prod

# 终端 B 执行任何命令后自动更新
$ pwd
/home/user
$ echo $ANTHROPIC_API_KEY
sk-prod-xxx  # ✅ 已自动更新为 prod 配置
```

### 新终端

```bash
# 新打开的终端
$ echo $ANTHROPIC_API_KEY
sk-prod-xxx  # ✅ 自动加载当前活动配置

$ apimgr status
当前激活的配置:
  别名: prod
  API Key: sk-p****xxx
  Base URL: https://api.anthropic.com
  Model: claude-3-sonnet
```

### 禁用功能

```bash
# 仅禁用（保留配置）
$ apimgr disable
🛑 正在禁用 apimgr 自动应用模式...
✓ 守护进程已停止
✓ 已从 ~/.zshrc 移除集成

✓ 自动应用模式已禁用

提示: 配置文件已保留，使用 --purge 删除所有数据

# 完全卸载
$ apimgr disable --purge
🛑 正在禁用 apimgr 自动应用模式...
✓ 守护进程已停止
✓ 已从 ~/.zshrc 移除集成
✓ 配置文件已删除

✓ 自动应用模式已禁用
```

---

## 技术细节

### Unix Socket 通信协议

**请求格式**:
```
命令支持：
- GET\n         # 获取环境变量
- VERSION\n     # 获取配置版本号
- PING\n        # 健康检查
- RELOAD\n      # 强制重载配置
```

**响应格式**:

GET 响应 (JSON):
```json
{
  "ANTHROPIC_API_KEY": "sk-dev-xxx",
  "ANTHROPIC_BASE_URL": "https://api.anthropic.com",
  "ANTHROPIC_MODEL": "claude-3-opus",
  "APIMGR_ACTIVE": "dev"
}
```

VERSION 响应:
```
1699123456
```

PING 响应:
```
PONG
```

### 性能优化（增强版）

1. **智能缓存机制**:
   - 版本号检查：仅在配置变化时更新环境变量
   - 时间戳缓存：默认 10 秒内不重复查询
   - 命令计数器：每 10 个命令检查一次
   - 强制刷新：switch/add/remove 后立即更新

2. **守护进程自动启动**:
   - Shell 启动时自动检测并启动守护进程
   - 无需用户手动干预
   - 交互式 Shell 限定，避免脚本环境启动

3. **依赖降级处理**:
   - jq 不存在时使用纯 Shell JSON 解析
   - nc/socat 都不存在时降级到文件读取
   - 所有核心功能无外部依赖

4. **配置监听去抖动**:
   - 100ms 去抖动窗口
   - 避免编辑器保存时的多次触发
   - 减少不必要的重载操作

### 错误处理（增强版）

1. **守护进程崩溃恢复**:
   - Shell 集成自动检测并重启
   - 保持用户无感知
   - 错误信息仅显示一次

2. **Socket 文件清理**:
   - 启动前检测并清理残留 Socket
   - 优雅关闭时自动清理
   - 信号处理确保清理执行

3. **配置文件损坏处理**:
   - 保持上次有效配置
   - 错误日志记录
   - 不影响已运行的环境

4. **友好错误提示**:
   - 仅在交互式 Shell 显示
   - 避免脚本执行时的噪音
   - 提供诊断函数 `apimgr_debug`

### 安全性

1. **Socket 权限**: `0600`（仅所有者可访问）
2. **PID 文件保护**: 防止多实例启动
3. **配置文件加密**: 未来可选支持系统 Keychain
4. **审计日志**: 记录所有配置切换操作（可选）

---

## 依赖管理

### 新增 Go 依赖

```bash
go get github.com/fsnotify/fsnotify  # 文件监听
```

### Shell 依赖

**必需**:
- `nc` (netcat) 或 `socat` - Unix Socket 通信
- `jq` - JSON 解析

**可选**:
- `curl` - API 测试（未来功能）

**检测与提示**:
```bash
# Shell 集成脚本中检测
if ! command -v nc &>/dev/null && ! command -v socat &>/dev/null; then
  echo "警告: 需要 nc 或 socat 才能使用自动应用功能" >&2
  echo "安装方法: brew install netcat  # macOS" >&2
  echo "         apt install netcat-openbsd  # Ubuntu" >&2
fi

if ! command -v jq &>/dev/null; then
  echo "警告: 需要 jq 才能解析配置" >&2
  echo "安装方法: brew install jq" >&2
fi
```

---

## 兼容性矩阵

| 平台 | Shell | 守护进程 | Socket | 状态 |
|------|-------|---------|--------|------|
| macOS 12+ | zsh | ✅ | ✅ | 完全支持 |
| macOS 12+ | bash | ✅ | ✅ | 完全支持 |
| Linux | zsh | ✅ | ✅ | 完全支持 |
| Linux | bash | ✅ | ✅ | 完全支持 |
| WSL2 | zsh/bash | ✅ | ✅ | 完全支持 |
| Windows | PowerShell | ⚠️ | ⚠️ | 未来支持 |

---

## 实现计划

### Phase 1: 核心功能（3-4 天）
- [x] 设计文档编写
- [ ] 守护进程基础框架
  - [ ] Unix Socket 服务器
  - [ ] 配置文件监听
  - [ ] 环境变量缓存
  - [ ] PID 管理
- [ ] `apimgr enable` 命令
- [ ] `apimgr daemon` 子命令
- [ ] 配置迁移逻辑

### Phase 2: Shell 集成（2 天）
- [ ] Shell 集成脚本生成器
- [ ] 自动添加到 `.zshrc`/`.bashrc`
- [ ] `precmd`/`PROMPT_COMMAND` 钩子
- [ ] 降级处理（依赖缺失）

### Phase 3: 测试与完善（2 天）
- [ ] 单元测试
  - [ ] 守护进程启动/停止
  - [ ] Socket 通信
  - [ ] 配置监听
- [ ] 集成测试
  - [ ] 完整工作流测试
  - [ ] 多终端同步测试
- [ ] 错误场景测试
  - [ ] 守护进程崩溃恢复
  - [ ] 配置文件损坏处理

### Phase 4: 文档与发布（1 天）
- [ ] 更新 README.md
- [ ] 编写迁移指南
- [ ] 发布 Release Notes

---

## 未来扩展

### 多服务商支持

```bash
# 同时管理多个服务商
apimgr add dev --provider anthropic --sk sk-xxx
apimgr add dev --provider openai --sk sk-xxx

# 切换时同时应用
apimgr switch dev
# 自动设置:
#   ANTHROPIC_API_KEY=sk-xxx
#   OPENAI_API_KEY=sk-yyy
```

### 配置同步

```bash
# 导出配置（脱敏）
apimgr export --output config-template.json

# 导入配置
apimgr import config-template.json
```

### API 健康检查

```bash
# 切换时自动测试连接
apimgr switch dev --verify

# 定期健康检查
apimgr daemon start --health-check-interval 5m
```

### 图形界面

```bash
# TUI 界面
apimgr tui

# Web 界面（守护进程提供）
apimgr web --port 8080
```

---

## 参考资料

- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
- [Unix Domain Socket in Go](https://pkg.go.dev/net#UnixListener)
- [fsnotify Documentation](https://github.com/fsnotify/fsnotify)
- [Zsh Hook Functions](https://zsh.sourceforge.io/Doc/Release/Functions.html#Hook-Functions)
- [Bash PROMPT_COMMAND](https://www.gnu.org/software/bash/manual/html_node/Controlling-the-Prompt.html)

---

## 维护者

- 设计: Droid (Factory AI)
- 实现: TBD
- 审核: TBD

**最后更新**: 2025-11-09
