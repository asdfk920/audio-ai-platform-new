@echo off
echo ============================================
echo   AI Worker WebSocket Fix - English Version
echo   No Chinese characters - No encoding issues
echo ============================================
echo.

cd /d "%~dp0"

echo [Step 1/5] Stopping and removing old container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
echo [OK] Container removed
echo.

echo [Step 2/5] Removing old Docker image...
docker rmi ai-worker:v1 >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Old image removed
) else (
    echo [INFO] No old image found or already removed
)
echo.

echo [Step 3/5] Building NEW image (NO CACHE)...
echo This will take 3-5 minutes, please wait...
echo.
call docker build --no-cache -t ai-worker:v1 .

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] BUILD FAILED!
    pause
    exit /b 1
)
echo.
echo [OK] Build successful!
echo.

echo [Step 4/5] Starting new container...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo [ERROR] Failed to start container!
    pause
    exit /b 1
)
echo [OK] Container started
echo.

echo [Step 5/5] Waiting for service to start (15 seconds)...
timeout /t 15 /nobreak >nul
echo.

echo ============================================
echo   TESTING WEBSOCKET ENDPOINT
echo ============================================
echo.

echo Test 1: Health check
curl -s http://localhost:8004/api/v1/health | findstr "ok"
echo.

echo Test 2: WebSocket endpoint (should return 400, NOT 404!)
for /f "tokens=*" %%a in ('curl -s -o nul -w "%%{http_code}" "http://localhost:8004/api/v1/audio/separate/ws?task_id=test_en"') do set STATUS=%%a

echo HTTP Status Code: %STATUS%
echo.

if "%STATUS%"=="400" (
    echo ============================================
    echo   SUCCESS! Endpoint found (HTTP 400)
    echo ============================================
    echo.
    echo The fix is working!
    echo Now test in Apifox with:
    echo ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001^&token=YOUR_TOKEN
) else if "%STATUS%"=="101" (
    echo ============================================
    echo   PERFECT! WebSocket upgrade works!
    echo ============================================
) else if "%STATUS%"=="404" (
    echo ============================================
    echo   STILL 404! Checking logs...
    echo ============================================
    docker logs ai-worker --tail 20 2>&1
) else (
    echo Status: %STATUS%
)

echo.
echo ============================================
echo   Last 30 log lines
echo ============================================
echo.
docker logs ai-worker --tail 30 2>&1
echo.

echo ============================================
echo   DONE!
echo ============================================
echo.
echo To watch live logs: docker logs ai-worker -f
echo.

pause
