package logic

import (
	"database/sql"
)

func statusToInt(s string) int16 {
	switch s {
	case "pending":
		return 0
	case "active":
		return 1
	case "rejected":
		return 2
	case "revoked":
		return 3
	case "expired":
		return 4
	case "quit":
		return 5
	default:
		return 0
	}
}

func formatNullTime(t sql.NullTime) string {
	if t.Valid {
		return t.Time.Format("2006-01-02 15:04:05")
	}
	return ""
}

func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}
