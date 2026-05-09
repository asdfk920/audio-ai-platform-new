@echo off
chcp 65001 >nul
echo ============================================================
echo   AI-Worker 服务重启脚本 - 应用最新代码更改
echo ============================================================
echo.

cd /d "d:\audio-ai-platform\services\ai-worker"

echo [1/4] 正在查找并停止旧的 ai-worker 进程...
tasklist | findstr /i "ai-worker" >nul 2>&1
if %errorlevel%==0 (
    echo     发现运行中的进程，正在停止...
    taskkill /F /IM ai-worker.exe >nul 2>&1
    timeout /t 2 /nobreak >nul
    echo     ✓ 进程已停止
) else (
    echo     ℹ 未发现运行中的进程
)

echo.
echo [2/4] 正在重新编译代码...
go build -o ai-worker.exe . 2>&1
if %errorlevel% neq 0 (
    echo.
    echo     ✗ 编译失败！请检查上面的错误信息
    pause
    exit /b 1
)
echo     ✓ 编译成功

echo.
echo [3/4] 正在启动新服务...
start "" cmd /k "ai-worker.exe"
timeout /t 3 /nobreak >nul
echo     ✓ 服务已启动

echo.
echo [4/4] 验证服务是否正常运行...
timeout /t 2 /nobreak >nul
curl -s http://localhost:8004/api/v1/health >nul 2>&1
if %errorlevel%==0 (
    echo     ✓ 服务健康检查通过！
) else (
    echo     ⚠ 服务可能还在启动中，请稍等几秒后再试
)

echo.
echo ============================================================
echo   ✅ 重启完成！
echo.
echo   现在可以在 Postman 中：
echo   1. 断开 WebSocket 连接
echo   2. 重新连接
echo   3. 发送 {"type":"separate","data":{"content_id":3}}
echo   4. 发送 {"type":"cancel"} 测试取消功能
echo ============================================================
echo.
pause
