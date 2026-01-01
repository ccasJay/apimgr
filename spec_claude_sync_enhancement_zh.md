# APIMgr Claude Code 配置同步增强规范

## 1. 概述

### 1.1 现状
目前，`apimgr` 工具在将配置同步到 Claude Code 设置文件（`~/.claude/settings.json` 和 `./.claude/settings.json`）时执行**完全文件覆盖**。这种方法：
- 将整个 JSON 文件读入内存
- 重新序列化完整的 JSON 结构
- 使用 `os.WriteFile()` 重写整个文件

### 1.2 问题陈述
当前实现存在几个关键问题：
1. **丢失用户注释和注释**（JSON 标准库不保留注释）
2. **格式变化**（缩进、键顺序可能被更改）
3. **数据损坏风险**（如果 JSON 解析失败）
4. **覆盖 `env` 字段中的非 ANTHROPIC 环境变量**
5. **破坏性操作**（无回滚机制）

### 1.3 目标
创建一个**精确的更新机制**，能够：
- ✅ 仅更新 `env` 字段
- ✅ 保留**所有其他字段、注释和格式**
- ✅ 保持向后兼容性
- ✅ 增加安全机制（备份、验证）
- ✅ 提供更好的错误恢复

## 2. 技术需求

### 2.1 功能需求
| 需求 | 优先级 | 描述 |
|------|--------|------|
| FR-001 | P0 | 仅更新 `env` 字段中的 ANTHROPIC 相关键 |
| FR-002 | P0 | 保持所有其他字段不变 |
| FR-003 | P0 | 保持与现有 API 的向后兼容性 |
| FR-004 | P1 | 修改前创建备份 |
| FR-005 | P1 | 更新前后验证 JSON |
| FR-006 | P1 | 支持全局和项目级配置 |
| FR-007 | P2 | 提供用于测试的 dry-run 模式 |
| FR-008 | P2 | 记录同步操作日志 |

### 2.2 非功能需求
| 需求 | 优先级 | 描述 |
|------|--------|------|
| NFR-001 | P0 | 保留 JSON 注释和格式 |
| NFR-002 | P0 | 原子更新（全有或全无） |
| NFR-003 | P1 | 性能：典型文件 <100ms |
| NFR-004 | P1 | 安全：保持文件权限（0600） |
| NFR-005 | P2 | 容错：优雅的错误恢复 |

## 3. 设计架构

### 3.1 高层方法
```
当前：读取 → 解析 → 修改整个对象 → 序列化 → 写入
提议：读取 → 定位 'env' 字段 → 更新特定键 → 写入部分更新
```

### 3.2 组件设计

#### 3.2.1 JSON 处理策略
```go
// 策略 1：JSON 感知库（推荐）
import "github.com/tidwall/gjson"
import "github.com/tidwall/sjson"

// 策略 2：自定义 JSON 解析，保留注释
type JSONProcessor struct {
    preserveComments bool
    indentStyle      string
}

// 策略 3：使用 JSON5 解析器（如果 Claude Code 使用 JSON5）
import "github.com/sanity-io/litter"
```

#### 3.2.2 更新流程
```
1. 读取原始文件内容（保留注释）
2. 提取当前 'env' 字段
3. 应用 ANTHROPIC 更新：
   - 删除：ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, ANTHROPIC_BASE_URL, ANTHROPIC_MODEL
   - 设置：仅来自 APIConfig 的非空值
4. 验证新的 'env' 结构
5. 创建备份文件
6. 写入更新后的内容
7. 验证写入成功
8. 成功时清理备份
```

### 3.3 文件备份策略
```go
type BackupManager struct {
    // 备份命名：settings.json.backup-<timestamp>-<pid>
    backupPattern string = "%s.backup-%s-%d"

    // 保留策略
    maxBackups     int    = 3
    cleanupOldBackups()
}
```

## 4. 详细实现

### 4.1 更新的函数签名
```go
// 当前
func (cm *Manager) syncClaudeSettings(cfg *APIConfig) error

// 提议的增强选项
type SyncOptions struct {
    DryRun        bool      // 仅验证，不写入
    CreateBackup  bool      // 更新前创建备份
    PreserveOther bool      // 保留非 ANTHROPIC 环境变量
}

func (cm *Manager) syncClaudeSettingsEx(cfg *APIConfig, opts SyncOptions) (string, error)
```

### 4.2 核心更新逻辑
```go
func updateEnvField(originalContent string, cfg *APIConfig, opts SyncOptions) (string, error) {
    // 1. 解析但保留结构
    result := gjson.Parse(originalContent)

    // 2. 获取当前 env 字段（如果存在）
    envPath := "env"
    currentEnv := result.Get(envPath)

    // 3. 创建更新的 env 映射
    updatedEnv := make(map[string]interface{})
    if currentEnv.Exists() {
        // 保留现有的非 ANTHROPIC 字段
        currentEnv.ForEach(func(key, value gjson.Result) bool {
            keyStr := key.String()
            if !strings.HasPrefix(keyStr, "ANTHROPIC_") || opts.PreserveOther {
                updatedEnv[keyStr] = value.Value()
            }
            return true
        })
    }

    // 4. 设置新的 ANTHROPIC 值（仅非空值）
    if cfg.APIKey != "" {
        updatedEnv["ANTHROPIC_API_KEY"] = cfg.APIKey
    }
    if cfg.AuthToken != "" {
        updatedEnv["ANTHROPIC_AUTH_TOKEN"] = cfg.AuthToken
    }
    if cfg.BaseURL != "" {
        updatedEnv["ANTHROPIC_BASE_URL"] = cfg.BaseURL
    }
    if cfg.Model != "" {
        updatedEnv["ANTHROPIC_MODEL"] = cfg.Model
    }

    // 5. 使用精确的 JSON 修改应用更新
    updatedContent, err := sjson.SetRaw(originalContent, envPath, marshalWithComments(updatedEnv))
    if err != nil {
        return "", fmt.Errorf("更新 env 字段失败: %w", err)
    }

    return updatedContent, nil
}
```

### 4.3 原子更新模式
```go
func atomicFileUpdate(filePath string, newContent string, createBackup bool) error {
    // 1. 在同一目录中创建临时文件
    tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".*.tmp")
    if err != nil {
        return fmt.Errorf("创建临时文件失败: %w", err)
    }
    defer os.Remove(tempFile.Name())

    // 2. 将新内容写入临时文件
    if err := os.WriteFile(tempFile.Name(), []byte(newContent), 0600); err != nil {
        return fmt.Errorf("写入临时文件失败: %w", err)
    }

    // 3. 如果请求，创建备份
    if createBackup {
        backupPath := fmt.Sprintf("%s.backup-%d", filePath, time.Now().Unix())
        if err := copyFile(filePath, backupPath); err != nil {
            return fmt.Errorf("创建备份失败: %w", err)
        }
    }

    // 4. 原子重命名（跨平台）
    if err := os.Rename(tempFile.Name(), filePath); err != nil {
        // 如果存在，尝试从备份恢复
        if restoreErr := restoreFromBackup(filePath); restoreErr != nil {
            return fmt.Errorf("更新失败且恢复失败: %w, 恢复错误: %v", err, restoreErr)
        }
        return fmt.Errorf("更新失败，已从备份恢复: %w", err)
    }

    return nil
}
```

## 5. 安全机制

### 5.1 更新前验证
```go
func validateJSONUpdate(original, updated string) error {
    // 1. 确保 JSON 结构有效
    if !json.Valid([]byte(updated)) {
        return errors.New("更新后的内容不是有效的 JSON")
    }

    // 2. 确保只有 'env' 字段发生变化
    originalMap, updatedMap := parseToMaps(original, updated)

    diff := deepCompare(originalMap, updatedMap)
    if len(diff) > 1 || (len(diff) == 1 && diff[0] != "env") {
        return fmt.Errorf("'env' 字段之外的意外变更: %v", diff)
    }

    // 3. 确保没有数据丢失（除故意的 ANTHROPIC 删除）
    originalEnv := extractEnv(original)
    updatedEnv := extractEnv(updated)

    for key, value := range originalEnv {
        if strings.HasPrefix(key, "ANTHROPIC_") {
            continue // 允许删除
        }
        if updatedValue, exists := updatedEnv[key]; !exists || updatedValue != value {
            return fmt.Errorf("非 ANTHROPIC 字段 '%s' 被修改或删除", key)
        }
    }

    return nil
}
```

### 5.2 回滚策略
```go
type UpdateTransaction struct {
    FilePath      string
    BackupPath    string
    OriginalHash  string // 用于完整性验证
    Status        UpdateStatus
}

func (tx *UpdateTransaction) Rollback() error {
    if tx.Status == Committed {
        // 尝试从备份恢复
        if err := copyFile(tx.BackupPath, tx.FilePath); err != nil {
            return fmt.Errorf("回滚失败: %w", err)
        }

        // 验证恢复内容的完整性
        if !verifyFileHash(tx.FilePath, tx.OriginalHash) {
            return errors.New("回滚完整性检查失败")
        }

        tx.Status = RolledBack
        return nil
    }

    return errors.New("事务未处于已提交状态")
}
```

## 6. 迁移计划

### 6.1 第一阶段：准备（第 1-2 周）
1. **添加新依赖**
   ```bash
   go get github.com/tidwall/gjson
   go get github.com/tidwall/sjson
   ```
2. **创建功能标志**
   ```go
   const FeatureSurgicalUpdates = "surgical_updates_v1"
   ```
3. **添加当前问题的遥测**
   ```go
   type SyncMetrics struct {
       FullOverwrites   int64
       CommentLossCount int64
       FormatChanges    int64
   }
   ```

### 6.2 第二阶段：实施（第 3-4 周）
1. **实现核心更新逻辑**
   - `updateEnvField()` 函数
   - `atomicFileUpdate()` 函数
   - 备份管理系统
2. **添加验证层**
   - JSON 结构验证
   - 变更影响分析
   - 完整性检查

### 6.3 第三阶段：测试（第 5 周）
1. **单元测试** 所有新函数
2. **集成测试** 使用真实的 JSON 文件
3. **性能基准测试**
4. **向后兼容性验证**

### 6.4 第四阶段：推出（第 6 周）
1. **金丝雀部署**（10% 的用户）
2. **监控** 错误和性能
3. **逐步推出** 到 100%
4. **废弃旧方法** 稳定后

## 7. 测试策略

### 7.1 测试场景
| 场景 | 输入 | 预期结果 |
|------|------|----------|
| TS-001 | 带注释的文件 | 注释被保留 |
| TS-002 | 缺少 'env' 字段 | 创建 'env' 字段 |
| TS-003 | 现有的 ANTHROPIC 字段 | 正确更新 |
| TS-004 | 非 ANTHROPIC 环境变量 | 保持不变 |
| TS-005 | 无效的 JSON | 错误，文件不变 |
| TS-006 | 权限问题 | 错误，文件不变 |
| TS-007 | 大文件 (>1MB) | 更新在 200ms 内 |

### 7.2 测试数据
```json
// 测试用例：带注释的复杂 settings.json
{
  // 用户的自定义注释
  "env": {
    "ANTHROPIC_API_KEY": "old-key",
    "USER_VAR": "should_preserve",
    "OTHER_SETTING": 123
  },

  "plugins": {
    "enabled": ["plugin1", "plugin2"]
  },

  /* 多行注释
     应该被保留 */
  "uiSettings": {
    "theme": "dark"
  }
}
```

## 8. 风险评估

### 8.1 技术风险
| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| JSON 解析错误 | 中 | 高 | 预验证、备份恢复 |
| 性能下降 | 低 | 中 | 基准测试、优化 |
| 第三方库问题 | 中 | 中 | 版本锁定、回退到旧方法 |
| 跨平台兼容性 | 低 | 高 | 广泛平台测试 |

### 8.2 运营风险
| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 数据损坏 | 低 | 关键 | 原子更新、备份系统 |
| 用户中断 | 中 | 中 | 逐步推出、回滚能力 |
| 监控缺失 | 低 | 中 | 全面的日志记录、警报 |

## 9. 成功指标

### 9.1 量化指标
| 指标 | 目标 | 测量方式 |
|------|------|----------|
| 注释保留率 | 100% | 自动检查 |
| 更新成功率 | >99.9% | 监控系统 |
| 平均更新时间 | <100ms | 性能测试 |
| 备份完整性率 | 100% | 哈希验证 |

### 9.2 定性指标
1. **用户满意度** 保留格式
2. **开发者信心** 更新安全性
3. **可维护性** 新代码库
4. **文档完整性** 新功能

## 10. 附录

### 10.1 依赖项
```toml
# go.mod 新增
github.com/tidwall/gjson v1.17.0
github.com/tidwall/sjson v1.2.5
github.com/stretchr/testify v1.9.0  # 用于增强测试
```

### 10.2 性能基准
```bash
# 基准测试命令
go test -bench=. -benchtime=10s ./config/

# 预期结果
# BenchmarkSurgicalUpdate-8   50000   185.2 ns/op
# BenchmarkFullOverwrite-8    30000   412.7 ns/op
```

### 10.3 回滚计划
```
步骤 1：禁用功能标志
步骤 2：恢复到原始实现
步骤 3：通知用户安全回滚
步骤 4：安排修复后重新实施
```

---

## 实施优先级摘要

| 优先级 | 组件 | 估计工作量 | 依赖项 |
|--------|------|------------|--------|
| P0 | 使用 gjson/sjson 的核心更新逻辑 | 3 天 | 无 |
| P0 | 原子文件更新机制 | 2 天 | 核心逻辑 |
| P1 | 备份和恢复系统 | 2 天 | 原子更新 |
| P1 | 验证和安全检查 | 3 天 | 上述所有 |
| P2 | Dry-run 和诊断模式 | 1 天 | 验证 |
| P2 | 增强的日志记录和指标 | 1 天 | 上述所有 |

**总估计开发时间**：12 人日
**推荐时间表**：4 周，包括测试和推出