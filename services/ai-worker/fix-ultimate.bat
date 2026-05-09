@echo off
chcp 65001 >nul
echo.
echo ══════════════════════════════════════
echo   🔧 ULTIMATE FIX v7 (超级简单版)
echo ══════════════════════════════════════
echo.

cd /d "%~dp0"

:: Step 1: Check if container is using OLD image
echo [1/5] Checking container status...
docker ps | findstr ai-worker
echo.

echo Container start time:
docker inspect ai-worker --format='{{.State.StartedAt}}'
echo.

Image creation time:
docker inspect ai-worker --format='{{.Created}}'
echo.

:: Step 2: COMPLETE DOCKER CLEANUP
echo.
echo [2/5] Complete Docker cleanup...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
docker rmi ai-worker:v1 >nul 2>&1
echo [OK] Cleaned all old containers and images
echo.

:: Step 3: Build FRESH image (guaranteed new)
echo.
echo [3/5] Building fresh image (this will take time)...
echo Running: docker build --no-cache -t ai-worker:v1 .
echo.

call docker build --no-cache -t ai-worker:v1 .

if %errorlevel% neq 0 (
    echo.
    echo ❌ BUILD FAILED!
    pause
    exit /b 1
)

echo.
echo ✅ BUILD SUCCESSFUL!
echo.

:: Step 4: Start container
echo [4/5] Starting fresh container...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo ❌ START FAILED!
    pause
    exit /b 1
)

echo ✅ Container started!
echo.

:: Step 5: Wait and verify
echo [5/5] Waiting for service (15 seconds)...
timeout /t 15 /nobreak >nul
echo.

echo ══════════════════════════════════════
echo   🧪 TESTING WEBSOCKET CONNECTION
echo ══════════════════════════════════════
echo.

echo Test 1: Health check
curl -s http://localhost:8004/api/v1/health | findstr "ok"
echo.

echo Test 2: WebSocket endpoint (should return 400, NOT 404!)
for /f "tokens=*" %%a in ('curl -s -o nul -w "%%{http_code}" "http://localhost:8004/api/v1/audio/separate/ws?task_id=ultimate_test"') do set STATUS=%%a

echo HTTP Status Code: %STATUS%
echo.

if "%STATUS%"=="400" (
    echo ✅✅✅ SUCCESS! Endpoint found!
    echo.
    echo 🎉🎉🎉 FIX WORKED! 🎉🎉🎉
    echo.
    echo Now test in Apifox:
    echo ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001^&token=YOUR_TOKEN
) else if "%STATUS%"=="404" (
    echo ❌❌❌ STILL 404! 
    echo.
    echo Showing last 20 log lines:
    echo ──────────────────────────────
    docker logs ai-worker --tail 20 2>&1
    echo ──────────────────────────────
) else (
    echo Status: %STATUS%
)

echo.
echo ══════════════════════════════════════
echo   📊 FULL LOGS (last 30 lines)
echo ══════════════════════════════════════
echo.
docker logs ai-worker --tail 30 2>&1
echo.

pause
