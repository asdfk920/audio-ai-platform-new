package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentNewLogic 最新内容推荐逻辑
type ContentNewLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentNewLogic 创建最新内容推荐逻辑实例
func NewContentNewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentNewLogic {
	return &ContentNewLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContentNew 获取最新内容推荐列表
func (l *ContentNewLogic) ContentNew(req *types.NewContentReq, userID int64) (*types.NewContentResp, error) {
	// 1. 解析请求参数
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	days := int(req.Days)
	if days <= 0 {
		days = 30
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

	// 3. 计算时间范围
	now := time.Now()
	startTime := now.AddDate(0, 0, -days)
	endTime := now

	// 4. 获取用户订阅的艺术家 ID 列表
	var subscribedArtistIDs []int64
	if userID > 0 {
		subscribedArtistIDs, _ = l.getSubscribedArtistIDs(userID)
	}

	// 5. 查询新内容
	contents, err := l.getNewContents(userVipLevel, req.Category, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("获取新内容失败: %v", err)
	}

	if len(contents) == 0 {
		return &types.NewContentResp{
			Total:              0,
			List:               []types.NewContentItem{},
			Days:               days,
			HasSubscriptionNew: false,
			UpdatedAt:          now.Format("2006-01-02T15:04:05Z"),
		}, nil
	}

	// 6. 处理结果
	threeDaysAgo := now.AddDate(0, 0, -3)
	var result []types.NewContentItem
	hasSubscriptionNew := false
	var subscriptionArtistNames []string

	for _, c := range contents {
		// 计算天数差
		daysAgo := int(now.Sub(c.CreatedAt).Hours() / 24)

		// 判断是否为新内容（3天内）
		isNew := c.CreatedAt.After(threeDaysAgo)

		// 判断是否订阅（通过 artist_id 匹配）
		isSubArtist := contains(subscribedArtistIDs, c.ArtistID)

		// 生成推荐理由
		reason := "新上架推荐"
		if isSubArtist {
			reason = "订阅艺术家新作品"
			hasSubscriptionNew = true
			subscriptionArtistNames = append(subscriptionArtistNames, c.Artist)
		}

		// 生成新内容标签
		newBadge := ""
		if isNew {
			if daysAgo == 0 {
				newBadge = "今日新"
			} else if daysAgo <= 3 {
				newBadge = "3日内新"
			}
		}

		result = append(result, types.NewContentItem{
			ID:                   c.ID,
			Title:                c.Title,
			CoverURL:             c.CoverURL,
			Artist:               c.Artist,
			Duration:             c.Duration,
			VipLevel:             c.VipLevel,
			IsVip:                c.VipLevel > 0,
			CanPlay:              userVipLevel >= c.VipLevel,
			IsNew:                isNew,
			NewBadge:             newBadge,
			Reason:               reason,
			DaysAgo:              daysAgo,
			PublishedAt:          c.CreatedAt.Format("2006-01-02T15:04:05Z"),
			IsSubscriptionArtist: isSubArtist,
			IsSubscriptionSeries: false,
		})
	}

	return &types.NewContentResp{
		Total:               int64(len(result)),
		List:                result,
		Days:                days,
		HasSubscriptionNew:  hasSubscriptionNew,
		SubscriptionArtists: subscriptionArtistNames,
		UpdatedAt:           now.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// newContentItem 新内容查询结果
type newContentItem struct {
	ID        int64
	Title     string
	CoverURL  string
	Artist    string
	ArtistID  int64
	Duration  int
	VipLevel  int16
	CreatedAt time.Time
}

// getNewContents 获取新内容列表
func (l *ContentNewLogic) getNewContents(userVipLevel int16, categoryID int64, startTime, endTime time.Time, limit int) ([]newContentItem, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	// 使用 GORM 链式查询
	db := l.svcCtx.DB.Model(&dao.ContentCatalog{}).
		Select("id, title, cover_url, artist, artist_id, duration_sec, vip_level, created_at").
		Where("status = ?", 1).
		Where("vip_level <= ?", int(userVipLevel)).
		Where("created_at >= ?", startTime).
		Where("created_at <= ?", endTime).
		Order("created_at DESC").
		Limit(limit)

	// 分类过滤
	if categoryID > 0 {
		db = db.Where("category_id = ?", categoryID)
	}

	rows, err := db.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []newContentItem
	for rows.Next() {
		var item newContentItem
		if err := rows.Scan(
			&item.ID, &item.Title, &item.CoverURL, &item.Artist, &item.ArtistID,
			&item.Duration, &item.VipLevel, &item.CreatedAt,
		); err != nil {
			continue
		}
		contents = append(contents, item)
	}

	return contents, nil
}

// getUserVipLevel 获取用户会员等级
func (l *ContentNewLogic) getUserVipLevel(userID int64) (int16, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	var vipLevel int16
	err := l.svcCtx.DB.Raw("SELECT COALESCE(vip_level, 0) FROM users WHERE id = ?", userID).Scan(&vipLevel).Error
	if err != nil {
		return 0, err
	}
	return vipLevel, nil
}

// getSubscribedArtistIDs 获取用户订阅的艺术家 ID 列表
func (l *ContentNewLogic) getSubscribedArtistIDs(userID int64) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	rows, err := l.svcCtx.DB.Raw(
		"SELECT artist_id FROM user_subscriptions WHERE user_id = ? AND artist_id > 0 AND is_active = 1",
		userID,
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// contains 检查切片中是否包含指定值
func contains(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
