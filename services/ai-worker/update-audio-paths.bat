@echo off
echo ========================================
echo   数据库音频路径更新工具
echo ========================================

cd /d %~dp0

echo [INFO] 正在更新数据库中的 audio_url...
echo.

:: 使用 psql 更新数据库
echo 请选择部署环境:
echo.
echo 1. Docker 部署（容器内路径: /app/audio-files/）
echo 2. Windows 本地开发（相对路径: ./audio-files/）
echo.
set /p choice=请输入选项 (1 或 2):

if "%choice%"=="1" (
    echo.
    echo [INFO] 使用 Docker 路径格式...
    echo UPDATE content SET audio_url = REPLACE(audio_url, '/audio/', '/app/audio-files/') WHERE audio_url LIKE '%%/audio/%%'; | psql -h localhost -U admin -d audio_platform
) else (
    echo.
    echo [INFO] 使用 Windows 本地路径格式...
    echo UPDATE content SET audio_url = REPLACE(audio_url, '/audio/', './audio-files/') WHERE audio_url LIKE '%%/audio/%%'; | psql -h localhost -U admin -d audio_platform
)

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo ✅ 音频路径更新成功！
    echo ========================================
    echo.
    echo 现在可以:
    echo 1. 将音频文件放入 audio-files 目录
    echo 2. 重启 ai-worker 服务
    echo 3. 测试 WebSocket 分离功能
    echo.
) else (
    echo.
    echo [ERROR] 更新失败！请检查数据库连接配置
)

pause
