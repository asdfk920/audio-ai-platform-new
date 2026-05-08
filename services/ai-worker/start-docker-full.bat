@echo off
echo ========================================
echo   AI Worker Docker 启动脚本（完整版）
echo   - 挂载输出目录
echo   - 挂载缓存目录
echo   - 挂载音频文件目录 ⭐
echo ========================================

:: 获取当前脚本所在目录
set "WORKER_DIR=%~dp0"
cd /d %WORKER_DIR%

:: 设置主机路径
set "HOST_OUTPUT=%WORKER_DIR%output"
set "HOST_CACHE=%WORKER_DIR%cache"
set "HOST_AUDIO=%WORKER_DIR%audio-files"

echo [INFO] 主机输出目录: %HOST_OUTPUT%
echo [INFO] 主机缓存目录: %HOST_CACHE%
echo [INFO] 主机音频目录: %HOST_AUDIO% ⭐
echo.

:: 创建目录
mkdir "%HOST_OUTPUT%" 2>nul
mkdir "%HOST_CACHE%" 2>nul
mkdir "%HOST_AUDIO%" 2>nul

:: 检查是否已有运行中的容器
docker ps -q --filter "name=ai-worker" | findstr . >nul
if %errorlevel%==0 (
    echo [WARN] 检测到已运行的 ai-worker 容器，正在停止...
    docker stop ai-worker
    docker rm ai-worker
)

:: 启动容器（挂载所有必要目录）
echo [INFO] 正在启动 AI Worker 容器...
echo.
echo 📁 目录映射:
echo    %HOST_OUTPUT%     → /app/output
echo    %HOST_CACHE%      → /app/cache
echo    %HOST_AUDIO%      → /app/audio-files ⭐
echo.

docker run -d ^
  --name ai-worker ^
  --gpus all ^
  -p 8004:8004 ^
  -v "%HOST_OUTPUT%:/app/output" ^
  -v "%HOST_CACHE%:/app/cache" ^
  -v "%HOST_AUDIO%:/app/audio-files" ^
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
echo 🎵 音频文件: %HOST_AUDIO% (容器内: /app/audio-files)
echo 🔗 WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
echo 💡 使用说明:
echo    1. 将音频文件放入 audio-files 目录
echo    2. 数据库 audio_url 设为: /app/audio-files/xxx.mp3
echo    3. AI 直接读取本地文件，无需 HTTP 下载
echo ========================================

pause
