@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   AI Worker Docker 挂载方案（100% 生效）
echo   -v D:\audio:/audio ⭐
echo ========================================
echo.

:: ==================== 步骤 0：检查 Docker ====================
echo [步骤 0] 检查 Docker 状态...
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker 未运行！
    echo.
    echo 请先启动 Docker Desktop：
    echo   1. 双击桌面上的 "Docker Desktop" 图标
    echo   2. 等待托盘图标变为稳定状态（约 30 秒）
    echo   3. 然后重新运行此脚本
    echo.
    pause
    exit /b 1
)
echo [INFO] Docker 已就绪 ✓
echo.

:: ==================== 步骤 1：停止并删除旧容器 ====================
echo [步骤 1] 停止并删除旧容器...
docker ps -q --filter "name=ai-worker" | findstr . >nul 2>&1
if %errorlevel%==0 (
    echo [INFO] 正在停止 ai-worker 容器...
    docker stop ai-worker >nul 2>&1
    docker rm ai-worker >nul 2>&1
    echo [INFO] 旧容器已删除 ✓
) else (
    echo [INFO] 未找到运行中的容器，跳过
)
echo.

:: ==================== 步骤 2：检查 D:\audio 目录 ====================
echo [步骤 2] 检查音频目录...
if not exist "D:\audio" (
    echo [WARN] D:\audio 目录不存在，正在创建...
    mkdir "D:\audio" 2>nul
)

if exist "D:\audio\test1.mp3" (
    echo [INFO] 找到测试文件: D:\audio\test1.mp3 ✓
) else (
    echo [WARN] 未找到 D:\audio\test1.mp3
    echo.
    echo 请手动复制音频文件：
    echo   copy "C:\Users\Lenovo\Downloads\file_example_MP3_700KB.mp3" "D:\audio\test1.mp3"
    echo.
    set /p continue=是否继续？（Y/N）:
    if /i not "%continue%"=="Y" exit /b 1
)
echo.

:: ==================== 步骤 3：启动新容器（关键挂载） ====================
echo [步骤 3] 启动新的 AI Worker 容器...
echo.
echo 🔑 关键挂载参数：-v D:\audio:/audio
echo    主机路径: D:\audio
echo    容器路径: /audio
echo.

docker run -d ^
  --name ai-worker ^
  --gpus all ^
  -p 8004:8004 ^
  -v "D:\audio:/audio" ^
  ai-worker:v1

if %errorlevel% neq 0 (
    echo [ERROR] 容器启动失败！
    pause
    exit /b 1
)
echo.
echo [INFO] 容器已启动 ✓
echo.

:: ==================== 步骤 4：验证挂载 ====================
echo [步骤 4] 验证挂载是否生效...
echo.
echo 📋 正在进入容器检查文件...

timeout /t 3 /nobreak >nul

docker exec ai-worker ls /audio/ >nul 2>&1
if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo ✅✅✅ 挂载成功！
    echo ========================================
    echo.
    echo 📁 容器内 /audio 目录内容：
    docker exec ai-worker ls -lh /audio/
    echo.
    echo 📍 服务地址: http://localhost:8004
    echo 🔗 WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
    echo.
    echo 💡 下一步操作：
    echo    1. 更新数据库: UPDATE content SET audio_url = '/audio/test1.mp3' WHERE id = 2;
    echo    2. 在 Apifox 测试 WebSocket 连接
    echo.
) else (
    echo.
    echo [ERROR] 挂载验证失败！
    echo.
    echo 可能的原因：
    echo    1. D:\audio 目录不存在或为空
    echo    2. Docker 权限问题
    echo.
    echo 请手动检查：
    echo    docker exec -it ai-worker bash
    echo    ls /audio/
    echo.
)

echo ========================================
echo 📊 容器信息
echo ========================================
docker ps | findstr ai-worker
echo.

pause
