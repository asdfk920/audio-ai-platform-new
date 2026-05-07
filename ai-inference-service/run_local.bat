@echo off
REM BSRoformer SCNet 本地运行脚本（无需 Docker）

echo ╔════════════════════════════════════════════╗
echo ║  BSRoformer SCNet 音轨分离服务 - 本地运行  ║
echo ╚════════════════════════════════════════════╝
echo.

REM 检查 Python 是否安装
where python >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 未检测到 Python，请先安装 Python 3.10+
    echo 下载地址：https://www.python.org/downloads/
    pause
    exit /b 1
)

echo [1/4] 检查 Python 版本...
python --version
echo.

echo [2/4] 创建虚拟环境...
if not exist "venv" (
    python -m venv venv
    echo [✓] 虚拟环境已创建
) else (
    echo [✓] 虚拟环境已存在
)

echo.
echo [3/4] 激活虚拟环境并安装依赖...
call venv\Scripts\activate.bat
pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 依赖安装失败
    pause
    exit /b 1
)
echo [✓] 依赖安装完成

echo.
echo [4/4] 创建必要目录...
if not exist "models" mkdir models
if not exist "data\input" mkdir data\input
if not exist "data\output" mkdir data\output
echo [✓] 目录创建完成

echo.
echo ╔════════════════════════════════════════════╗
echo ║  准备就绪，启动服务...                     ║
echo ╚════════════════════════════════════════════╝
echo.
echo 服务信息:
echo   - API 地址：http://localhost:8004
echo   - 健康检查：http://localhost:8004/health
echo   - 文档地址：http://localhost:8004/docs
echo.
echo 按 Ctrl+C 停止服务
echo.

REM 启动服务
python server.py
