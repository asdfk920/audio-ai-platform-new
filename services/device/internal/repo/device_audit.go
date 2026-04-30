package repo

import (
	"context"
	"database/sql"
	"strings"
)

// LogDeviceStatusQuery 写入状态查询审计（失败不影响主流程）。
func LogDeviceStatusQuery(ctx context.Context, db *sql.DB, userID int64, snNorm, clientIP, outcome, source string, errCode int) {
	if db == nil {
		return
	}
	sn := strings.TrimSpace(snNorm)
	if len(sn) > 64 {
		sn = sn[:64]
	}
	ip := strings.TrimSpace(clientIP)
	if len(ip) > 64 {
		ip = ip[:64]
	}
	out := strings.TrimSpace(outcome)
	if len(out) > 32 {
		out = out[:32]
	}
	src := strings.TrimSpace(source)
	if len(src) > 16 {
		src = src[:16]
	}
	_, _ = db.ExecContext(ctx, `
INSERT INTO device_status_query_audit (user_id, sn_norm, client_ip, outcome, source, error_code)
VALUES ($1,$2,$3,$4,$5,$6)`, userID, sn, ip, out, src, errCode)
}
