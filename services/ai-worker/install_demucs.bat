@echo off
echo ========================================
echo   Demucs 音频分离模型 安装脚本
echo ========================================
echo.

:: 检查 Python
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] 未找到 Python，请先安装 Python 3.8+
    pause
    exit /b 1
)

echo [INFO] 检测到 Python 环境

:: 升级 pip
echo [1/3] 正在升级 pip...
python -m pip install --upgrade pip -i https://pypi.tuna.tsinghua.edu.cn/simple

:: 安装 demucs
echo.
echo [2/3] 正在安装 Demucs（音频分离模型）...
echo 这可能需要几分钟时间，请耐心等待...
echo.
pip install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple

if %errorlevel% neq 0 (
    echo [ERROR] Demucs 安装失败！
    pause
    exit /b 1
)

:: 验证安装
echo.
echo [3/3] 验证安装...
python -c "import demucs; print(f'Demucs 版本: {demucs.__version__}')"

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo ✅ Demucs 安装成功！
    echo ========================================
    echo.
    echo 可用命令:
    echo   demucs -h          # 查看帮助
    echo   demucs --version   # 查看版本
    echo.
    echo 使用示例:
    echo   demucs input.mp3   # 分离音频文件
    echo.
) else (
    echo [WARN] Demucs 模块导入测试失败，但可能已安装
)

pause
