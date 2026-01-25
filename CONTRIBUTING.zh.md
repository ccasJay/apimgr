# 贡献指南

[English version](CONTRIBUTING.md)

感谢您对 apimgr 项目的兴趣！我们欢迎各种形式的贡献。

## 贡献方式

### 1. 报告问题
如果您发现了 bug 或有功能建议，请通过 GitHub Issues 提交。在提交时，请包含：
- 清晰的问题描述
- 重现步骤（如果是 bug）
- 预期行为
- 实际行为
- 环境信息（操作系统、Go 版本等）

### 2. 提交 Pull Request
1. **Fork 仓库**：点击 GitHub 页面右上角的 "Fork" 按钮
2. **克隆仓库**：`git clone https://github.com/your-username/apimgr.git`（将 `your-username` 替换为你的 GitHub 用户名）
3. **创建分支**：`git checkout -b feature/your-feature-name`
4. **实现功能**：编写代码，确保符合项目的代码风格
5. **运行测试**：`go test ./...`
6. **运行 Lint**：`golangci-lint run`
7. **提交代码**：`git commit -m "feat: add your feature"`
8. **推送分支**：`git push origin feature/your-feature-name`
9. **创建 PR**：在 GitHub 页面上创建 Pull Request

## 开发流程

### 代码风格
- 使用 Go 官方推荐的代码风格（`go fmt`）
- 使用 `golint` 和 `staticcheck` 进行代码检查
- 注释清晰，尤其是公共函数和结构体
- 变量名采用驼峰式命名，避免缩写

### 测试
- 为新功能编写单元测试
- 确保测试覆盖率不低于 80%
- 运行 `go test ./...` 确保所有测试通过
- 使用表格驱动测试处理多个测试用例
- 使用有意义的测试名称描述测试内容

### 提交信息
- 使用常规提交格式：`type(scope): description`
- 类型：`feat`、`fix`、`docs`、`style`、`refactor`、`test`、`chore`
- 示例：`feat(ping): add streaming support for compatibility tests`
- 保持第一行在 72 个字符以内
- 如需详细说明，在提交正文中添加

### 文档
- 为新命令或功能更新 README.md
- 为复杂功能编写详细的使用文档
- 保持文档与代码一致

## 社区规范

请遵守 [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) 行为准则。

## 开发环境

### 依赖
- Go 1.21 或更高版本
- golangci-lint（用于代码检查）
- goreleaser（可选，用于发布构建）

### 环境设置

1. **Fork 并克隆仓库**
   ```bash
   # 将 'your-username' 替换为你的 GitHub 用户名
   git clone https://github.com/your-username/apimgr.git
   cd apimgr
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **构建项目**
   ```bash
   make build
   # 或
   go build -o apimgr .
   ```

4. **运行测试**
   ```bash
   go test ./...
   ```

5. **运行代码检查**
   ```bash
   golangci-lint run
   # 或使用项目配置
   golangci-lint run --config .golangci.yml
   ```

### 常用命令
```bash
# 构建二进制文件
make build
# 或
go build -o apimgr .

# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 运行测试并显示详细输出
go test -v ./...

# 运行特定测试
go test -run TestConfigLoad ./config

# 运行基准测试
go test -bench=. ./...

# 运行 Lint 检查
golangci-lint run

# 清理构建文件
make clean
# 或
rm -f apimgr

# 本地安装
make install
```

### 项目结构
```
apimgr/
├── cmd/           # CLI 命令
├── config/        # 配置管理
├── internal/      # 内部包
│   ├── providers/ # API 提供商实现
│   ├── tui/       # 终端 UI
│   └── utils/     # 工具函数
├── main.go        # 入口点
└── ...
```

详细的架构信息请参阅 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 许可证

apimgr 使用 MIT 许可证，您的贡献将自动获得相同的许可证。

感谢您的贡献！
