#!/usr/bin/env bash
# 按顺序执行用户库正向迁移（与 docker-entrypoint-initdb.d 顺序一致，含 008 用户中心扩展）。
# 若报「对模式 public 权限不够」，请超级用户先执行:
#   scripts/db/grant_public_schema_to_admin.sql
# 若手工建过 users 且报「必须是表 users 的属主」，请超级用户先执行:
#   scripts/db/grant_users_table_to_admin.sql 或 scripts/db/pre_migrate_owner_to_admin.sql
# 用法: DATABASE_URL=postgresql://admin:admin123@localhost:5432/audio_platform ./scripts/db/apply-all-migrations.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DIR="$ROOT/scripts/db/migrations"
URL="${DATABASE_URL:-postgresql://admin:admin123@localhost:5432/audio_platform}"
# 若本机未装 psql：DOCKER_POSTGRES_CONTAINER=audio-platform-postgres ./scripts/db/apply-all-migrations.sh
for f in \
  001_init.sql \
  002_user_register_oauth.sql \
  002_register_security.sql \
  003_users_extended_fields.sql \
  004_users_mobile_unique_playback_device_nullable.sql \
  005_user_realname_auth.sql \
  006_user_realname_idcard_photos.sql \
  007_user_cancellation.sql \
  008_user_center_schema.sql \
  009_user_schema_optimize.sql \
  010_go_admin_bootstrap_admin.sql \
  011_go_admin_sys_config.sql \
  012_go_admin_sys_user_sys_role_views.sql \
  013_users_real_name.sql \
  014_audio_platform_roles.sql \
  015_sys_role_display_name.sql \
  016_fix_roles_description_utf8.sql \
  017_go_admin_sys_menu.sql \
  018_fix_sys_menu_titles_utf8.sql \
  019_fix_pg_comments_utf8.sql \
  020_users_avatar.sql \
  021_go_admin_upload_storage_config.sql \
  022_membership_tables.sql \
  023_roles_add_member_mgmt_module.sql \
  024_go_admin_sys_menu_add_member.sql \
  025_users_profile_extended.sql \
  026_device_module.sql \
  027_fix_membership_comments_utf8.sql \
  028_device_comments_utf8.sql \
  029_add_missing_table_comments_utf8.sql \
  030_go_admin_sys_menu_add_realname_review.sql \
  031_fix_platform_realname_menu_utf8.sql \
  032_go_admin_sys_menu_fix_member_id_conflict.sql \
  033_user_device_bind_user_service.sql \
  034_drop_legacy_devices_tables.sql \
  035_user_device_bind_status_zero.sql \
  036_member_pay_orders.sql \
  037_order_master_spec_comments.sql \
  038_member_benefit_admin_extensions.sql \
  039_member_level_admin_extensions.sql \
  052_streaming_tables.sql \
  053_streaming_tables_comments.sql \
  054_stream_channels_extend.sql \
  055_user_device_bind_operator.sql \
  056_user_device_bind_log.sql \
  065_device_status_alert.sql
do
  echo "==> $f"
  export PGCLIENTENCODING=UTF8
  if [ -n "${DOCKER_POSTGRES_CONTAINER:-}" ]; then
    cat "$DIR/$f" | docker exec -i "$DOCKER_POSTGRES_CONTAINER" psql -U admin -d audio_platform -v ON_ERROR_STOP=1 || {
      echo ""
      echo "迁移失败: $f"
      exit 1
    }
  else
  psql "$URL" -v ON_ERROR_STOP=1 -f "$DIR/$f" || {
    echo ""
    echo "迁移失败: $f"
    echo "若提示「对模式 public 权限不够」，请先:"
    echo "  psql -U postgres -d audio_platform -f scripts/db/grant_public_schema_to_admin.sql"
    echo "若提示「必须是表 ... 的属主」，请以 postgres 超级用户任选其一:"
    echo "  A) psql -U postgres -d audio_platform -f scripts/db/pre_migrate_owner_to_admin.sql"
    echo "  B) psql -U postgres -d postgres -f scripts/db/grant_admin_superuser.sql"
    exit 1
  }
  fi
done
echo "apply-all-migrations: OK"
