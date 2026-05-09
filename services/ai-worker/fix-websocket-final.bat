@echo off
chcp 65001 >nul
echo ════════════════════════════════════════════════════════
echo   AI Worker WebSocket 最终修复脚本 v5.1
echo   - 使用已修复的 ai-worker-final-fix.go
echo   - 强制重建（无缓存）
echo ════════════════════════════════════════════════════════
echo.

cd /d "%~dp0"

:: ============================================
:: Step 1: Replace with fixed final version
:: ============================================
echo.
echo [Step 1/6] Replacing ai-worker.go with fixed version...
echo.

if not exist "ai-worker-final-fix.go" (
    echo ❌ ERROR: ai-worker-final-fix.go not found!
    pause
    exit /b 1
)

if exist "ai-worker.go.bak" del "ai-worker.go.bak"
copy ai-worker.go ai-worker.go.bak >nul 2>&1
copy /y ai-worker-final-fix.go ai-worker.go >nul 2>&1

if %errorlevel% neq 0 (
    echo ❌ ERROR: Failed to replace!
    pause
    exit /b 1
)

echo ✅ OK: File replaced successfully
echo    Backup: ai-worker.go.bak
echo.

:: ============================================
:: Step 2: Verify Go syntax before building
:: ============================================
echo [Step 2/6] Checking Go syntax...
go vet ./... 2>&1

if %errorlevel% neq 0 (
    echo.
    echo ❌ ERROR: Go syntax errors detected!
    echo Restoring original...
    copy /y ai-worker.go.bak ai-worker.go >nul 2>&1
    pause
    exit /b 1
)

echo ✅ OK: No syntax errors
echo.

:: ============================================
:: Step 3: Force rebuild Docker image
:: ============================================
echo [Step 3/6] Force rebuilding Docker image (NO CACHE)...
echo ⚠️  This will take 3-5 minutes...
echo.

docker build --no-cache -t ai-worker:v1 . 2>&1

if %errorlevel% neq 0 (
    echo.
    echo ❌ ERROR: Docker build failed!
    copy /y ai-worker.go.bak ai-worker.go >nul 2>&1
    pause
    exit /b 1
)

echo.
echo ✅ SUCCESS: Image rebuilt!
echo.

:: ============================================
:: Step 4: Restart container
:: ============================================
echo [Step 4/6] Restarting container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo ❌ ERROR: Failed to start container!
    pause
    exit /b 1
)

echo ✅ OK: Container started
echo.

:: ============================================
:: Step 5: Wait and check logs
:: ============================================
echo [Step 5/6] Waiting for initialization (15 seconds)...
timeout /t 15 /nobreak >nul
echo.

echo ════════════════════════════════════════════════════════
echo   🔍 CHECKING STARTUP LOGS
echo ════════════════════════════════════════════════════════
echo.
docker logs ai-worker 2>&1 | findstr /i "FINAL-FIX\|WS-Interceptor\|ERROR"
echo.

:: ============================================
:: Step 6: Test WebSocket
:: ============================================
echo ════════════════════════════════════════════════════════
echo   🧪 TESTING WEBSOCKET ENDPOINT
echo ════════════════════════════════════════════════════════
echo.
echo [Step 6/6] Testing...
curl -s -w "\n\nHTTP Status: %%{http_code}\n" "http://localhost:8004/api/v1/audio/separate/ws?task_id=test_v51" 2>&1
echo.

echo.
echo ════════════════════════════════════════════════════════
echo   📊 LAST 30 LOG LINES
echo ════════════════════════════════════════════════════════
echo.
docker logs ai-worker --tail 30 2>&1
echo.

echo ════════════════════════════════════════════════════════
echo   ✅ FIX COMPLETE!
echo ════════════════════════════════════════════════════════
echo.
echo Next:
echo   1. Check if HTTP Status is 400 or 101 (not 404)
echo   2. If success, test in Apifox with ws://
echo   3. Watch logs: docker logs ai-worker -f
echo.

pause
