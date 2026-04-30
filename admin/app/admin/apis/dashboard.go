package apis

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
)

// 说明：统计 SQL 按 PostgreSQL 编写（settings.yml 默认 driver: postgres）。

// Dashboard 首页数据概览（GET /api/v1/dashboard）
type Dashboard struct {
	api.Api
}

type dayCountRow struct {
	Day string `gorm:"column:day"`
	Cnt int64  `gorm:"column:cnt"`
}

// Get 聚合平台用户数、内容、设备与近 14 日趋势（PostgreSQL）
func (e Dashboard) Get(c *gin.Context) {
	e.MakeContext(c)
	if err := e.MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	db := e.Orm

	var totalUsers, activeUsers, totalContents, todayPlays, onlineDevices int64
	var processedTotal, processedCompleted int64

	_ = db.Raw(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&totalUsers).Error
	_ = db.Raw(`
		SELECT COUNT(*) FROM users
		WHERE deleted_at IS NULL AND last_login_at IS NOT NULL AND last_login_at::date = CURRENT_DATE
	`).Scan(&activeUsers).Error

	_ = db.Raw(`SELECT COUNT(*) FROM contents`).Scan(&totalContents).Error

	_ = db.Raw(`
		SELECT COUNT(*) FROM content_play_records
		WHERE play_start_time::date = CURRENT_DATE
	`).Scan(&todayPlays).Error

	_ = db.Raw(`SELECT COUNT(*) FROM device WHERE deleted_at IS NULL AND online_status = 1`).Scan(&onlineDevices).Error

	_ = db.Raw(`SELECT COUNT(*) FROM processed_contents`).Scan(&processedTotal).Error
	_ = db.Raw(`SELECT COUNT(*) FROM processed_contents WHERE status = 'completed'`).Scan(&processedCompleted).Error

	var processedRate float64
	if processedTotal > 0 {
		processedRate = float64(processedCompleted) / float64(processedTotal)
	}

	since := time.Now().AddDate(0, 0, -13).Truncate(24 * time.Hour)
	var regRows []dayCountRow
	_ = db.Raw(`
		SELECT to_char(created_at::date, 'YYYY-MM-DD') AS day, COUNT(*)::bigint AS cnt
		FROM users
		WHERE deleted_at IS NULL AND created_at >= ?
		GROUP BY created_at::date
		ORDER BY created_at::date
	`, since).Scan(&regRows).Error
	regMap := make(map[string]int64)
	for _, r := range regRows {
		regMap[r.Day] = r.Cnt
	}

	var playRows []dayCountRow
	_ = db.Raw(`
		SELECT to_char(play_start_time::date, 'YYYY-MM-DD') AS day, COUNT(*)::bigint AS cnt
		FROM content_play_records
		WHERE play_start_time >= ?
		GROUP BY play_start_time::date
		ORDER BY play_start_time::date
	`, since).Scan(&playRows).Error
	playMap := make(map[string]int64)
	for _, r := range playRows {
		playMap[r.Day] = r.Cnt
	}

	userRegTrend := fillLast14Days(regMap)
	playbackTrend := fillLast14Days(playMap)

	e.OK(gin.H{
		"total_users":       totalUsers,
		"active_users":      activeUsers,
		"total_contents":    totalContents,
		"processed_rate":    processedRate,
		"today_play_count":  todayPlays,
		"online_devices":    onlineDevices,
		"user_reg_trend":    userRegTrend,
		"playback_trend":    playbackTrend,
	}, "ok")
}

func fillLast14Days(m map[string]int64) []gin.H {
	out := make([]gin.H, 0, 14)
	now := time.Now()
	for i := 13; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, gin.H{"date": d, "count": m[d]})
	}
	return out
}
