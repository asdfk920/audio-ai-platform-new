package logic

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentRecommendLogic 内容推荐逻辑
type ContentRecommendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentRecommendLogic 创建内容推荐逻辑实例
func NewContentRecommendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentRecommendLogic {
	return &ContentRecommendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContentRecommend 获取个性化推荐
// 流程：
//  1. 解析请求参数
//  2. 获取用户会员等级
//  3. 提取用户特征数据（播放历史、收藏、订阅）
//  4. 生成推荐候选集
//  5. 计算相似度得分
//  6. 加权排序
//  7. 组装推荐理由
//  8. 返回推荐结果
func (l *ContentRecommendLogic) ContentRecommend(req *types.RecommendReq, userID int64) (*types.RecommendResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 1. 解析请求参数
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	recType := req.Type
	if recType == "" {
		recType = "all"
	}

	// 2. 获取用户会员等级
	userVipLevel := int16(0)
	vipLevel, err := l.getUserVipLevel(userID)
	if err != nil {
		logx.Errorf("获取用户会员等级失败: user_id=%d, err=%v", userID, err)
	} else {
		userVipLevel = vipLevel
	}

	// 3. 提取用户特征数据
	userFeatures, err := l.extractUserFeatures(userID)
	if err != nil {
		logx.Errorf("提取用户特征失败: user_id=%d, err=%v", userID, err)
		userFeatures = &userFeatureData{}
	}

	// 4. 生成推荐候选集
	candidates, err := l.generateCandidates(userVipLevel, userFeatures.PlayedContentIDs)
	if err != nil {
		return nil, fmt.Errorf("生成候选集失败: %v", err)
	}

	if len(candidates) == 0 {
		return &types.RecommendResp{
			Total: 0,
			List:  []types.RecommendItem{},
		}, nil
	}

	// 5. 计算推荐得分
	scoredItems := l.calculateScores(candidates, userFeatures, recType)

	// 6. 按得分排序
	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].Score > scoredItems[j].Score
	})

	// 7. 截取前 limit 条
	if len(scoredItems) > limit {
		scoredItems = scoredItems[:limit]
	}

	// 8. 组装推荐理由
	reasonMap := make(map[string]int64)
	resultList := make([]types.RecommendItem, len(scoredItems))
	for i := range scoredItems {
		scoredItems[i].Reason = l.generateReason(scoredItems[i], userFeatures)
		reasonMap[scoredItems[i].Reason]++
		resultList[i] = scoredItems[i].RecommendItem
		resultList[i].Reason = scoredItems[i].Reason
	}

	return &types.RecommendResp{
		Total:  int64(len(resultList)),
		List:   resultList,
		Reason: reasonMap,
	}, nil
}

// userFeatureData 用户特征数据
type userFeatureData struct {
	PlayedContentIDs   []int64
	PlayedTagWeights   map[int64]float64
	FavoriteContentIDs []int64
	FavoriteTagWeights map[int64]float64
	SubscribedArtists  map[string]bool
	SubscribedSeries   map[string]bool
}

// candidateItem 候选内容
type candidateItem struct {
	ID         int64
	Title      string
	CoverURL   string
	Artist     string
	Duration   int
	VipLevel   int16
	CategoryID *int64
	CreatedAt  time.Time
	TagIDs     []int64
	ArtistName string
	SeriesName string
}

// scoredItem 带得分的推荐项
type scoredItem struct {
	types.RecommendItem
	TagOverlap  float64
	ArtistMatch bool
	SeriesMatch bool
	HotScore    float64
	FreshScore  float64
}

// extractUserFeatures 提取用户特征数据
func (l *ContentRecommendLogic) extractUserFeatures(userID int64) (*userFeatureData, error) {
	features := &userFeatureData{
		PlayedTagWeights:   make(map[int64]float64),
		FavoriteTagWeights: make(map[int64]float64),
		SubscribedArtists:  make(map[string]bool),
		SubscribedSeries:   make(map[string]bool),
	}

	// 获取用户最近播放的内容
	playedIDs, err := l.getRecentPlayedContentIDs(userID, 50)
	if err != nil {
		logx.Errorf("获取播放历史失败: %v", err)
	} else {
		features.PlayedContentIDs = playedIDs
	}

	// 获取播放内容的标签权重
	if len(features.PlayedContentIDs) > 0 {
		tagWeights, err := l.getContentTagWeights(features.PlayedContentIDs)
		if err != nil {
			logx.Errorf("获取播放标签权重失败: %v", err)
		} else {
			features.PlayedTagWeights = tagWeights
		}
	}

	// 获取用户收藏的内容
	favIDs, err := l.getUserFavoriteContentIDs(userID)
	if err != nil {
		logx.Errorf("获取收藏内容失败: %v", err)
	} else {
		features.FavoriteContentIDs = favIDs
	}

	// 获取收藏内容的标签权重
	if len(features.FavoriteContentIDs) > 0 {
		tagWeights, err := l.getContentTagWeights(features.FavoriteContentIDs)
		if err != nil {
			logx.Errorf("获取收藏标签权重失败: %v", err)
		} else {
			features.FavoriteTagWeights = tagWeights
		}
	}

	// 获取用户订阅
	subs, err := l.getUserSubscriptions(userID)
	if err != nil {
		logx.Errorf("获取订阅失败: %v", err)
	} else {
		for _, sub := range subs {
			if sub.SubscribeType == 1 {
				features.SubscribedArtists[sub.TargetName] = true
			} else if sub.SubscribeType == 2 {
				features.SubscribedSeries[sub.TargetName] = true
			}
		}
	}

	return features, nil
}

// subscriptionItem 订阅项
type subscriptionItem struct {
	SubscribeType int16
	TargetID      int64
	TargetName    string
}

// getRecentPlayedContentIDs 获取用户最近播放的内容 ID
func (l *ContentRecommendLogic) getRecentPlayedContentIDs(userID int64, limit int) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT content_id 
		FROM (
			SELECT DISTINCT ON (content_id) content_id, played_at 
			FROM user_play_record 
			WHERE user_id = $1 
			ORDER BY content_id, played_at DESC
		) sub
		ORDER BY played_at DESC 
		LIMIT $2
	`

	rows, err := l.svcCtx.DB.Raw(query, userID, limit).Rows()
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

// getUserFavoriteContentIDs 获取用户收藏的内容 ID
func (l *ContentRecommendLogic) getUserFavoriteContentIDs(userID int64) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT content_id 
		FROM user_favorite 
		WHERE user_id = $1 
		ORDER BY created_at DESC
	`

	rows, err := l.svcCtx.DB.Raw(query, userID).Rows()
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

// getUserSubscriptions 获取用户订阅
func (l *ContentRecommendLogic) getUserSubscriptions(userID int64) ([]subscriptionItem, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	query := `
		SELECT subscribe_type, target_id, target_name 
		FROM user_subscriptions 
		WHERE user_id = $1
	`

	rows, err := l.svcCtx.DB.Raw(query, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []subscriptionItem
	for rows.Next() {
		var sub subscriptionItem
		if err := rows.Scan(&sub.SubscribeType, &sub.TargetID, &sub.TargetName); err != nil {
			continue
		}
		subs = append(subs, sub)
	}

	return subs, nil
}

// getContentTagWeights 获取内容列表的标签权重
func (l *ContentRecommendLogic) getContentTagWeights(contentIDs []int64) (map[int64]float64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	// 构造 IN 子句
	placeholders := make([]string, len(contentIDs))
	args := make([]interface{}, len(contentIDs))
	for i, id := range contentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT tag_id, COUNT(*) as cnt 
		FROM content_tag_relation 
		WHERE content_id IN (%s) 
		GROUP BY tag_id
	`, joinPlaceholders(len(contentIDs)))

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weights := make(map[int64]float64)
	for rows.Next() {
		var tagID int64
		var cnt int64
		if err := rows.Scan(&tagID, &cnt); err != nil {
			continue
		}
		weights[tagID] = float64(cnt)
	}

	return weights, nil
}

// generateCandidates 生成推荐候选集
func (l *ContentRecommendLogic) generateCandidates(userVipLevel int16, playedIDs []int64) ([]candidateItem, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	now := time.Now()

	// 基础查询条件
	whereConditions := []string{
		"c.is_deleted = 0",
		"c.status = 1",
		fmt.Sprintf("c.vip_level <= %d", userVipLevel),
		"(c.audio_valid_from IS NULL OR c.audio_valid_from <= $1)",
		"(c.audio_valid_until IS NULL OR c.audio_valid_until >= $1)",
	}
	args := []interface{}{now}
	argIdx := 2

	// 排除已播放的内容
	if len(playedIDs) > 0 {
		placeholders := make([]string, len(playedIDs))
		for i, id := range playedIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIdx+i)
			args = append(args, id)
		}
		whereConditions = append(whereConditions, fmt.Sprintf("c.id NOT IN (%s)", joinStrings(placeholders)))
	}

	whereClause := joinStrings(whereConditions, " AND ")

	query := fmt.Sprintf(`
		SELECT c.id, c.title, c.cover_url, c.artist, c.duration_sec, 
		       c.vip_level, c.category_id, c.created_at,
		       COALESCE(string_agg(DISTINCT ctr.tag_id::text, ','), '') as tag_ids
		FROM content c
		LEFT JOIN content_tag_relation ctr ON c.id = ctr.content_id
		WHERE %s
		GROUP BY c.id, c.title, c.cover_url, c.artist, c.duration_sec, 
		         c.vip_level, c.category_id, c.created_at
		ORDER BY c.sort_order DESC, c.created_at DESC
		LIMIT 200
	`, whereClause)

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		logx.Errorf("生成候选集查询失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var candidates []candidateItem
	for rows.Next() {
		var item candidateItem
		var tagIDsStr string
		if err := rows.Scan(
			&item.ID, &item.Title, &item.CoverURL, &item.Artist, &item.Duration,
			&item.VipLevel, &item.CategoryID, &item.CreatedAt, &tagIDsStr,
		); err != nil {
			logx.Errorf("扫描候选集行失败: %v", err)
			continue
		}

		if tagIDsStr != "" {
			item.TagIDs = parseTagIDList(tagIDsStr)
		}

		candidates = append(candidates, item)
	}

	logx.Infof("生成候选集完成: user_id=%d, candidates=%d", userVipLevel, len(candidates))
	return candidates, nil
}

// calculateScores 计算推荐得分
func (l *ContentRecommendLogic) calculateScores(candidates []candidateItem, features *userFeatureData, recType string) []scoredItem {
	// 根据推荐类型调整权重
	var wTag, wSub, wHot, wFresh float64
	switch recType {
	case "similar":
		wTag, wSub, wHot, wFresh = 0.5, 0.15, 0.15, 0.2
	case "following":
		wTag, wSub, wHot, wFresh = 0.15, 0.5, 0.15, 0.2
	case "new":
		wTag, wSub, wHot, wFresh = 0.15, 0.15, 0.2, 0.5
	default:
		wTag, wSub, wHot, wFresh = 0.3, 0.2, 0.1, 0.4
	}

	now := time.Now()
	var scored []scoredItem

	for _, c := range candidates {
		item := scoredItem{
			RecommendItem: types.RecommendItem{
				ID:           c.ID,
				Title:        c.Title,
				CoverURL:     c.CoverURL,
				Artist:       c.Artist,
				Duration:     c.Duration,
				VipLevel:     c.VipLevel,
				IsVipContent: c.VipLevel > 0,
				CanPlay:      true,
				CanPlayFull:  true,
			},
		}

		// 标签相似度得分
		tagScore := l.calcTagSimilarity(c.TagIDs, features.PlayedTagWeights, features.FavoriteTagWeights)
		item.TagOverlap = tagScore

		// 订阅匹配得分
		artistMatch := features.SubscribedArtists[c.Artist]
		seriesMatch := features.SubscribedSeries[c.Artist]
		item.ArtistMatch = artistMatch
		item.SeriesMatch = seriesMatch
		subScore := 0.0
		if artistMatch {
			subScore += 0.6
		}
		if seriesMatch {
			subScore += 0.4
		}

		// 热度得分（基于排序权重）
		catID := int64(0)
		if c.CategoryID != nil {
			catID = *c.CategoryID
		}
		item.HotScore = float64(catID%100) / 100.0

		// 新鲜度得分
		daysSincePublish := now.Sub(c.CreatedAt).Hours() / 24.0
		item.FreshScore = math.Max(0, 1.0-daysSincePublish/365.0)

		// 综合得分
		totalScore := wTag*tagScore + wSub*subScore + wHot*item.HotScore + wFresh*item.FreshScore
		item.Score = int(totalScore * 100)

		scored = append(scored, item)
	}

	return scored
}

// calcTagSimilarity 计算标签相似度
func (l *ContentRecommendLogic) calcTagSimilarity(contentTagIDs []int64, playedWeights, favWeights map[int64]float64) float64 {
	if len(contentTagIDs) == 0 {
		return 0
	}

	var totalWeight float64
	var matchWeight float64

	for _, tagID := range contentTagIDs {
		totalWeight++
		playW := playedWeights[tagID] * 2
		favW := favWeights[tagID] * 3
		if playW+favW > 0 {
			matchWeight += math.Min(1.0, (playW+favW)/5.0)
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return matchWeight / totalWeight
}

// generateReason 生成推荐理由
func (l *ContentRecommendLogic) generateReason(item scoredItem, features *userFeatureData) string {
	if item.ArtistMatch {
		return "订阅更新"
	}
	if item.SeriesMatch {
		return "订阅更新"
	}
	if item.TagOverlap > 0.5 {
		return "根据播放历史推荐"
	}
	if item.FreshScore > 0.8 {
		return "新内容推荐"
	}
	if item.HotScore > 0.7 {
		return "热门推荐"
	}
	return "猜你喜欢"
}

// joinPlaceholders 生成占位符字符串
func joinPlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := 0; i < n; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return joinStrings(placeholders, ",")
}

// joinStrings 连接字符串
func joinStrings(strs []string, sep ...string) string {
	s := ","
	if len(sep) > 0 {
		s = sep[0]
	}
	result := ""
	for i, str := range strs {
		if i > 0 {
			result += s
		}
		result += str
	}
	return result
}

// parseTagIDList 解析标签 ID 列表
func parseTagIDList(tagIDsStr string) []int64 {
	if tagIDsStr == "" {
		return nil
	}

	var ids []int64
	// 简单解析逗号分隔的 ID
	start := 0
	for i := 0; i <= len(tagIDsStr); i++ {
		if i == len(tagIDsStr) || tagIDsStr[i] == ',' {
			if i > start {
				var id int64
				fmt.Sscanf(tagIDsStr[start:i], "%d", &id)
				if id > 0 {
					ids = append(ids, id)
				}
			}
			start = i + 1
		}
	}
	return ids
}

// getUserVipLevel 获取用户会员等级
func (l *ContentRecommendLogic) getUserVipLevel(userID int64) (int16, error) {
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
