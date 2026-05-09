@echo off
chcp 65001 >nul
echo.
echo ╔═════════════════════════════════════════════════════════╗
echo ║     🎯 FINAL FIX v8 - ROOT CAUSE SOLUTION              ║
echo ║     修复根本原因: routes.go 中注册 WS 路由             ║
echo ╚═════════════════════════════════════════════════════════╝
echo.

cd /d "%~dp0"

:: ============================================
:: Step 1: Show what we fixed
:: ============================================
echo [Step 1/5] ✅ Changes made:
echo.
echo   📝 File: internal\handler\routes.go
echo   ➕ Added WebSocket route to publicRoutes:
echo      Path: /api/v1/audio/separate/ws
echo      Handler: AudioSeparateWSHandler()
echo.
echo   📝 File: ai-worker.go  
echo   ➖ Removed non-working middleware code
echo   ✅ Simplified to use routes.go registration
echo.

:: ============================================
:: Step 2: Complete Docker cleanup
:: ============================================
echo [Step 2/5] Cleaning up Docker (removing old image)...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
docker rmi ai-worker:v1 >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ Old image removed
) else (
    echo   ℹ️  No old image found (OK)
)
echo.

:: ============================================
:: Step 3: Build new image (NO CACHE!)
:: ============================================
echo [Step 3/5] Building NEW image (NO CACHE)...
echo ⏳ This will take 3-5 minutes, please wait...
echo.

call docker build --no-cache -t ai-worker:v1 . 2>&1

if %errorlevel% neq 0 (
    echo.
    echo ❌ BUILD FAILED!
    echo Check the error messages above.
    pause
    exit /b 1
)

echo.
echo ✅✅✅ BUILD SUCCESSFUL! ✅✅✅
echo.

:: ============================================
:: Step 4: Start container
:: ============================================
echo [Step 4/5] Starting container...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo ❌ START FAILED!
    pause
    exit /b 1
)

echo ✅ Container started successfully!
echo.

:: ============================================
:: Step 5: Wait and test
:: ============================================
echo [Step 5/5] Waiting for service (15 seconds)...
timeout /t 15 /nobreak >nul
echo.

echo ╔═════════════════════════════════════════════════════════╗
echo ║            🧪 TESTING WEBSOCKET ROUTE                   ║
echo ╚═════════════════════════════════════════════════════════╝
echo.

echo Test 1: Health check
curl -s http://localhost:8004/api/v1/health | findstr "ok"
if %errorlevel% equ 0 (
    echo   ✅ Health check passed
) else (
    echo   ⚠️  Health check failed (service may still be starting)
)
echo.

echo Test 2: WebSocket endpoint (CRITICAL TEST)
echo ──────────────────────────────────────────────────────
for /f "tokens=*" %%a in ('curl -s -o nul -w "%%{http_code}" "http://localhost:8004/api/v1/audio/separate/ws?task_id=v8_final_test"') do set STATUS=%%a

echo   HTTP Status Code: %STATUS%
echo ──────────────────────────────────────────────────────
echo.

if "%STATUS%"=="101" (
    echo ╔═════════════════════════════════════════════════════════╗
    echo ║  🎉🎉🎉 PERFECT! WebSocket upgrade successful! 🎉🎉🎉       ║
    echo ╚═════════════════════════════════════════════════════════╝
    echo.
    echo ✅ The fix worked! Now test in Apifox:
    echo    ws://localhost:8004/api/v1/audio/separate/ws^
    echo        ?task_id=test_001^&token=YOUR_JWT_TOKEN
) else if "%STATUS%"=="400" (
    echo ╔═════════════════════════════════════════════════════════╗
    echo ║  ✅✅✅ SUCCESS! Endpoint found (HTTP 400 = normal) ✅✅✅    ║
    echo ╚═════════════════════════════════════════════════════════╝
    echo.
    echo ✅ The fix worked! 
    echo    HTTP 400 is expected for non-WebSocket clients
    echo    In Apifox, it will upgrade to WebSocket (101)
    echo.
    echo 🚀 Now test in Apifox:
    echo    ws://localhost:8004/api/v1/audio/separate/ws^
    echo        ?task_id=test_001^&token=YOUR_JWT_TOKEN
) else if "%STATUS%"=="404" (
    echo ╔═════════════════════════════════════════════════════════╗
    echo ║  ❌❌❌ STILL 404! Checking logs... ❌❌❌                  ║
    echo ╚═════════════════════════════════════════════════════════╝
    echo.
    echo Last 20 log lines:
    echo ──────────────────────────────────────────────────────
    docker logs ai-worker --tail 20 2>&1
    echo ──────────────────────────────────────────────────────
) else (
    echo Unexpected status: %STATUS%
    echo Check logs for details
)

echo.
echo ╔═════════════════════════════════════════════════════════╗
echo ║              📊 FULL LOGS (last 30 lines)                ║
echo ╚═════════════════════════════════════════════════════════╝
echo.
docker logs ai-worker --tail 30 2>&1
echo.

echo ╔═════════════════════════════════════════════════════════╗
echo ║                    🔍 KEY CHECKS                         ║
echo ╚═════════════════════════════════════════════════════════╝
echo.
echo Looking for [FIXED] marker in startup logs...
docker logs ai-worker 2>&1 | findstr /i "\[FIXED\]"
echo.
echo Looking for WebSocket connection logs...
docker logs ai-worker 2>&1 | findstr /i "\[WebSocket\]"
echo.

echo ╔═════════════════════════════════════════════════════════╗
echo ║                    ✅ DONE!                              ║
echo ╚═════════════════════════════════════════════════════════╝
echo.
echo 📋 Next steps:
echo.
echo If status is 400 or 101:
echo   1. Open Apifox
echo   2. Create new WebSocket request
echo   3. URL: ws://localhost:8004/api/v1/audio/separate/ws^
echo          ?task_id=test_001^&token=PASTE_YOUR_TOKEN_HERE
echo   4. Click Connect
echo   5. Send message:
echo      {"type":"separate","data":{"content_id":2}}
echo.
echo To watch live logs:
echo   docker logs ai-worker -f
echo.

pause
