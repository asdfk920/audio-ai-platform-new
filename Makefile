.PHONY: help start stop clean build test lint ci-local docker-up docker-down migrate-up migrate-down goctl-install start-admin start-services swagger swagger-user swagger-device swagger-content swagger-export

# 默认目标
help:
	@echo "Available commands:"
	@echo "  make start         - 🚀 一键启动整个项目（推荐）"
	@echo "  make stop          - 停止所有服务"
	@echo "  make start-admin   - 启动 go-admin 后台"
	@echo "  make start-services- 启动所有微服务"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make build         - Build all services"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make ci-local      - Lint+test+build (Linux/macOS/Git Bash: scripts/ci-local.sh)"
	@echo "  make docker-up     - Start docker-compose services"
	@echo "  make docker-down   - Stop docker-compose services"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback database migrations"
	@echo "  make goctl-install - Install goctl tool"
	@echo "  make swagger       - Generate Swagger docs (all APIs)"
	@echo "  make swagger-user  - Generate Swagger for user service"
	@echo "  make swagger-device - Generate Swagger for device service"
	@echo "  make swagger-content - Generate Swagger for content service"
	@echo "  make swagger-export  - Export ReDoc HTML+PDF under doc/swagger/export/ (needs Node npx)"

# 安装 goctl
goctl-install:
	@echo "Installing goctl..."
	@go install github.com/zeromicro/go-zero/tools/goctl@latest
	@goctl --version

# 启动 Docker 环境
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Services are ready!"

# 停止 Docker 环境
docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

# 🚀 一键启动整个项目
start: docker-up
	@echo "=========================================="
	@echo "🚀 启动整个项目..."
	@echo "=========================================="
	@echo ""
	@echo "📦 1. Docker 服务已启动"
	@echo "   - PostgreSQL: localhost:5433 (docker-compose 5433->5432)"
	@echo "   - Redis: localhost:6379"
	@echo ""
	@echo "🔧 2. 启动 go-admin 后台..."
	@mkdir -p $(PWD)/logs
	@cd admin && go run main.go server -c config/settings.yml > $(PWD)/logs/admin.log 2>&1 & echo $$! > $(PWD)/logs/admin.pid
	@sleep 3
	@echo "   ✅ go-admin 后台: http://localhost:8000"
	@echo ""
	@echo "🎯 3. 启动微服务..."
	@cd services/user && go run user.go > $(PWD)/logs/user-service.log 2>&1 & echo $$! > $(PWD)/logs/user-service.pid
	@sleep 2
	@cd services/device && go run device.go > $(PWD)/logs/device-service.log 2>&1 & echo $$! > $(PWD)/logs/device-service.pid
	@sleep 2
	@cd services/content && go run content.go > $(PWD)/logs/content-service.log 2>&1 & echo $$! > $(PWD)/logs/content-service.pid
	@echo "   ✅ 用户服务: http://localhost:8001"
	@echo "   ✅ 设备服务: http://localhost:8002"
	@echo "   ✅ 内容服务: http://localhost:8003"
	@echo ""
	@echo "=========================================="
	@echo "✅ 所有服务启动完成！"
	@echo "=========================================="
	@echo ""
	@echo "📝 访问地址："
	@echo "   - go-admin 后台: http://localhost:8000"
	@echo "   - go-admin 前端: http://localhost:9527 (需手动启动)"
	@echo "   - 用户服务: http://localhost:8001"
	@echo "   - 设备服务: http://localhost:8002"
	@echo "   - 内容服务: http://localhost:8003"
	@echo ""
	@echo "📋 日志文件："
	@echo "   - logs/admin.log"
	@echo "   - logs/user-service.log"
	@echo "   - logs/device-service.log"
	@echo "   - logs/content-service.log"
	@echo ""
	@echo "🛑 停止服务: make stop"
	@echo ""

# 启动 go-admin 后台
start-admin:
	@echo "启动 go-admin 后台..."
	@mkdir -p logs
	@cd admin && go run main.go server -c config/settings.yml > ../logs/admin.log 2>&1 & echo $$! > ../logs/admin.pid
	@sleep 3
	@echo "✅ go-admin 后台已启动: http://localhost:8000"
	@echo "📋 日志: logs/admin.log"

# 启动所有微服务
start-services:
	@echo "启动所有微服务..."
	@mkdir -p logs
	@cd services/user && go run user.go > ../../logs/user-service.log 2>&1 & echo $$! > ../../logs/user-service.pid
	@sleep 2
	@cd services/device && go run device.go > ../../logs/device-service.log 2>&1 & echo $$! > ../../logs/device-service.pid
	@sleep 2
	@cd services/content && go run content.go > ../../logs/content-service.log 2>&1 & echo $$! > ../../logs/content-service.pid
	@echo "✅ 用户服务: http://localhost:8001"
	@echo "✅ 设备服务: http://localhost:8002"
	@echo "✅ 内容服务: http://localhost:8003"

# 停止所有服务
stop:
	@echo "=========================================="
	@echo "🛑 停止所有服务..."
	@echo "=========================================="
	@echo ""
	@echo "停止 Go 服务进程..."
	@pkill -9 -f "go run main.go server" || true
	@pkill -9 -f "go run user.go" || true
	@pkill -9 -f "go run device.go" || true
	@pkill -9 -f "go run content.go" || true
	@lsof -ti:8000 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8001 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8002 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8003 | xargs kill -9 2>/dev/null || true
	@rm -f logs/*.pid
	@sleep 1
	@echo "停止 Docker 服务..."
	@docker-compose down
	@echo ""
	@echo "✅ 所有服务已停止"

# 清理
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# 构建所有服务
build:
	@echo "Building all services..."
	@mkdir -p bin
	@go build -o bin/user-service ./services/user
	@go build -o bin/device-service ./services/device
	@go build -o bin/content-service ./services/content
	@go build -o bin/media-processing ./services/media-processing
	@go build -o bin/ai-worker ./services/ai-worker
	@echo "Build complete!"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v -race -cover $(shell go list ./... | grep -v 'go-admin/common/file_store')

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 与 CI 对齐的本地检查（子模块 lint / 合并 coverage / 构建四服务）；Windows 请用 scripts/ci-local.ps1
ci-local:
	@bash scripts/ci-local.sh

# 数据库迁移（上）：001–007 全量（含 password_algo，注册写库必需）
migrate-up:
	@echo "Running database migrations (001..007)..."
	@bash scripts/db/apply-all-migrations.sh

# 数据库迁移（下）：仅回滚 001（历史行为；完整回滚请按需执行各 *_down.sql）
migrate-down:
	@echo "Rolling back database migrations..."
	@psql postgresql://admin:admin123@localhost:5433/audio_platform -f scripts/db/migrations/001_init_down.sql

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# 生成 API 代码
gen-api:
	@echo "Generating API code..."
	@cd services/user && goctl api go -api user.api -dir .
	@cd services/device && goctl api go -api device.api -dir .
	@cd services/content && goctl api go -api content.api -dir .

# 生成 Swagger 文档（由 goctl 从 *.api 生成 swagger.json）
swagger: swagger-user swagger-device swagger-content
	@echo "✅ Swagger docs generated under doc/swagger/"

swagger-user:
	@echo "Generating Swagger for user..."
	@go run github.com/zeromicro/go-zero/tools/goctl@latest api swagger --api services/user/user.api --dir doc/swagger/user --filename user

swagger-device:
	@echo "Generating Swagger for device..."
	@go run github.com/zeromicro/go-zero/tools/goctl@latest api swagger --api services/device/device.api --dir doc/swagger/device --filename device

swagger-content:
	@echo "Generating Swagger for content..."
	@go run github.com/zeromicro/go-zero/tools/goctl@latest api swagger --api services/content/content.api --dir doc/swagger/content --filename content

swagger-admin:
	@echo "Generating Swagger for go-admin..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@cd admin && swag init --parseDependency --parseDepth=6 --instanceName admin -o ./docs/admin
	@mkdir -p doc/swagger/admin
	@cp -f admin/docs/admin/admin_swagger.json doc/swagger/admin/admin.json
	@cp -f admin/docs/admin/admin_swagger.yaml doc/swagger/admin/admin.yaml

# ReDoc 单页 HTML + PDF（需 Node.js；先 swagger + swagger-admin 再导出可包含 admin）
swagger-export: swagger swagger-admin
	@mkdir -p doc/swagger/export
	@npx --yes redoc-cli bundle doc/swagger/user/user.json -o doc/swagger/export/user-api.html
	@npx --yes redoc-cli bundle doc/swagger/user/user.json -o doc/swagger/export/user-api.pdf
	@npx --yes redoc-cli bundle doc/swagger/device/device.json -o doc/swagger/export/device-api.html
	@npx --yes redoc-cli bundle doc/swagger/device/device.json -o doc/swagger/export/device-api.pdf
	@npx --yes redoc-cli bundle doc/swagger/content/content.json -o doc/swagger/export/content-api.html
	@npx --yes redoc-cli bundle doc/swagger/content/content.json -o doc/swagger/export/content-api.pdf
	@npx --yes redoc-cli bundle doc/swagger/admin/admin.json -o doc/swagger/export/admin-api.html
	@npx --yes redoc-cli bundle doc/swagger/admin/admin.json -o doc/swagger/export/admin-api.pdf
	@echo "✅ ReDoc export: doc/swagger/export/*.html *.pdf"
