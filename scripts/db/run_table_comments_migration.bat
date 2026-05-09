@echo off
chcp 65001 >nul
echo ============================================================
echo   数据库表注释迁移脚本 - 自动执行工具
echo ============================================================
echo.

set /p DB_HOST="请输入数据库主机地址 (默认: localhost): "
if "%DB_HOST%"=="" set DB_HOST=localhost

set /p DB_PORT="请输入数据库端口 (默认: 5433): "
if "%DB_PORT%"=="" set DB_PORT=5433

set /p DB_USER="请输入数据库用户名 (默认: postgres): "
if "%DB_USER%"=="" set DB_USER=postgres

set /p DB_PASS="请输入数据库密码: "

set /p DB_NAME="请输入数据库名称 (默认: audio_platform): "
if "%DB_NAME%"=="" set DB_NAME=audio_platform

echo.
echo 正在连接数据库并执行迁移脚本...
echo 主机: %DB_HOST%:%DB_PORT%
echo 数据库: %DB_NAME%
echo 用户: %DB_USER%
echo.

set PGPASSWORD=%DB_PASS%
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -f "d:\audio-ai-platform\scripts\db\migrations\097_add_complete_table_comments.sql"

echo.
if %ERRORLEVEL% EQU 0 (
    echo ✅ 迁移成功完成！所有表注释已添加。
    echo 请刷新您的数据库管理工具查看效果。
) else (
    echo ❌ 迁移失败！请检查上面的错误信息。
    echo 可能的原因：
    echo   1. 数据库连接信息不正确
    echo   2. 用户没有 COMMENT 权限
    echo   3. 数据库不存在
)

echo.
pause
