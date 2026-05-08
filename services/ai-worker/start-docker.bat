@echo off
echo ========================================
echo   AI Worker Docker 启动脚本（带目录挂载）
echo ========================================

:: 获取当前脚本所在目录
set "WORKER_DIR=%~dp0"
cd /d %WORKER_DIR%

:: 设置主机输出路径（Windows 绝对路径）
set "HOST_OUTPUT=%WORKER_DIR%output"
set "HOST_CACHE=%WORKER_DIR%cache"

echo [INFO] 主机输出目录: %HOST_OUTPUT%
echo [INFO] 主机缓存目录: %HOST_CACHE%
echo.

:: 检查是否已有运行中的容器
docker ps -q --filter "name=ai-worker" | findstr . >nul
if %errorlevel%==0 (
    echo [WARN] 检测到已运行的 ai-worker 容器，正在停止...
    docker stop ai-worker
    docker rm ai-worker
)

:: 启动容器（挂载输出和缓存目录）
echo [INFO] 正在启动 AI Worker 容器...
docker run -d ^
  --name ai-worker ^
  --gpus all ^
  -p 8004:8004 ^
  -v "%HOST_OUTPUT%:/app/output" ^
  -v "%HOST_CACHE%:/app/cache" ^
  -v "%HOST_OUTPUT%\static-media:/app/static-media" ^
  ai-worker:v1

if %errorlevel% neq 0 (
    echo [ERROR] 容器启动失败！
    pause
    exit /b 1
)

echo.
echo ========================================
echo ✅ AI Worker 启动成功！
echo ========================================
echo 📍 服务地址: http://localhost:8004
echo 📁 输出目录: %HOST_OUTPUT%
echo 🗄️  缓存目录: %HOST_CACHE%
echo 🔗 WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
echo 💡 分离后的文件会保存在主机的 output 目录中
echo ========================================

pause
