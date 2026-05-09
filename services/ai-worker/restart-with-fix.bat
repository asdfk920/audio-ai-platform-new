@echo off
chcp 65001 >nul
echo ========================================
echo   重启 AI Worker 容器（修复 WebSocket）
echo ========================================
echo.

:: Step 1: Stop & Remove old container
echo [1/3] Stopping old container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
echo [OK] Container removed
echo.

:: Step 2: Start new container with mount
echo [2/3] Starting new container (with WebSocket fix)...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo [ERROR] Failed to start container!
    pause
    exit /b 1
)
echo [OK] Container started
echo.

:: Step 3: Wait and verify
echo [3/3] Waiting for service to initialize...
timeout /t 8 /nobreak >nul

:: Check if service is healthy
echo.
echo Checking service health...
curl -s http://localhost:8004/api/v1/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Service is running!
) else (
    echo [WARN] Service may still be starting...
)

echo.
echo ========================================
echo   Checking WebSocket route...
echo ========================================

:: Test WebSocket route (should not return 404 anymore)
curl -s -o nul -w "%%{http_code}" "http://localhost:8004/api/v1/audio/separate/ws?task_id=test" >temp_status.txt 2>&1
set /p status=<temp_status.txt
del temp_status.txt 2>nul

if "%status%"=="101" (
    echo [SUCCESS] WebSocket upgrade works! (HTTP 101 Switching Protocols)
) else if "%status%"=="400" (
    echo [INFO] WebSocket endpoint found (returned 400, which is expected for non-WS client)
) else if "%status%"=="404" (
    echo [ERROR] Still returning 404! Route not registered properly.
) else (
    echo [INFO] Status code: %status%
)

echo.
echo ========================================
echo   Verifying audio mount...
echo ========================================
docker exec ai-worker ls /audio/ 2>nul
if %errorlevel% equ 0 (
    echo [OK] Audio directory mounted successfully!
) else (
    echo [WARN] Could not verify mount. Check if D:\audio\test1.mp3 exists.
)

echo.
echo ========================================
echo ✅ Container restarted with fixes!
echo ========================================
echo.
echo Service URL: http://localhost:8004
echo WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
echo Next step:
echo   Test in Apifox with WebSocket connection!
echo.

pause
