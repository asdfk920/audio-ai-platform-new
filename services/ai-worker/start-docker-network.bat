@echo off
chcp 65001 >nul
echo ========================================
echo   AI Worker Docker 启动脚本
echo ========================================
echo.

:: 检查 Docker 是否运行
docker info >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker 未运行！请先启动 Docker Desktop
    pause
    exit /b 1
)

echo 请选择网络模式：
echo   [1] Host 网络模式（推荐 - 直接访问外网，性能最好）
echo   [2] Bridge 网络模式（生产环境 - 隔离性好）
echo   [3] 检查容器网络状态
echo   [4] 停止并删除容器
echo   [0] 退出
echo.

set /p choice=请输入选项 (0-4):

if "%choice%"=="1" goto host_mode
if "%choice%"=="2" goto bridge_mode
if "%choice%"=="3" goto check_network
if "%choice%"=="4" goto stop_container
if "%choice%"=="0" goto end

echo ⚠️ 无效选项
pause
goto end

:host_mode
echo.
echo 🚀 使用 Host 网络模式启动...
echo.
echo ✅ 特点：
echo    - 容器直接使用宿主机网络
echo    - 可直接访问外网（OSS、互联网）
echo    - 性能最佳，延迟最低
echo    - 端口：8004（API）、9090（监控）
echo.
docker-compose down 2>nul
docker-compose up -d --build ai-worker
if errorlevel 1 (
    echo ❌ 启动失败
    pause
    exit /b 1
)
echo.
echo ✅ 启动成功！
echo 📡 服务地址：
echo    API: http://localhost:8004/api/v1/health
echo    WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
goto show_logs

:bridge_mode
echo.
echo 🔗 使用 Bridge 网络模式启动...
echo.
echo ✅ 特点：
echo    - 独立网络命名空间，隔离性好
echo    - 配置 DNS + 静态 IP
echo    - 适合生产环境部署
echo.
docker-compose -f docker-compose.bridge.yml down 2>nul
docker-compose -f docker-compose.bridge.yml up -d --build ai-worker
if errorlevel 1 (
    echo ❌ 启动失败
    pause
    exit /b 1
)
echo.
echo ✅ 启动成功！
echo 📡 服务地址：
echo    API: http://localhost:8004/api/v1/health
echo    WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
goto show_logs

:check_network
echo.
echo 🔍 检查容器网络状态...
echo.
docker ps --filter "name=ai-worker" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo.
if exist "ai-worker" (
    echo 📊 网络配置：
    docker inspect ai-worker --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'
    echo.
    echo 🌐 测试外网连通性...
    docker exec ai-worker curl -sI https://www.aliyun.com --connect-timeout 5 || echo ❌ 无法访问外网
) else (
    echo ⚠️ 容器未运行
)
echo.
pause
goto end

:stop_container
echo.
echo 🛑 停止 AI Worker 容器...
docker-compose down 2>nul
docker-compose -f docker-compose.bridge.yml down 2>nul
echo ✅ 容器已停止并删除
echo.
pause
goto end

:show_logs
echo.
echo 📝 查看实时日志？（Y/N）
set /p view_logs=
if /i "%view_logs%"=="Y" (
    echo.
    echo 按 Ctrl+C 停止查看日志
    echo ========================================
    docker logs -f ai-worker --tail 100
)
echo.
goto end

:end
echo.
echo 👋 完成！
