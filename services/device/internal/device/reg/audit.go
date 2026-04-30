package reg

import (
	"context"
	"database/sql"
	"strings"
)

// LogRegisterAudit 写入注册审计（失败不影响主流程）。
func LogRegisterAudit(ctx context.Context, db *sql.DB, snNorm, productKey, clientIP, outcome string, errCode int) {
	if db == nil {
		return
	}
	sn := strings.TrimSpace(snNorm)
	if len(sn) > 64 {
		sn = sn[:64]
	}
	pkSuf := MaskProductKey(productKey)
	if len(pkSuf) > 24 {
		pkSuf = pkSuf[:24]
	}
	ip := strings.TrimSpace(clientIP)
	if len(ip) > 64 {
		ip = ip[:64]
	}
	out := strings.TrimSpace(outcome)
	if len(out) > 32 {
		out = out[:32]
	}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO device_register_audit (sn_norm, product_key_suffix, client_ip, outcome, error_code)
		VALUES ($1, $2, $3, $4, $5)
	`, sn, pkSuf, ip, out, errCode)
}
