# 贡献指南

[English version](CONTRIBUTING.en.md)

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
2. **克隆仓库**：`git clone https://github.com/your-username/apimgr.git`
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

### 文档
- 为新命令或功能更新 README.md
- 为复杂功能编写详细的使用文档
- 保持文档与代码一致

## 社区规范

请遵守 [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) 行为准则。

## 开发环境

### 依赖
- Go 1.21+
- golangci-lint
- goreleaser (用于发布)

### 常用命令
```bash
# 运行所有测试
go test ./...

# 运行 Lint 检查
golangci-lint run

# 构建二进制文件
go build

# 清理构建文件
go clean

# 生成文档
godoc -http=:6060
```

## 许可证

apimgr 使用 MIT 许可证，您的贡献将自动获得相同的许可证。

感谢您的贡献！
