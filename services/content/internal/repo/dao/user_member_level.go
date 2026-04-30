package dao

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// EffectiveVipLevel 读取 user_member 当前有效会员档位（0=非会员/过期）。
// 与列表接口可见性规则一致：status=1 且（永久 或 未过期）。
func EffectiveVipLevel(ctx context.Context, db *gorm.DB, userID int64) int32 {
	if db == nil || userID <= 0 {
		return 0
	}
	var lvl sql.NullInt32
	q := `
SELECT um.level
FROM user_member um
WHERE um.user_id = ?
  AND um.status = 1
  AND (
    COALESCE(um.is_permanent, 0) = 1
    OR (COALESCE(um.expire_at, um.expired_at) IS NOT NULL AND COALESCE(um.expire_at, um.expired_at) > NOW())
  )
LIMIT 1`
	err := db.WithContext(ctx).Raw(q, userID).Scan(&lvl).Error
	if err != nil || !lvl.Valid {
		return 0
	}
	if lvl.Int32 < 0 {
		return 0
	}
	return lvl.Int32
}
