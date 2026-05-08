@echo off
echo ========================================
echo   AI Worker 启动脚本（使用 Demucs 虚拟环境）
echo ========================================

cd /d %~dp0

:: 设置虚拟环境路径
set "VENV_PATH=%~dp0demucs_env"
set "PATH=%VENV_PATH%\Scripts;%PATH%"
set "PYTHON=%VENV_PATH%\Scripts\python.exe"

echo [INFO] 使用 Python: %PYTHON%
echo [INFO] Demucs 位置: %VENV_PATH%\Scripts\demucs.exe
echo.

:: 验证 demucs 可用
%VENV_PATH%\Scripts\demucs.exe -h >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Demucs 未找到！请先运行安装脚本
    pause
    exit /b 1
)

echo [INFO] Demucs 就绪，正在启动 AI Worker...
echo.

:: 启动服务
%PYTHON% ai-worker.exe -f etc/ai-worker.yaml

pause
