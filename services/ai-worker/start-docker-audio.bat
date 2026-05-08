@echo off
echo ========================================
echo   AI Worker Docker 启动脚本
echo   音频文件挂载: D:\audio -> /audio ⭐
echo ========================================

:: 检查音频目录是否存在
if not exist "D:\audio" (
    echo [ERROR] 未找到 D:\audio 目录！
    echo 请先创建目录并放入音频文件：
    echo.
    echo   mkdir D:\audio
    echo   copy 你的音频.mp3 D:\audio\test1.mp3
    echo.
    pause
    exit /b 1
)

:: 检查测试文件
if not exist "D:\audio\test1.mp3" (
    echo [WARN] 未找到 D:\audio\test1.mp3
    echo 请确保已将音频文件放入该目录
    echo.
)

cd /d %~dp0

:: 停止旧容器
docker ps -q --filter "name=ai-worker" | findstr . >nul 2>&1
if %errorlevel%==0 (
    echo [INFO] 正在停止旧的 ai-worker 容器...
    docker stop ai-worker >nul 2>&1
    docker rm ai-worker >nul 2>&1
)

:: 启动容器（挂载 D:\audio 到容器的 /audio）
echo [INFO] 正在启动 AI Worker 容器...
echo.
echo 📁 目录映射:
echo    D:\audio          → /audio (只读)
echo    ./output          → /app/output (输出结果)
echo.

docker run -d ^
  --name ai-worker ^
  --gpus all ^
  -p 8004:8004 ^
  -v "D:\audio:/audio:ro" ^
  -v "%~dp0output:/app/output" ^
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
echo 🎵 音频目录: D:\audio → 容器内 /audio
echo 📁 输出目录: %~dp0output
echo 🔗 WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
echo 💡 数据库更新 SQL:
echo    UPDATE content SET audio_url = '/audio/test1.mp3' WHERE id = 2;
echo ========================================

pause
