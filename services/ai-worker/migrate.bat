@echo off
echo 正在执行数据库迁移...
echo.

set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=admin
set DB_PASSWORD=admin123
set DB_NAME=audio_platform

echo 连接到 PostgreSQL 数据库...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -f migrations\001_create_audio_separation_tasks.sql

echo.
if %ERRORLEVEL% EQU 0 (
    echo 迁移成功完成！
) else (
    echo 迁移失败，请检查错误信息。
)

pause
