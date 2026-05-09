@echo off
chcp 65001 >nul
echo ═══════════════════════════════════════════
echo   AI Worker WebSocket 诊断修复脚本 v4
echo   - 强制重建（无缓存）
echo   - 完整日志诊断
echo ═══════════════════════════════════════════
echo.

cd /d "%~dp0"

:: ============================================
:: Step 1: Force rebuild Docker image (NO CACHE)
:: ============================================
echo.
echo [Step 1/5] Force rebuilding Docker image (NO CACHE)...
echo ⚠️  This will take 2-4 minutes...
echo.

docker build --no-cache -t ai-worker:v1 . 2>&1

if %errorlevel% neq 0 (
    echo.
    echo [❌ ERROR] Docker build failed!
    echo Please check the error messages above.
    pause
    exit /b 1
)

echo.
echo [✅ SUCCESS] Image rebuilt successfully!
echo.

:: ============================================
:: Step 2: Stop & Remove old container
:: ============================================
echo [Step 2/5] Removing old container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
echo [✅ OK] Old container removed
echo.

:: ============================================
:: Step 3: Start new container
:: ============================================
echo [Step 3/5] Starting new container with mount...
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1

if %errorlevel% neq 0 (
    echo [❌ ERROR] Failed to start container!
    pause
    exit /b 1
)

echo [✅ OK] Container started!
echo.

:: ============================================
:: Step 4: Wait and check diagnostic logs
:: ============================================
echo [Step 4/5] Waiting for service to initialize (12 seconds)...
timeout /t 12 /nobreak >nul
echo.

echo ═══════════════════════════════════════════
echo   🔍 DIAGNOSTIC LOGS CHECK
echo ═══════════════════════════════════════════
echo.

echo Checking for [DIAGNOSTIC] logs in container startup...
echo ────────────────────────────────────────
docker logs ai-worker 2>&1 | findstr /i "DIAGNOSTIC"
echo ────────────────────────────────────────
echo.

:: Check if diagnostic logs exist
docker logs ai-worker 2>&1 | findstr /i "[DIAGNOSTIC]" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ GOOD! Diagnostic code is running in container!
) else (
    echo ❌ WARNING! No [DIAGNOSTIC] logs found!
    echo    This means the OLD code is still running!
    echo    Possible causes:
    echo    - Docker used cached layers despite --no-cache
    echo    - Build cache not cleared properly
)

echo.
echo ═══════════════════════════════════════════
echo   🧪 TESTING WEBSOCKET ROUTE
echo ═══════════════════════════════════════════
echo.

:: ============================================
:: Step 5: Test WebSocket route
:: ============================================
echo [Step 5/5] Testing WebSocket endpoint...
echo Request: GET http://localhost:8004/api/v1/audio/separate/ws?task_id=diag_test
echo.
echo ────────────────────────────────────────
curl -s -w "\nHTTP Status Code: %%{http_code}\n" "http://localhost:8004/api/v1/audio/separate/ws?task_id=diag_test" 2>&1
echo ────────────────────────────────────────
echo.

echo.
echo ═══════════════════════════════════════════
echo   📊 REAL-TIME LOGS (Last 30 lines)
echo ═══════════════════════════════════════════
echo.
docker logs ai-worker --tail 30 2>&1
echo.

echo ═══════════════════════════════════════════
echo   🔎 SEARCHING FOR KEY LOGS
echo ═══════════════════════════════════════════
echo.

echo [Check 1] Looking for [DIAG-MW] middleware logs...
docker logs ai-worker 2>&1 | findstr /i "DIAG-MW"
if %errorlevel% neq 0 (
    echo    ❌ NOT FOUND - Middleware is NOT intercepting requests!
)
echo.

echo [Check 2] Looking for [WS-Middleware] interception...
docker logs ai-worker 2>&1 | findstr /i "WS-Middleware"
if %errorlevel% neq 0 (
    echo    ❌ NOT FOUND - WS handler was never called!
)
echo.

echo [Check 3] Looking for [WebSocket] connection logs...
docker logs ai-worker 2>&1 | findstr /i "WebSocket"
if %errorlevel% neq 0 (
    echo    ❌ NOT FOUND - No WS connection attempts logged!
)
echo.

echo ═══════════════════════════════════════════
echo   ✅ DIAGNOSTIC COMPLETE!
echo ═══════════════════════════════════════════
echo.
echo 📋 INTERPRETATION GUIDE:
echo.
echo If you see [DIAG-MW] but NO [WS-Middleware]:
echo    → Middleware is working, but path check failed
echo    → Check URL path exact match
echo.
echo If you see NEITHER [DIAG-MW] nor [WS-Middleware]:
echo    → server.Use() middleware is NOT being called
echo    → go-zero routing happens BEFORE middleware
echo    → Need different approach (see below)
echo.
echo If you see [WS-Middleware]:
echo    → ✅ FIX IS WORKING! Test in Apifox now!
echo.
echo ═══════════════════════════════════════════
echo   📁 NEXT STEPS
echo ═══════════════════════════════════════════
echo.
echo 1. Review the logs above carefully
echo 2. Look for the key markers: [DIAG-MW], [WS-Middleware], [WebSocket]
echo 3. Based on findings, proceed to fix-websocket-final.bat
echo.
echo To watch live logs: docker logs ai-worker -f
echo To test in Apifox: ws://localhost:8004/api/v1/audio/separate/ws
echo.

pause
