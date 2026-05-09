@echo off
chcp 65001 >nul
echo.
echo ══════════════════════════════════════
echo   AI Worker WebSocket 修复 v6 (SIMPLE)
echo ══════════════════════════════════════
echo.

cd /d "%~dp0"

:: Step 1: Clean up old files
echo [1/4] Cleaning up...
if exist "ai-worker-final-fix.go" del "ai-worker-final-fix.go"
if exist "ai-worker.go.bak" del "ai-worker.go.bak"
echo [OK] Cleaned
echo.

:: Step 2: Force rebuild (NO CACHE!)
echo [2/4] Rebuilding Docker image (NO CACHE)...
echo This takes 3-5 minutes...
docker build --no-cache -t ai-worker:v1 . 2>&1 | findstr /i "error\|ERROR\|DONE\|naming"
if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Build failed! Check output above.
    pause
    exit /b 1
)
echo [OK] Build complete!
echo.

:: Step 3: Restart container
echo [3/4] Restarting container...
docker stop ai-worker >nul 2>&1
docker rm ai-worker >nul 2>&1
docker run -d --name ai-worker --gpus all -p 8004:8004 -v "D:\audio:/audio" ai-worker:v1
echo [OK] Container started
echo.

:: Step 4: Wait and test
echo [4/4] Waiting 15 seconds for service to start...
timeout /t 15 /nobreak >nul
echo.
echo Testing WebSocket endpoint...
echo.
curl -s -w "HTTP Status: %%{http_code}\n" "http://localhost:8004/api/v1/audio/separate/ws?task_id=simple_test" 2>&1
echo.
echo.
echo Checking for WS middleware in logs...
docker logs ai-worker 2>&1 | findstr /i "WS-Middleware\|DIAG-MW\|WebSocket"
echo.
echo.
echo ══════════════════════════════════════
echo   ✅ DONE!
echo ══════════════════════════════════════
echo.
echo If you see HTTP Status: 400 or 101 = SUCCESS!
echo If HTTP Status: 404 = Check logs above
echo.
echo To watch live: docker logs ai-worker -f
echo.

pause
