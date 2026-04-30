# Windows 本地启动指南

在 Windows 上不依赖 Make，用 PowerShell 一键启动/停止本项目的步骤。

## 前置要求

- **Go** 1.22+，已加入 PATH
- **Docker Desktop** 已安装并处于运行状态
- **PowerShell** 5.1 或 PowerShell Core 7+

## 一键启动

在项目根目录（`audio-ai-platform`）打开 PowerShell，执行：

```powershell
.\scripts\start-windows.ps1
```

若提示“无法加载，因为在此系统上禁止运行脚本”，可先执行：

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

或直接绕过策略运行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-windows.ps1
```

脚本会自动完成：

1. 启动 Docker（PostgreSQL、Redis、LocalStack）
2. 执行数据库迁移（通过 Docker，无需本机安装 psql）
3. 启动 go-admin 后台（端口 8000）
4. 启动用户 / 设备 / 内容 三个微服务（8001、8002、8003）

## 一键停止

```powershell
.\scripts\stop-windows.ps1
```

会结束上述 Go 进程并执行 `docker-compose down`。

## 分步启动（可选）

若希望分步操作或排查问题，可手动执行：

```powershell
# 1. 进入项目根目录
cd C:\Users\你的用户名\Desktop\audio-ai-platform

# 2. 启动 Docker
docker-compose up -d
Start-Sleep -Seconds 5

# 3. 执行数据库迁移（无需本机 psql）
Get-Content .\scripts\db\migrations\001_init.sql -Raw | docker exec -i audio-platform-postgres psql -U admin -d audio_platform

# 4. 创建日志目录
New-Item -ItemType Directory -Force -Path logs

# 5. 启动 go-admin（新开窗口或后台）
Start-Process -FilePath "go" -ArgumentList "run", "main.go", "server", "-c", "config/settings.yml" -WorkingDirectory admin

# 6. 启动三个微服务（各开一个窗口或重定向到 logs 下对应 .log）
Start-Process -FilePath "go" -ArgumentList "run", "user.go"   -WorkingDirectory services\user
Start-Process -FilePath "go" -ArgumentList "run", "device.go"  -WorkingDirectory services\device
Start-Process -FilePath "go" -ArgumentList "run", "content.go" -WorkingDirectory services\content
```

## 访问地址

| 服务         | 地址                    |
|--------------|-------------------------|
| go-admin 后台 | http://localhost:8000   |
| go-admin 前端 | http://localhost:9527（需在 `admin-ui` 下执行 `npm run dev`） |
| 用户服务     | http://localhost:8001   |
| 设备服务     | http://localhost:8002   |
| 内容服务     | http://localhost:8003   |

## 日志

日志目录：项目根目录下的 `logs\`。

- `admin.log` — go-admin 后台
- `user-service.log` — 用户服务
- `device-service.log` — 设备服务
- `content-service.log` — 内容服务

## 常见问题

- **docker-compose 报错**：确认 Docker Desktop 已启动，且 WSL2 或 Hyper-V 正常。
- **迁移报错或表已存在**：可忽略；若表未创建，可单独再执行一次上面的 `Get-Content ... | docker exec ...` 命令。
- **停止脚本释放端口时提示权限不足**：以管理员身份运行 PowerShell 再执行 `.\scripts\stop-windows.ps1`，或先手动关闭占用 8000/8001/8002/8003 的程序后再运行脚本。
