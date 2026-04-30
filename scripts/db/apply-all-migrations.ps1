# 按顺序执行用户库正向迁移（方案 A：001–008 全量）。
#
# 若报「对模式 public 权限不够」，请超级用户先执行：
#   scripts/db/grant_public_schema_to_admin.sql
# 若手工建过 users 且报「必须是表 users 的属主」，请超级用户执行：
#   scripts/db/grant_users_table_to_admin.sql
# 或将整个 public 交给 admin：scripts/db/pre_migrate_owner_to_admin.sql
#
# 默认连接：Windows 本地开发默认使用 docker-compose 的 Postgres（主机 5433 → 容器 5432）。
# - 若你使用本机 PostgreSQL（通常 5432），请设 $env:POSTGRES_PORT=5432 或直接设 $env:DATABASE_URL 覆盖。
# - 可覆盖: $env:DATABASE_URL = "postgresql://..."
#
# 若报错「必须是表 ... 的属主」，任选其一（均需 postgres 超级用户）：
#   A) 移交属主（推荐）：psql -U postgres -d audio_platform -f scripts/db/pre_migrate_owner_to_admin.sql
#   B) 给 admin 超管（慎用）：psql -U postgres -d postgres -f scripts/db/grant_admin_superuser.sql
# 再重新运行本脚本。
#
# 若希望整段迁移都用超级用户连接（不推荐生产），可设:
#   $env:DATABASE_URL = "postgresql://postgres:密码@localhost:5432/audio_platform"

$ErrorActionPreference = "Stop"
# Windows 下管道/文件默认编码易导致 SQL 中文注释入库成乱码；统一按 UTF-8 读迁移文件
$PSDefaultParameterValues['Get-Content:Encoding'] = 'utf8'
$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$dir = Join-Path $root "scripts\db\migrations"
$port = if ($env:POSTGRES_PORT) { $env:POSTGRES_PORT } else { "5433" }
$url = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgresql://admin:admin123@localhost:$port/audio_platform" }
# 若本机未装 psql，可设: $env:DOCKER_POSTGRES_CONTAINER = "audio-platform-postgres"
$dockerContainer = $env:DOCKER_POSTGRES_CONTAINER
$migrations = @(
    "001_init.sql",
    "002_user_register_oauth.sql",
    "002_register_security.sql",
    "003_users_extended_fields.sql",
    "004_users_mobile_unique_playback_device_nullable.sql",
    "005_user_realname_auth.sql",
    "006_user_realname_idcard_photos.sql",
    "007_user_cancellation.sql",
    "008_user_center_schema.sql",
    "009_user_schema_optimize.sql",
    "010_go_admin_bootstrap_admin.sql",
    "011_go_admin_sys_config.sql",
    "012_go_admin_sys_user_sys_role_views.sql",
    "013_users_real_name.sql",
    "014_audio_platform_roles.sql",
    "015_sys_role_display_name.sql",
    "016_fix_roles_description_utf8.sql",
    "017_go_admin_sys_menu.sql",
    "018_fix_sys_menu_titles_utf8.sql",
    "019_fix_pg_comments_utf8.sql",
    "020_users_avatar.sql",
    "021_go_admin_upload_storage_config.sql",
    "022_membership_tables.sql",
    "023_roles_add_member_mgmt_module.sql",
    "024_go_admin_sys_menu_add_member.sql",
    "025_users_profile_extended.sql",
    "026_device_module.sql",
    "027_fix_membership_comments_utf8.sql",
    "028_device_comments_utf8.sql",
    "029_add_missing_table_comments_utf8.sql",
    "030_go_admin_sys_menu_add_realname_review.sql",
    "031_fix_platform_realname_menu_utf8.sql",
    "032_go_admin_sys_menu_fix_member_id_conflict.sql",
    "033_user_device_bind_user_service.sql",
    "034_drop_legacy_devices_tables.sql",
    "035_user_device_bind_status_zero.sql",
    "036_member_pay_orders.sql",
    "037_order_master_spec_comments.sql",
    "038_member_benefit_admin_extensions.sql",
    "039_member_level_admin_extensions.sql",
    "040_device_admin_fields.sql",
    "041_go_admin_sys_menu_add_platform_device.sql",
    "042_device_register_audit.sql",
    "043_device_status_query_audit.sql",
    "044_device_status_snapshot.sql",
    "045_content_service_catalog.sql",
    "046_content_is_deleted.sql",
    "047_device_instruction_history.sql",
    "048_device_instruction_execution.sql",
    "049_ota_firmware_admin_fields.sql",
    "050_ota_firmware_sha256.sql",
    "051_ota_firmware_soft_delete.sql",
    "052_streaming_tables.sql",
    "053_streaming_tables_comments.sql"
    ,"054_stream_channels_extend.sql"
    ,"055_user_device_bind_operator.sql"
    ,"056_user_device_bind_log.sql"
    ,"065_device_status_alert.sql"
    ,"073_device_admin_display_columns.sql"
    ,"074_device_activate_nonce.sql"
    ,"075_device_event_log_operator.sql"
    ,"076_device_shadow_normalized.sql"
    ,"077_device_status_logs.sql"
    ,"078_device_status_logs_report_type.sql"
    ,"079_device_admin_info_extend.sql"
    ,"080_go_admin_sys_job.sql"
    ,"083_platform_org_dept_seed.sql"
    ,"084_fix_sys_dept_names_encoding.sql"
    ,"085_content_audio_validity.sql"
)
foreach ($f in $migrations) {
    $path = Join-Path $dir $f
    Write-Host "==> $f"
    # -f 配合 Get-Content UTF8 在部分环境仍可能异常，显式设置客户端编码
    $env:PGCLIENTENCODING = 'UTF8'
    if ($dockerContainer) {
        Get-Content -LiteralPath $path -Raw -Encoding utf8 | & docker exec -i $dockerContainer psql -U admin -d audio_platform -v ON_ERROR_STOP=1
    } else {
        Get-Content -LiteralPath $path -Raw -Encoding utf8 | & psql $url -v ON_ERROR_STOP=1
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "迁移失败: $f (psql exit $LASTEXITCODE)" -ForegroundColor Red
        Write-Host "若提示「对模式 public 权限不够」，请先:" -ForegroundColor Yellow
        Write-Host "  psql -U postgres -d audio_platform -f scripts/db/grant_public_schema_to_admin.sql" -ForegroundColor Yellow
        Write-Host "若提示「必须是表 ... 的属主」，请以 postgres 超级用户执行:" -ForegroundColor Yellow
        Write-Host "  A) psql -U postgres -d audio_platform -f scripts/db/pre_migrate_owner_to_admin.sql" -ForegroundColor Yellow
        Write-Host "  B) psql -U postgres -d postgres -f scripts/db/grant_admin_superuser.sql" -ForegroundColor Yellow
        exit $LASTEXITCODE
    }
}
Write-Host "apply-all-migrations: OK" -ForegroundColor Green
