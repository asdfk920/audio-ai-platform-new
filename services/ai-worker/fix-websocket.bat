@echo off
chcp 65001 >nul
echo ========================================
echo   AI Worker WebSocket 修复脚本 v3
echo   - 强制重建镜像（无缓存）
=======================================
echo.

cd /d "%~dp0"

:: Step 1: Force rebuild Docker image (no cache)
echo [1/4] Force rebuilding Docker image (no cache)...
echo This may take 2-3 minutes...
docker build --no-cache -t ai-worker:v1 .

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Docker build failed!
    pause
    exit /b 1
)
echo [OK] Image rebuilt successfully!
echo.

:: Step 2: Stop & Remove old container
echo [2/4] Removing old container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
echo [OK] Old container removed
echo.

:: Step 3: Start new container
echo [3/4] Starting new container with mount...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo [ERROR] Failed to start container!
    pause
    exit /b 1
)
echo [OK] Container started!
echo.

:: Step 4: Wait and verify
echo [4/4] Waiting for service to initialize (10 seconds)...
timeout /t 10 /nobreak >nul

:: Test health endpoint
echo.
echo Testing service health...
curl -s http://localhost:8004/api/v1/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Health check passed!
) else (
    echo [WARN] Service may still be starting...
)

echo.
echo ========================================
echo   Testing WebSocket Route...
echo ========================================

:: Test WS route (should return 400, not 404!)
curl -s -w "\nHTTP Status: %%{http_code}\n" "http://localhost:8004/api/v1/audio/separate/ws?task_id=test" 2>&1 | findstr /i "HTTP Status\|WebSocket\|error"

echo.
echo Checking container logs for [WS-Middleware]...
docker logs ai-worker --tail 20 2>&1 | findstr "WS-Middleware"

echo.
echo ========================================
echo   Verifying Audio Mount...
echo ========================================
docker exec ai-worker ls /audio/ 2>nul
if %errorlevel% equ 0 (
    echo [OK] Audio directory mounted!
) else (
    echo [INFO] Could not verify. Check if D:\audio\test1.mp3 exists.
)

echo.
echo ========================================
echo ✅ Fix Complete!
echo ========================================
echo.
echo Service URL: http://localhost:8004
echo WebSocket: ws://localhost:8004/api/v1/audio/separate/ws
echo.
echo IMPORTANT:
echo   If you see HTTP Status: 400 (not 404), the fix worked!
echo   If you still see 404, please check container logs:
echo     docker logs ai-worker --tail 50
echo.
echo Next step:
echo   Test in Apifox with WebSocket connection!
echo.

pause
