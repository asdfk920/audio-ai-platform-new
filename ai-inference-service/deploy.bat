@echo off
REM BSRoformer SCNet 音轨分离服务 - Windows 快速部署脚本

echo ╔════════════════════════════════════════════╗
echo ║  BSRoformer SCNet 音轨分离服务部署         ║
echo ╚════════════════════════════════════════════╝
echo.

REM 检查 Docker 是否安装
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 未检测到 Docker，请先安装 Docker Desktop
    echo 下载地址：https://www.docker.com/products/docker-desktop
    pause
    exit /b 1
)

echo [1/5] 检查 Docker 状态...
docker version >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [错误] Docker 未运行，请启动 Docker Desktop
    pause
    exit /b 1
)
echo [✓] Docker 运行正常

echo.
echo [2/5] 创建必要目录...
if not exist "models" mkdir models
if not exist "data\input" mkdir data\input
if not exist "data\output" mkdir data\output
echo [✓] 目录创建完成

echo.
echo [3/5] 复制环境配置文件...
if not exist ".env" (
    copy .env.example .env >nul
    echo [✓] .env 文件已创建
) else (
    echo [✓] .env 文件已存在
)

echo.
echo [4/5] 构建 Docker 镜像...
docker-compose build
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 镜像构建失败
    pause
    exit /b 1
)
echo [✓] 镜像构建完成

echo.
echo [5/5] 启动服务...
docker-compose up -d
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 服务启动失败
    pause
    exit /b 1
)
echo [✓] 服务启动成功

echo.
echo ╔════════════════════════════════════════════╗
echo ║  部署完成！                                ║
echo ╚════════════════════════════════════════════╝
echo.
echo 服务信息:
echo   - API 地址：http://localhost:8004
echo   - 健康检查：http://localhost:8004/health
echo   - 文档地址：http://localhost:8004/docs
echo.
echo 常用命令:
echo   - 查看日志：docker-compose logs -f
echo   - 停止服务：docker-compose down
echo   - 重启服务：docker-compose restart
echo.
echo 测试服务:
echo   python test_api.py
echo.

pause
