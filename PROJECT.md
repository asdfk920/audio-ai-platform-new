# Audio AI Platform - 项目概览

## 🎯 项目简介

Audio AI Platform 是一个基于 AWS 的音频 AI 处理平台，支持 IoT 设备音频播放、用户上传音频、云端 AI 推理和 CDN 分发。

## 📁 项目结构

```
audio-ai-platform/
├── services/              # 微服务目录
│   ├── user/             # 用户服务 (8001)
│   ├── device/           # 设备服务 (8002)
│   ├── content/          # 内容服务 (8003)
│   ├── media-processing/ # 媒体处理服务 (8004)
│   └── ai-worker/        # AI Worker
├── admin/                # 后台管理 (8000)
├── common/               # 公共代码
├── pkg/                  # 可复用库
│   ├── awsx/            # AWS 封装
│   ├── redisx/          # Redis 封装
│   └── logger/          # 日志封装
├── deploy/               # 部署配置
│   ├── docker/          # Dockerfile
│   ├── k8s/             # Kubernetes 配置
│   └── terraform/       # 基础设施代码
├── scripts/              # 脚本
│   ├── db/              # 数据库迁移
│   └── build/           # 构建脚本
├── doc/                  # 文档
└── .github/workflows/    # CI/CD
```

## 🚀 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/jacklau/audio-ai-platform.git
cd audio-ai-platform

# 2. 启动开发环境
make docker-up
make migrate-up

# 3. 启动服务
make dev

# 4. 验证
curl http://localhost:8001/health
```

## 📚 文档导航

| 文档 | 说明 |
|------|------|
| [快速开始](doc/快速开始.md) | 5 分钟快速启动指南 |
| [架构设计](doc/架构设计.md) | 系统架构详细设计 |
| [开发规范](doc/开发规范.md) | 代码规范和最佳实践 |
| [API 文档](doc/API文档.md) | 完整的 API 接口文档 |
| [部署文档](doc/部署文档.md) | 部署和运维指南 |
| [项目总结](doc/项目搭建总结.md) | 项目搭建完成情况 |

## 🛠️ 技术栈

- **语言**: Go 1.22
- **框架**: go-zero, go-admin
- **数据库**: PostgreSQL, Redis
- **消息队列**: Kafka / SQS
- **云服务**: AWS (S3, ECS, Aurora, ElastiCache)
- **容器化**: Docker, Kubernetes
- **CI/CD**: GitHub Actions

## 🏗️ 核心功能

### 用户服务
- 用户注册/登录
- JWT 认证
- 权限管理

### 设备服务
- 设备绑定/解绑
- 设备状态监控
- 指令下发

### 内容服务
- 音频上传
- AI 处理
- CDN 分发

## 📊 数据库设计

11 张核心表：
- users, user_auth, roles, user_role_rel
- devices, device_user_rel, device_commands
- raw_contents, processed_contents, contents
- content_play_records

## 🔧 常用命令

```bash
make dev           # 启动开发环境
make stop          # 停止所有服务
make test          # 运行测试
make lint          # 代码检查
make build         # 构建所有服务
make docker-up     # 启动 Docker 服务
make migrate-up    # 运行数据库迁移
```

## 🌐 服务端口

| 服务 | 端口 |
|------|------|
| 用户服务 | 8001 |
| 设备服务 | 8002 |
| 内容服务 | 8003 |
| 媒体处理 | 8004 |
| 后台管理 | 8000 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| Kafka | 9092 |
| LocalStack | 4566 |

## 📝 开发流程

1. 从 `develop` 创建 `feature/xxx` 分支
2. 开发并提交代码
3. 运行 `make test` 和 `make lint`
4. 创建 PR 到 `develop`
5. Code Review 通过后合并

## 🔐 环境变量

```bash
DATABASE_URL=postgresql://admin:admin123@localhost:5432/audio_platform
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=redis123
AWS_REGION=us-east-1
AWS_ENDPOINT=http://localhost:4566  # LocalStack
```

## 📈 下一步

- [ ] 实现业务逻辑
- [ ] 编写单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 安全加固

## 📞 联系方式

- 项目负责人: [Your Name]
- 邮箱: [Your Email]
- GitHub: https://github.com/jacklau/audio-ai-platform

## 📄 许可证

MIT License
