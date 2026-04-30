package logic

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentPopularLogic 热门推荐逻辑
type ContentPopularLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentPopularLogic 创建热门推荐逻辑实例
func NewContentPopularLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentPopularLogic {
	return &ContentPopularLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContentPopular 获取热门推荐列表
func (l *ContentPopularLogic) ContentPopular(req *types.PopularReq, userID int64) (*types.PopularResp, error) {
	// 1. 解析请求参数
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	period := req.Period
	if period == "" {
		period = "all"
	}

	// 2. 获取用户会员等级
	userVipLevel := int16(0)
	if userID > 0 {
		vipLevel, err := l.getUserVipLevel(userID)
		if err != nil {
			logx.Errorf("获取用户会员等级失败: user_id=%d, err=%v", userID, err)
		} else {
			userVipLevel = vipLevel
		}
	}

	// 3. 确定统计时间范围
	startTime := l.getStartTimeByPeriod(period)

	// 4. 获取热度统计数据
	playStats, err := l.getPlayStats(startTime)
	if err != nil {
		logx.Errorf("获取播放统计失败: %v", err)
		playStats = make(map[int64]int64)
	}

	likeStats, err := l.getLikeStats()
	if err != nil {
		logx.Errorf("获取点赞统计失败: %v", err)
		likeStats = make(map[int64]int64)
	}

	recentStats, err := l.getRecentPlayStats(7)
	if err != nil {
		logx.Errorf("获取近期播放统计失败: %v", err)
		recentStats = make(map[int64]int64)
	}

	// 5. 计算最大值用于归一化
	maxPlayCount := int64(1)
	maxLikeCount := int64(1)
	maxRecentPlay := int64(1)

	for _, v := range playStats {
		if v > maxPlayCount {
			maxPlayCount = v
		}
	}
	for _, v := range likeStats {
		if v > maxLikeCount {
			maxLikeCount = v
		}
	}
	for _, v := range recentStats {
		if v > maxRecentPlay {
			maxRecentPlay = v
		}
	}

	// 6. 获取候选内容列表
	candidates, err := l.getPopularCandidates(userVipLevel, req.Category, startTime)
	if err != nil {
		return nil, fmt.Errorf("获取候选内容失败: %v", err)
	}

	if len(candidates) == 0 {
		return &types.PopularResp{
			Total:     0,
			List:      []types.PopularItem{},
			Period:    period,
			UpdatedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		}, nil
	}

	// 7. 计算热度得分
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	var scoredItems []popularScoredItem
	for _, c := range candidates {
		playCount := playStats[c.ID]
		likeCount := likeStats[c.ID]
		recentCount := recentStats[c.ID]

		// 播放得分 = log(play_count+1) / log(max_play_count+1) * 100
		playScore := 0.0
		if maxPlayCount > 0 {
			playScore = math.Log(float64(playCount)+1) / math.Log(float64(maxPlayCount)+1) * 100
		}

		// 点赞得分 = log(like_count+1) / log(max_like_count+1) * 100
		likeScore := 0.0
		if maxLikeCount > 0 {
			likeScore = math.Log(float64(likeCount)+1) / math.Log(float64(maxLikeCount)+1) * 100
		}

		// 近期得分 = recent_play / max_recent_play * 100
		recentScore := 0.0
		if maxRecentPlay > 0 {
			recentScore = float64(recentCount) / float64(maxRecentPlay) * 100
		}

		// VIP 内容额外加分
		vipBonus := 0.0
		if c.VipLevel > 0 {
			vipBonus = 20
		}

		// 综合得分 = 播放得分*0.4 + 点赞得分*0.3 + 近期得分*0.2 + VIP加分*0.1
		totalScore := playScore*0.4 + likeScore*0.3 + recentScore*0.2 + vipBonus*0.1

		// 判断是否为新内容（7天内创建）
		isNew := c.CreatedAt.After(sevenDaysAgo)

		scoredItems = append(scoredItems, popularScoredItem{
			ID:          c.ID,
			Title:       c.Title,
			CoverURL:    c.CoverURL,
			Artist:      c.Artist,
			Duration:    c.Duration,
			VipLevel:    c.VipLevel,
			IsVip:       c.VipLevel > 0,
			CanPlay:     userVipLevel >= c.VipLevel,
			HotScore:    int(totalScore),
			PlayCount:   playCount,
			LikeCount:   likeCount,
			IsNew:       isNew,
			PublishedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// 8. 按热度得分降序排序
	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].HotScore > scoredItems[j].HotScore
	})

	// 9. 添加排名并截取前 limit 条
	var result []types.PopularItem
	for i, item := range scoredItems {
		if i >= limit {
			break
		}
		item.Rank = i + 1
		result = append(result, types.PopularItem(item))
	}

	return &types.PopularResp{
		Total:     int64(len(result)),
		List:      result,
		Period:    period,
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// popularScoredItem 带热度得分的内容项
type popularScoredItem struct {
	ID          int64
	Title       string
	CoverURL    string
	Artist      string
	Duration    int
	VipLevel    int16
	IsVip       bool
	CanPlay     bool
	HotScore    int
	PlayCount   int64
	LikeCount   int64
	IsNew       bool
	Rank        int
	PublishedAt string
}

// getStartTimeByPeriod 根据周期获取开始时间
func (l *ContentPopularLogic) getStartTimeByPeriod(period string) *time.Time {
	now := time.Now()
	var startTime time.Time

	switch period {
	case "day":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		startTime = now.AddDate(0, 0, -7)
	case "month":
		startTime = now.AddDate(0, 0, -30)
	default:
		return nil
	}

	return &startTime
}

// getPlayStats 获取播放统计
func (l *ContentPopularLogic) getPlayStats(startTime *time.Time) (map[int64]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	var query string
	var args []interface{}

	if startTime != nil {
		query = `
			SELECT content_id, COUNT(*) as play_count
			FROM play_history
			WHERE started_at >= $1 AND status = 2
			GROUP BY content_id
		`
		args = append(args, *startTime)
	} else {
		query = `
			SELECT content_id, COUNT(*) as play_count
			FROM play_history
			WHERE status = 2
			GROUP BY content_id
		`
	}

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[int64]int64)
	for rows.Next() {
		var contentID int64
		var count int64
		if err := rows.Scan(&contentID, &count); err != nil {
			continue
		}
		stats[contentID] = count
	}

	return stats, nil
}

// getLikeStats 获取点赞统计
func (l *ContentPopularLogic) getLikeStats() (map[int64]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT content_id, COUNT(*) as like_count
		FROM user_likes
		GROUP BY content_id
	`

	rows, err := l.svcCtx.DB.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[int64]int64)
	for rows.Next() {
		var contentID int64
		var count int64
		if err := rows.Scan(&contentID, &count); err != nil {
			continue
		}
		stats[contentID] = count
	}

	return stats, nil
}

// getRecentPlayStats 获取近期播放统计
func (l *ContentPopularLogic) getRecentPlayStats(days int) (map[int64]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	startTime := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT content_id, COUNT(*) as recent_count
		FROM play_history
		WHERE started_at >= $1 AND status = 2
		GROUP BY content_id
	`

	rows, err := l.svcCtx.DB.Raw(query, startTime).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[int64]int64)
	for rows.Next() {
		var contentID int64
		var count int64
		if err := rows.Scan(&contentID, &count); err != nil {
			continue
		}
		stats[contentID] = count
	}

	return stats, nil
}

// popularCandidate 热门候选内容
type popularCandidate struct {
	ID        int64
	Title     string
	CoverURL  string
	Artist    string
	Duration  int
	VipLevel  int16
	CreatedAt time.Time
}

// getPopularCandidates 获取热门候选内容
func (l *ContentPopularLogic) getPopularCandidates(userVipLevel int16, categoryID int64, startTime *time.Time) ([]popularCandidate, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	// 使用 GORM 链式查询
	db := l.svcCtx.DB.Model(&dao.ContentCatalog{}).
		Select("id, title, cover_url, artist, duration_sec, vip_level, created_at").
		Where("status = ?", 1).
		Where("vip_level <= ?", int(userVipLevel)).
		Order("created_at DESC").
		Limit(200)

	// 分类过滤
	if categoryID > 0 {
		db = db.Where("category_id = ?", categoryID)
	}

	rows, err := db.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []popularCandidate
	for rows.Next() {
		var item popularCandidate
		if err := rows.Scan(
			&item.ID, &item.Title, &item.CoverURL, &item.Artist, &item.Duration, &item.VipLevel, &item.CreatedAt,
		); err != nil {
			continue
		}
		candidates = append(candidates, item)
	}

	// 过滤时间范围
	if startTime != nil {
		var filtered []popularCandidate
		for _, c := range candidates {
			if c.CreatedAt.After(*startTime) || c.CreatedAt.Equal(*startTime) {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	return candidates, nil
}

// getUserVipLevel 获取用户会员等级
func (l *ContentPopularLogic) getUserVipLevel(userID int64) (int16, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT level 
		FROM user_member 
		WHERE user_id = $1 AND status = 1 
		AND (is_permanent = 1 OR expire_at > NOW())
		LIMIT 1
	`

	var level int16
	err := l.svcCtx.DB.Raw(query, userID).Scan(&level).Error
	if err != nil {
		return 0, err
	}

	return level, nil
}
