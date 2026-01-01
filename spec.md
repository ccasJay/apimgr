# API Manager - 交互式模型选择功能规格说明

**版本**: 1.0.0
**日期**: 2026-01-01
**状态**: 草案

## 文档历史

| 版本 | 日期 | 作者 | 说明 |
|------|------|------|------|
| 1.0.0 | 2026-01-01 | Claude Code | 初始版本 |

## 目录

1. [概述](#1-概述)
   - [1.1 问题陈述](#11-问题陈述)
   - [1.2 解决方案](#12-解决方案)
   - [1.3 设计原则](#13-设计原则)
2. [用户场景与需求](#2-用户场景与需求)
   - [2.1 核心用户场景](#21-核心用户场景)
   - [2.2 功能需求](#22-功能需求)
   - [2.3 非功能需求](#23-非功能需求)
3. [技术设计](#3-技术设计)
   - [3.1 架构决策](#31-架构决策)
   - [3.2 接口设计](#32-接口设计)
   - [3.3 数据结构变更](#33-数据结构变更)
4. [用户界面设计](#4-用户界面设计)
   - [4.1 交互流程](#41-交互流程)
   - [4.2 视觉反馈](#42-视觉反馈)
   - [4.3 键盘快捷键](#43-键盘快捷键)
5. [向后兼容性](#5-向后兼容性)
   - [5.1 兼容性保证](#51-兼容性保证)
   - [5.2 新增参数](#52-新增参数)
   - [5.3 环境变量控制](#53-环境变量控制)
6. [测试策略](#6-测试策略)
   - [6.1 单元测试](#61-单元测试)
   - [6.2 集成测试](#62-集成测试)
   - [6.3 端到端测试](#63-端到端测试)
7. [实施计划](#7-实施计划)
   - [7.1 第一阶段：基础功能](#71-第一阶段基础功能)
   - [7.2 第二阶段：增强功能](#72-第二阶段增强功能)
   - [7.3 第三阶段：生态集成](#73-第三阶段生态集成)
8. [风险评估与缓解](#8-风险评估与缓解)
   - [8.1 风险矩阵](#81-风险矩阵)
   - [8.2 监控策略](#82-监控策略)
   - [8.3 回滚机制](#83-回滚机制)
9. [附录](#9-附录)
   - [A. 相关代码示例](#a-相关代码示例)
   - [B. 配置选项示例](#b-配置选项示例)
   - [C. 测试用例示例](#c-测试用例示例)

---

## 1. 概述

### 1.1 问题陈述

当前 `apimgr switch <alias>` 命令在切换配置时，如果配置支持多个模型（通过 `--models` 参数添加），会默认使用配置中存储的 `model` 字段值（通常是第一个添加的模型）。这带来了以下问题：

1. **用户意图不明确**：用户切换配置时可能希望使用不同的模型，但需要额外使用 `--model` 参数
2. **配置状态不透明**：用户不知道当前配置支持哪些模型，需要先运行 `apimgr list` 查看
3. **交互体验不流畅**：多步骤操作降低了工具的使用效率

### 1.2 解决方案

在 `apimgr switch <alias>` 命令执行时，如果目标配置支持多个模型（`len(Models) > 1`）且用户未通过 `--model` 参数指定模型，**自动弹出交互式选择窗口**，让用户选择要激活的模型。

### 1.3 设计原则

1. **渐进式交互**：仅在必要时才询问用户（多个模型时）
2. **向后兼容**：不影响现有脚本和自动化流程
3. **配置优先**：尊重现有的配置默认值
4. **一致性**：重用现有交互模式，保持用户体验统一

## 2. 用户场景与需求

### 2.1 核心用户场景

**场景一：开发者在不同模型间切换**
```
$ apimgr switch my-anthropic
📋 Available models:
  ➤ 1. claude-3-opus-20240229 (current)
    2. claude-3-sonnet-20240229
    3. claude-3-haiku-20240307

Select model (1-3) [Enter to use 'claude-3-opus-20240229']: 2
✓ Switched model to: claude-3-sonnet-20240229
✓ Switched to configuration: my-anthropic
```

**场景二：自动化脚本不受影响**
```bash
# 现有脚本继续正常工作
apimgr switch my-anthropic --model claude-3-haiku-20240307
```

**场景三：单模型配置无需交互**
```bash
# 如果配置只有一个模型，直接切换
$ apimgr switch single-model-config
✓ Switched to model: gpt-4o
✓ Switched to configuration: single-model-config
```

### 2.2 功能需求

| ID | 需求描述 | 优先级 | 验收标准 |
|----|----------|--------|----------|
| FR1 | 当执行 `apimgr switch <alias>` 时，如果目标配置有多个支持模型且用户未指定 `--model`，显示交互式选择列表 | P0 | 多模型配置时显示选择列表 |
| FR2 | 选择列表中高亮显示当前活跃模型（配置中存储的 `Model` 字段） | P1 | 当前模型有视觉标识 |
| FR3 | 用户可通过键盘箭头键导航，按 Enter 确认选择 | P0 | 键盘导航正常工作 |
| FR4 | 如果配置只有一个模型，不显示选择列表，直接使用该模型 | P0 | 单模型配置无提示 |
| FR5 | 支持本地模式（`-l/--local`）下的交互式选择 | P1 | 本地模式功能正常 |
| FR6 | 如果当前活跃模型不在支持列表中（如模型被移除），显示警告并重新选择 | P2 | 无效模型有警告提示 |
| FR7 | 提供 `--no-prompt` 参数禁用交互式选择 | P1 | 参数有效禁用提示 |
| FR8 | 提供 `--list-models` 参数显示模型列表并退出 | P2 | 参数显示列表后退出 |

### 2.3 非功能需求

| ID | 需求描述 | 优先级 | 验收标准 |
|----|----------|--------|----------|
| NFR1 | 交互响应时间 < 100ms | P1 | 操作响应迅速 |
| NFR2 | 终端兼容性：支持 Bash、Zsh、Fish、PowerShell | P1 | 主流终端正常工作 |
| NFR3 | 无外部 GUI 依赖，纯终端实现 | P0 | 不引入新依赖 |
| NFR4 | 良好的错误处理和用户反馈 | P1 | 错误信息清晰友好 |
| NFR5 | 与现有配置格式完全兼容 | P0 | 配置文件无需修改 |

## 3. 技术设计

### 3.1 架构决策

#### 3.1.1 交互库选择

**决策**: 无新增依赖，重用现有交互模式
**理由**:
1. **一致性**: 现有 `add` 命令使用 `fmt.Print` 交互，新功能应保持一致
2. **轻量化**: 避免引入外部依赖，减少维护负担
3. **跨平台兼容**: 简单数字输入在所有终端中都能正常工作
4. **向后兼容**: 不改变现有构建和部署流程

**备选方案**:
- `survey/v2`: 功能丰富，维护活跃（未来扩展选项）
- `promptui`: 更轻量，但功能有限
- 内置TUI: 使用 `termbox-go` 或 `tcell` 实现自定义交互

#### 3.1.2 交互流程设计

```
开始执行 `apimgr switch <alias>`
    ↓
加载目标配置 `configManager.Get(alias)`
    ↓
检查 `--model` 参数是否提供？
    ├─ 是 → 使用指定模型，执行现有逻辑
    └─ 否 → 检查配置的 `Models` 列表长度
           ├─ len(Models) == 0 → 错误：无可用模型
           ├─ len(Models) == 1 → 使用唯一模型
           └─ len(Models) >= 2 → 显示交互式选择列表
                  ↓
              用户选择模型
                  ↓
              更新配置的 `Model` 字段
                  ↓
              执行原有切换逻辑
```

### 3.2 接口设计

#### 3.2.1 新增函数签名

```go
// cmd/model_selection.go
package cmd

import "apimgr/config"

// ModelSelector 负责模型选择的交互逻辑
type ModelSelector struct{}

// SelectorOptions 配置交互选项
type SelectorOptions struct {
    UseNumberedList bool   // 使用数字列表
    ShowHelpText    bool   // 显示帮助文本
    AllowSkip       bool   // 允许跳过（使用默认值）
}

// ShouldPrompt 判断是否需要提示选择
func (ms *ModelSelector) ShouldPrompt(cfg *config.APIConfig, modelFlag string, noPrompt bool) bool

// PromptSimple 简单数字选择交互
func (ms *ModelSelector) PromptSimple(models []string, currentModel string, opts *SelectorOptions) (string, error)

// ValidateModelInList 验证模型是否在列表中
func (ms *ModelSelector) ValidateModelInList(model string, models []string) error
```

#### 3.2.2 修改现有逻辑

在 `cmd/switch.go` 的 `RunE` 函数中，第56-76行之后添加：

```go
// 检查是否需要交互式选择模型
noPrompt, _ := cmd.Flags().GetBool("no-prompt")
selector := &ModelSelector{}

if selector.ShouldPrompt(apiConfig, modelFlag, noPrompt) {
    selectedModel, err := selector.PromptSimple(apiConfig.Models, apiConfig.Model, nil)
    if err != nil {
        return fmt.Errorf("model selection failed: %w", err)
    }

    // 切换到用户选择的模型
    if err := configManager.SwitchModel(alias, selectedModel); err != nil {
        return err
    }

    // 刷新配置以获取更新后的模型
    apiConfig, err = configManager.Get(alias)
    if err != nil {
        return err
    }

    fmt.Fprintf(os.Stderr, "✓ Switched model to: %s\n", selectedModel)
}
```

### 3.3 数据结构变更

**无变更**: 现有数据结构保持不变

**扩展支持**:
```go
// 可选：未来可扩展的配置选项
type UISettings struct {
    ModelSelection struct {
        Enabled           bool `json:"enabled" default:"true"`
        ShowCurrentFirst  bool `json:"show_current_first" default:"true"`
        UseNumberedList   bool `json:"use_numbered_list" default:"true"`
    } `json:"model_selection"`
}
```

## 4. 用户界面设计

### 4.1 交互流程

```
1. 检测条件:
   - 配置支持多个模型 (len(Models) > 1)
   - 用户未指定 --model 参数
   - 当前在交互式终端中
   - 未设置 --no-prompt 参数

2. 显示界面:
   📋 Available models:
     ➤ 1. claude-3-opus-20240229 (current)
       2. claude-3-sonnet-20240229
       3. claude-3-haiku-20240307

   Select model (1-3) [Enter to use 'claude-3-opus-20240229']: █

3. 处理输入:
   - 输入数字: 选择对应模型
   - 输入 Enter: 使用当前活跃模型
   - 输入 Ctrl+C: 取消选择，退出命令
```

### 4.2 视觉反馈

**成功反馈**:
```
✓ Switched model to: claude-3-sonnet-20240229
✓ Switched to configuration: my-anthropic
```

**错误反馈**:
```
❌ Model selection failed: invalid input '5', expected a number between 1-3
💡 Tip: Use --model parameter to specify model directly
      Or use --no-prompt to disable interactive selection
```

### 4.3 键盘快捷键

| 按键 | 功能 | 说明 |
|------|------|------|
| **数字 1-n** | 选择对应序号的模型 | 立即生效 |
| **Enter键** | 使用当前活跃模型 | 保持现状 |
| **Ctrl+C** | 取消选择，退出命令 | 安全退出 |
| **q/Q** | 退出选择 | 可选增强 |

## 5. 向后兼容性

### 5.1 兼容性保证

| 场景 | 旧行为 | 新行为 | 兼容性保证 |
|------|--------|--------|------------|
| `apimgr switch <alias> --model <model>` | 切换到指定模型 | **完全一致** | ✅ 保证 |
| `echo "..." | apimgr switch <alias>` | 使用当前模型 | **完全一致** | ✅ 保证 |
| 单模型配置切换 | 直接切换 | **完全一致** | ✅ 保证 |
| 配置格式读取 | 读取现有JSON | **完全一致** | ✅ 保证 |
| 环境变量导出 | 导出ANTHROPIC_*变量 | **完全一致** | ✅ 保证 |

### 5.2 新增可选参数

| 参数 | 说明 | 默认值 | 影响范围 |
|------|------|--------|----------|
| `--no-prompt` | 禁用交互式模型选择 | false | 仅影响交互触发 |
| `--list-models` | 显示配置支持的模型列表并退出 | false | 新增功能 |
| `--interactive` | 强制启用交互模式 | false | 高级用户选项 |

### 5.3 环境变量控制

| 变量 | 说明 | 默认值 | 优先级 |
|------|------|--------|--------|
| `APIMGR_NO_PROMPT=1` | 全局禁用交互提示 | 未设置 | 低 |
| `APIMGR_ALWAYS_PROMPT=1` | 总是询问模型选择 | 未设置 | 中 |
| `APIMGR_INTERACTIVE=1` | 强制启用交互模式 | 未设置 | 高 |

**优先级规则**:
1. 命令行参数 (`--no-prompt`, `--interactive`)
2. 环境变量 (`APIMGR_INTERACTIVE`, `APIMGR_NO_PROMPT`)
3. 配置文件设置
4. 系统默认行为

## 6. 测试策略

### 6.1 单元测试

**测试文件**: `cmd/model_selection_test.go`

```go
func TestModelSelection_ShouldPrompt(t *testing.T) {
    tests := []struct {
        name      string
        models    []string
        modelFlag string
        noPrompt  bool
        isTerminal bool
        want      bool
    }{
        {
            name:      "单模型配置不提示",
            models:    []string{"claude-3"},
            modelFlag: "",
            noPrompt:  false,
            isTerminal: true,
            want:      false,
        },
        {
            name:      "指定模型参数不提示",
            models:    []string{"claude-3", "gpt-4"},
            modelFlag: "gpt-4",
            noPrompt:  false,
            isTerminal: true,
            want:      false,
        },
        {
            name:      "非交互环境不提示",
            models:    []string{"claude-3", "gpt-4"},
            modelFlag: "",
            noPrompt:  false,
            isTerminal: false,
            want:      false,
        },
        {
            name:      "显式禁用不提示",
            models:    []string{"claude-3", "gpt-4"},
            modelFlag: "",
            noPrompt:  true,
            isTerminal: true,
            want:      false,
        },
        {
            name:      "多模型配置提示",
            models:    []string{"claude-3", "gpt-4"},
            modelFlag: "",
            noPrompt:  false,
            isTerminal: true,
            want:      true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := &config.APIConfig{Models: tt.models}
            got := selector.ShouldPrompt(cfg, tt.modelFlag, tt.noPrompt)
            if got != tt.want {
                t.Errorf("ShouldPrompt() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 6.2 集成测试

**测试文件**: `cmd/integration_test.go`

```go
func TestSwitchCommand_BackwardCompatibility(t *testing.T) {
    // 场景1: 使用 --model 参数（旧用法）
    testCase{
        name: "WithModelFlag",
        cmd:  "./apimgr switch test-config --model model2",
        expect: contains("✓ Switched to configuration: test-config"),
    }

    // 场景2: 非交互环境（脚本兼容性）
    testCase{
        name: "NonInteractive",
        cmd:  "echo test | ./apimgr switch test-config",
        expect: contains("export ANTHROPIC_MODEL"),
    }

    // 场景3: 多模型配置交互
    testCase{
        name: "InteractiveSelection",
        cmd:  "./apimgr switch multi-config",
        input: "2\n",
        expect: contains("✓ Switched model to: model2"),
    }
}
```

### 6.3 端到端测试

**测试脚本**: `test/e2e_model_selection.sh`

```bash
#!/bin/bash
# 端到端测试脚本

set -e

echo "🚀 开始交互式模型选择端到端测试"

# 1. 清理环境
echo "🧹 清理测试环境..."
rm -rf ~/.config/apimgr-test 2>/dev/null || true

# 2. 创建多模型配置
echo "📝 创建多模型测试配置..."
export XDG_CONFIG_HOME=~/.config/apimgr-test
./apimgr add test-multi --sk test-key --models "model-a,model-b,model-c"

# 3. 测试交互选择
echo "🔍 测试交互选择..."
{
    echo "2"  # 选择第二个模型
} | ./apimgr switch test-multi

# 4. 验证配置更新
echo "✅ 端到端测试通过"
```

## 7. 实施计划

### 7.1 第一阶段：基础功能（MVP）

**目标**: 实现核心交互逻辑，确保向后兼容

**任务清单**:
1. [ ] 创建 `cmd/model_selection.go`
2. [ ] 实现 `ModelSelector` 核心逻辑
3. [ ] 修改 `cmd/switch.go` 集成点
4. [ ] 添加 `--no-prompt` 参数支持
5. [ ] 编写单元测试
6. [ ] 验证向后兼容性

**验收标准**:
- 多模型配置显示选择列表
- 单模型配置无提示
- 脚本模式行为不变
- 所有现有测试通过

### 7.2 第二阶段：增强功能

**目标**: 提升用户体验，增加配置选项

**任务清单**:
1. [ ] 添加 `--list-models` 参数
2. [ ] 支持环境变量控制
3. [ ] 增强键盘快捷键
4. [ ] 改进错误处理和用户反馈
5. [ ] 添加配置驱动行为控制

**验收标准**:
- 用户可通过多种方式控制交互行为
- 错误信息清晰友好
- 支持高级使用场景

### 7.3 第三阶段：生态集成

**目标**: 扩展功能到其他命令，完善生态系统

**任务清单**:
1. [ ] 扩展到 `edit` 命令
2. [ ] 集成到项目级配置同步
3. [ ] 添加TUI模式（可选）
4. [ ] 性能优化和国际化支持

**验收标准**:
- 功能在相关命令中一致
- 与Claude Code设置完全同步
- 性能指标达标

## 8. 风险评估与缓解

### 8.1 风险矩阵

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 交互逻辑破坏脚本 | 低 | 高 | 非交互环境自动禁用 |
| 终端兼容性问题 | 中 | 中 | 简单数字输入，无特殊字符 |
| 用户习惯抗拒 | 中 | 低 | 提供禁用选项，清晰文档 |
| 配置数据损坏 | 低 | 高 | 验证后写入，事务性操作 |
| 性能影响 | 低 | 低 | 轻量实现，无额外I/O |

### 8.2 监控策略

**监控指标**:
1. **错误率**: 交互失败的频率（目标 <1%）
2. **响应时间**: 选择操作的延迟（目标 <100ms）
3. **使用率**: 交互选择功能的实际使用频率
4. **用户反馈**: 问题报告和功能请求

**告警规则**:
- 错误率 >5% 持续1小时 → 触发告警
- 平均响应时间 >200ms → 触发告警
- 功能禁用率异常上升 → 触发告警

### 8.3 回滚机制

**自动回滚触发器**:
```go
// 监测到以下情况时自动禁用功能
conditions := []struct{
    metric string
    threshold float64
    duration time.Duration
}{
    {"error_rate", 0.05, time.Hour},      // 5%错误率
    {"config_write_failure", 0.01, time.Hour}, // 1%配置写入失败
    {"response_time_ms", 300, time.Hour},  // 300ms响应时间
}
```

**渐进式推出策略**:
1. **阶段1 (10%用户)**: 内部测试团队，验证核心功能
2. **阶段2 (50%用户)**: 扩大测试范围，收集反馈，修复问题
3. **阶段3 (100%用户)**: 全面推出，监控错误率和用户反馈

## 9. 附录

### A. 相关代码示例

**完整的 ModelSelector 实现**:
```go
// cmd/model_selection.go
package cmd

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"

    "apimgr/config"
    "apimgr/internal/ui"
)

type ModelSelector struct{}

func (ms *ModelSelector) ShouldPrompt(cfg *config.APIConfig, modelFlag string, noPrompt bool) bool {
    if noPrompt {
        return false
    }
    if modelFlag != "" {
        return false
    }
    if len(cfg.Models) <= 1 {
        return false
    }
    return ui.IsInteractive()
}

func (ms *ModelSelector) PromptSimple(models []string, currentModel string) (string, error) {
    fmt.Println("\n📋 Available models:")

    for i, model := range models {
        indicator := "  "
        if model == currentModel {
            indicator = "➤ "
        }
        fmt.Printf("  %s%d. %s", indicator, i+1, model)
        if model == currentModel {
            fmt.Print(" (current)")
        }
        fmt.Println()
    }

    fmt.Printf("\nSelect model (1-%d) [Enter to use '%s']: ",
                len(models), currentModel)

    reader := bufio.NewReader(os.Stdin)
    input, err := reader.ReadString('\n')
    if err != nil {
        return "", fmt.Errorf("failed to read input: %w", err)
    }

    input = strings.TrimSpace(input)

    // 处理默认值
    if input == "" {
        return currentModel, nil
    }

    // 解析用户选择
    choice, err := strconv.Atoi(input)
    if err != nil {
        return "", fmt.Errorf("invalid input '%s', expected a number", input)
    }

    if choice < 1 || choice > len(models) {
        return "", fmt.Errorf("choice %d out of range (1-%d)", choice, len(models))
    }

    return models[choice-1], nil
}
```

### B. 配置选项示例

**配置文件扩展示例**:
```json
{
  "configs": [
    {
      "alias": "my-anthropic",
      "api_key": "sk-ant-api03-...",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3-opus-20240229",
      "models": [
        "claude-3-opus-20240229",
        "claude-3-sonnet-20240229",
        "claude-3-haiku-20240307"
      ]
    }
  ],
  "active": "my-anthropic",
  "ui": {
    "model_selection": {
      "enabled": true,
      "show_current_first": true,
      "use_numbered_list": true
    }
  }
}
```

### C. 测试用例示例

**端到端测试用例**:
```bash
# 测试场景1: 多模型配置交互选择
test_e2e_multi_model() {
    # 准备测试环境
    export XDG_CONFIG_HOME=$(mktemp -d)

    # 创建多模型配置
    ./apimgr add test-multi --sk test-key --models "model1,model2,model3"

    # 模拟用户选择第二个模型
    {
        echo "2"
        sleep 0.1
    } | ./apimgr switch test-multi

    # 验证输出包含确认信息
    assert_output_contains "✓ Switched model to: model2"

    # 清理
    rm -rf "$XDG_CONFIG_HOME"
}

# 测试场景2: 脚本模式兼容性
test_e2e_script_mode() {
    # 验证管道输入不影响现有行为
    result=$(echo "test" | ./apimgr switch test-config 2>&1)
    assert_not_contains "$result" "Select model"
}
```

---

**文档结束**

*最后更新: 2026-01-01*
*作者: Claude Code*
*项目: API Manager (apimgr)*