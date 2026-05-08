@echo off
chcp 65001 >nul
echo ========================================
echo   AI Worker Docker 挂载脚本 (3步)
echo   -v D:\audio:/audio
echo ========================================
echo.

:: Step 1: Stop & Remove old container
echo [Step 1/3] Stopping and removing old container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Old container removed
) else (
    echo [INFO] No existing container found (this is OK)
)
echo.

:: Step 2: Start new container with mount
echo [Step 2/3] Starting new container with D:\audio:/audio mount...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Container failed to start!
    echo.
    echo Possible reasons:
    echo   1. Docker Desktop is not running
    echo   2. Port 8004 is already in use
    echo   3. Image ai-worker:v1 does not exist
    echo.
    pause
    exit /b 1
)

echo.
echo [OK] Container started successfully!
echo.

:: Step 3: Verify mount
echo [Step 3/3] Verifying mount (waiting for container to initialize)...
timeout /t 5 /nobreak >nul

echo.
echo Checking /audio directory inside container:
echo ----------------------------------------
docker exec ai-worker ls -lh /audio/

if %errorlevel% equ 0 (
    echo ----------------------------------------
    echo.
    echo ============================================
    echo   SUCCESS! Mount verified!
    echo ============================================
    echo.
    echo Service URL: http://localhost:8004
    echo WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
    echo.
    echo Next steps:
    echo   1. Update database: UPDATE content SET audio_url = '/audio/test1.mp3' WHERE id = 2;
    echo   2. Test WebSocket connection in Apifox
) else (
    echo ----------------------------------------
    echo.
    echo [WARNING] Mount verification failed or empty
    echo Please check if D:\audio\test1.mp3 exists on your host machine
)

echo.
pause
