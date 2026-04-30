# CI/CD 文档

## 概述

本项目使用 GitHub Actions 实现持续集成和持续部署，包括代码检查、单元测试、API 集成测试和自动部署。

## CI 流程

### 触发条件

- **CI (持续集成)**：
  - Push 到 `develop` 或 `feature/**` 分支
  - Pull Request 到 `develop` 或 `main` 分支

- **CD (持续部署)**：
  - Push 到 `main` 分支
  - 创建 `v*` 标签

### CI 工作流程

CI 流程包含 4 个主要任务：

#### 1. Lint (代码检查)

- 使用 `golangci-lint` 进行代码质量检查
- 检查代码规范、潜在错误、性能问题等
- Go 版本：1.22

#### 2. Test (单元测试)

- 运行所有单元测试
- 生成代码覆盖率报告
- 上传覆盖率到 Codecov
- 测试环境：
  - PostgreSQL 15
  - Redis 7
  - 并发测试（race detector）

#### 3. Build (构建)

- 编译所有微服务和后台管理系统
- 生成可执行文件：
  - `bin/user-service` - 用户服务
  - `bin/device-service` - 设备服务
  - `bin/content-service` - 内容服务
  - `bin/admin` - go-admin 后台
- 上传构建产物供后续使用

#### 4. Integration Test (API 集成测试)

- 启动所有服务（用户、设备、内容、go-admin）
- 运行 API 集成测试脚本
- 测试所有关键接口的可用性
- 验证服务间的连通性

## API 集成测试

### 测试脚本

位置：`scripts/test-api.sh`

### 测试覆盖

#### 用户服务 (8001)
- ✅ 用户注册接口：`POST /api/v1/user/register`
- ✅ 用户登录接口：`POST /api/v1/user/login`
- 用户分页列表已迁至管理后台：`GET /api/v1/platform-user/list`（go-admin，需管理员 JWT）

#### 设备服务 (8002)
- ✅ 设备列表接口：`GET /api/v1/device/list`
- ✅ 设备绑定接口：`POST /api/v1/device/bind`
- ✅ 设备心跳接口：`POST /api/v1/device/heartbeat`

#### 内容服务 (8003)
- ✅ 内容列表接口：`GET /api/v1/content/list`
- ✅ 获取上传地址接口：`POST /api/v1/content/upload`
- ✅ 内容处理状态接口：`GET /api/v1/content/status`

#### go-admin 服务 (8000)
- ✅ 服务可访问性测试
- ✅ 登录接口：`POST /api/v1/login`

### 本地运行测试

```bash
# 确保所有服务已启动
./scripts/start-all.sh

# 运行 API 测试
./scripts/test-api.sh
```

### 测试环境变量

可以通过环境变量自定义服务地址：

```bash
export USER_SERVICE_URL=http://localhost:8001
export DEVICE_SERVICE_URL=http://localhost:8002
export CONTENT_SERVICE_URL=http://localhost:8003
export ADMIN_SERVICE_URL=http://localhost:8000

./scripts/test-api.sh
```

## CD 流程

### 部署流程

1. **配置 AWS 凭证**
   - 使用 GitHub Secrets 存储 AWS 访问密钥
   - 需要配置：
     - `AWS_ACCESS_KEY_ID`
     - `AWS_SECRET_ACCESS_KEY`

2. **登录 Amazon ECR**
   - 自动登录到 AWS ECR 容器镜像仓库

3. **构建和推送 Docker 镜像**
   - 构建各个服务的 Docker 镜像
   - 使用 git commit SHA 作为镜像标签
   - 推送到 ECR

4. **部署到 ECS**
   - 更新 ECS 服务（待实现）

### Docker 镜像

- `user-service` - 用户服务镜像
- `device-service` - 设备服务镜像
- `content-service` - 内容服务镜像

## GitHub Secrets 配置

需要在 GitHub 仓库设置中配置以下 Secrets：

### AWS 相关
- `AWS_ACCESS_KEY_ID` - AWS 访问密钥 ID
- `AWS_SECRET_ACCESS_KEY` - AWS 访问密钥

### Codecov (可选)
- `CODECOV_TOKEN` - Codecov 上传令牌

## 工作流文件

### CI 配置
文件：`.github/workflows/ci.yml`

```yaml
触发条件：
  - push: develop, feature/**
  - pull_request: develop, main

任务：
  1. lint - 代码检查
  2. test - 单元测试
  3. build - 构建服务
  4. integration-test - API 集成测试
```

### CD 配置
文件：`.github/workflows/cd.yml`

```yaml
触发条件：
  - push: main
  - tags: v*

任务：
  1. 构建 Docker 镜像
  2. 推送到 ECR
  3. 部署到 ECS
```

## 最佳实践

### 开发流程

1. **功能开发**
   ```bash
   git checkout -b feature/new-feature
   # 开发代码
   git push origin feature/new-feature
   ```
   - 自动触发 CI 检查

2. **创建 Pull Request**
   - 从 feature 分支到 develop
   - CI 自动运行所有测试
   - 代码审查通过后合并

3. **发布到生产**
   ```bash
   git checkout main
   git merge develop
   git tag v1.0.0
   git push origin main --tags
   ```
   - 自动触发 CD 部署

### 测试要求

- 所有新功能必须包含单元测试
- 新增 API 接口需要添加到集成测试脚本
- 代码覆盖率目标：> 70%
- 所有测试必须通过才能合并

### 代码质量

- 遵循 Go 代码规范
- 通过 golangci-lint 检查
- 无 race condition
- 合理的错误处理

## 故障排查

### CI 失败常见原因

1. **Lint 失败**
   - 运行 `golangci-lint run` 本地检查
   - 修复代码规范问题

2. **测试失败**
   - 检查测试日志
   - 本地运行 `go test -v ./...`
   - 确保数据库和 Redis 配置正确

3. **Go 版本问题**
   - 项目要求 Go 1.22+
   - 如遇到 "requires go >= 1.25" 错误，已通过 go.mod 中的 replace 指令修复
   - 使用 `go mod tidy` 更新依赖

4. **构建失败**
   - 检查依赖是否完整
   - 运行 `go mod tidy`
   - 确保所有导入路径正确

5. **集成测试失败**
   - 检查服务是否正常启动
   - 查看服务日志
   - 验证配置文件是否正确

### 本地调试

```bash
# 运行 lint
golangci-lint run

# 运行单元测试
go test -v -race ./...

# 构建所有服务
make build

# 启动服务并运行集成测试
./scripts/start-all.sh
./scripts/test-api.sh
```

## 性能优化

### 缓存策略

- Go modules 缓存：加速依赖下载
- 构建缓存：复用编译结果

### 并行执行

- Lint 和 Test 可以并行运行
- 多个服务的构建可以并行

### 资源限制

- CI 运行器：ubuntu-latest
- 超时设置：默认 60 分钟
- 数据库健康检查：确保服务就绪

## 未来改进

- [ ] 添加性能测试
- [ ] 集成安全扫描
- [ ] 自动化数据库迁移
- [ ] 蓝绿部署支持
- [ ] 回滚机制
- [ ] 监控和告警集成
- [ ] 多环境部署（dev/staging/prod）

## 相关文档

- [快速开始](./快速开始.md)
- [API 文档](./API文档.md)
- [部署文档](./部署文档.md)
- [开发规范](./开发规范.md)
