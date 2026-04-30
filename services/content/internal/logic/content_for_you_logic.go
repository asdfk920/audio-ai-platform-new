package logic

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentForYouLogic 猜你喜欢逻辑
type ContentForYouLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentForYouLogic 创建猜你喜欢逻辑实例
func NewContentForYouLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentForYouLogic {
	return &ContentForYouLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ForYou 获取猜你喜欢推荐列表
func (l *ContentForYouLogic) ForYou(req *types.ForYouReq, userID int64) (*types.ForYouResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	excludeIDs := parseExcludeIDs(req.ExcludeIDs)

	userVipLevel := int16(0)
	vipLevel, err := l.getUserVipLevel(userID)
	if err != nil {
		logx.Errorf("获取用户会员等级失败: %v", err)
	} else {
		userVipLevel = vipLevel
	}

	now := time.Now()

	profile, err := l.buildUserProfile(userID)
	if err != nil {
		logx.Errorf("构建用户画像失败: %v", err)
		profile = &userProfile{}
	}

	cfScores, err := l.collaborativeFiltering(userID, profile.PlayedIDs, excludeIDs)
	if err != nil {
		logx.Errorf("协同过滤失败: %v", err)
		cfScores = make(map[int64]float64)
	}

	tagScores, tagReasons, err := l.tagMatching(profile.TopTags, userVipLevel, excludeIDs)
	if err != nil {
		logx.Errorf("标签匹配失败: %v", err)
		tagScores = make(map[int64]float64)
		tagReasons = make(map[int64]string)
	}

	favScores, err := l.favoriteExtension(userID, profile.FavoriteIDs, excludeIDs, userVipLevel)
	if err != nil {
		logx.Errorf("收藏延伸失败: %v", err)
		favScores = make(map[int64]float64)
	}

	hotScores, err := l.hotContentSupplement(userVipLevel, append(excludeIDs, getMapKeys(cfScores)...), append(getMapKeys(tagScores), getMapKeys(favScores)...), limit*2)
	if err != nil {
		logx.Errorf("热门补足失败: %v", err)
		hotScores = make(map[int64]float64)
	}

	allCandidates := mergeScoreMaps(cfScores, tagScores, favScores, hotScores)

	var scoredItems []forYouScoredItem
	for contentID, scores := range allCandidates {
		finalScore := scores.CFScore*0.4 + scores.TagScore*0.25 + scores.FavScore*0.2 + scores.HotScore*0.15

		scoredItems = append(scoredItems, forYouScoredItem{
			ContentID: contentID,
			Score:     finalScore,
			CFScore:   scores.CFScore,
			TagScore:  scores.TagScore,
			FavScore:  scores.FavScore,
			HotScore:  scores.HotScore,
		})
	}

	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].Score > scoredItems[j].Score
	})

	candidateContents, err := l.getCandidateContents(getContentIDs(scoredItems[:min(limit*2, len(scoredItems))]), userVipLevel)
	if err != nil {
		return nil, fmt.Errorf("获取候选内容失败: %v", err)
	}

	contentMap := make(map[int64]*candidateContent)
	for i := range candidateContents {
		contentMap[candidateContents[i].ID] = &candidateContents[i]
	}

	var result []types.ForYouItem
	count := 0
	for _, item := range scoredItems {
		if count >= limit {
			break
		}

		content, ok := contentMap[item.ContentID]
		if !ok {
			continue
		}

		reason := "你可能喜欢这类内容"
		source := "综合推荐"

		if item.FavScore > 0 && (item.FavScore >= item.TagScore && item.FavScore >= item.CFScore) {
			reason = "根据你收藏的内容推荐"
			source = "收藏延伸"
		} else if item.TagScore > 0 && (item.TagScore >= item.CFScore && item.TagScore >= item.FavScore) {
			if r, ok := tagReasons[item.ContentID]; ok {
				reason = r
			} else {
				reason = fmt.Sprintf("标签匹配度 %.0f%%", item.TagScore)
			}
			source = "标签匹配"
		} else if item.CFScore > 0 {
			reason = "相似用户也在听"
			source = "协同过滤"
		} else if item.HotScore > 0 {
			reason = "热门内容推荐"
			source = "热门推荐"
		}

		result = append(result, types.ForYouItem{
			ID:       content.ID,
			Title:    content.Title,
			CoverURL: content.CoverURL,
			Artist:   content.Artist,
			Duration: content.Duration,
			VipLevel: content.VipLevel,
			IsVip:    content.VipLevel > 0,
			CanPlay:  userVipLevel >= content.VipLevel,
			Score:    math.Round(item.Score*100) / 100,
			Reason:   reason,
			Tags:     content.Tags,
			Source:   source,
		})
		count++
	}

	return &types.ForYouResp{
		Total:     int64(len(result)),
		List:      result,
		UserTags:  profile.TopTags,
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// userProfile 用户画像
type userProfile struct {
	PlayedIDs        []int64
	FavoriteIDs      []int64
	SubArtists       []int64
	SubSeries        []int64
	TagWeights       map[string]int
	TopTags          []string
	PreferredArtists []string
}

// forYouScoredItem 得分项
type forYouScoredItem struct {
	ContentID int64
	Score     float64
	CFScore   float64
	TagScore  float64
	FavScore  float64
	HotScore  float64
}

// scoreComponents 得分组件
type scoreComponents struct {
	CFScore  float64
	TagScore float64
	FavScore float64
	HotScore float64
}

// candidateContent 候选内容
type candidateContent struct {
	ID       int64
	Title    string
	CoverURL string
	Artist   string
	Duration int
	VipLevel int16
	Tags     []string
}

// buildUserProfile 构建用户画像
func (l *ContentForYouLogic) buildUserProfile(userID int64) (*userProfile, error) {
	profile := &userProfile{
		TagWeights: make(map[string]int),
	}

	playedIDs, _ := l.getRecentPlayHistory(userID, 50)
	profile.PlayedIDs = playedIDs

	favoriteIDs, _ := l.getFavoriteIDs(userID)
	profile.FavoriteIDs = favoriteIDs

	subArtists, _ := l.getSubscribedArtistIDs(userID)
	profile.SubArtists = subArtists

	subSeries, _ := l.getSubscribedSeriesIDs(userID)
	profile.SubSeries = subSeries

	tagWeights, err := l.getTagWeightsFromPlays(playedIDs)
	if err == nil {
		profile.TagWeights = tagWeights
	}

	type pairs []struct {
		Tag   string
		Count int
	}
	var sortedTags pairs
	for tag, count := range profile.TagWeights {
		sortedTags = append(sortedTags, struct {
			Tag   string
			Count int
		}{tag, count})
	}
	sort.Slice(sortedTags, func(i, j int) bool {
		return sortedTags[i].Count > sortedTags[j].Count
	})

	for i, p := range sortedTags {
		if i >= 10 {
			break
		}
		profile.TopTags = append(profile.TopTags, p.Tag)
	}

	preferredArtists, _ := l.getPreferredArtists(playedIDs)
	profile.PreferredArtists = preferredArtists

	return profile, nil
}

// collaborativeFiltering 协同过滤推荐
func (l *ContentForYouLogic) collaborativeFiltering(userID int64, playedIDs, excludeIDs []int64) (map[int64]float64, error) {
	if len(playedIDs) == 0 || l.svcCtx.DB == nil {
		return make(map[int64]float64), nil
	}

	rows, err := l.svcCtx.DB.Raw(`
		SELECT upr2.content_id, COUNT(*) as like_count
		FROM user_play_record upr1
		JOIN user_play_record upr2 ON upr1.user_id != upr2.user_id AND upr1.content_id = upr2.content_id
		WHERE upr1.user_id = ?
		AND upr2.content_id NOT IN (?)
		GROUP BY upr2.content_id
		ORDER BY like_count DESC
		LIMIT 50
	`, userID, playedIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := make(map[int64]float64)
	maxCount := float64(1)
	for rows.Next() {
		var contentID int64
		var count int64
		if err := rows.Scan(&contentID, &count); err != nil {
			continue
		}
		if !containsInt64(excludeIDs, contentID) {
			scores[contentID] = float64(count)
			if float64(count) > maxCount {
				maxCount = float64(count)
			}
		}
	}

	for id := range scores {
		scores[id] = scores[id] / maxCount * 100
	}

	return scores, nil
}

// tagMatching 标签匹配推荐
func (l *ContentForYouLogic) tagMatching(userTags []string, userVipLevel int16, excludeIDs []int64) (map[int64]float64, map[int64]string, error) {
	scores := make(map[int64]float64)
	reasons := make(map[int64]string)

	if len(userTags) == 0 || l.svcCtx.DB == nil {
		return scores, reasons, nil
	}

	query := `
		SELECT c.id, c.title, t.name as tag_name
		FROM content c
		LEFT JOIN content_tags ct ON c.id = ct.content_id
		LEFT JOIN tags t ON ct.tag_id = t.id
		WHERE c.status = 1
		AND c.vip_level <= ?
		AND t.name IN (?)
	`
	args := []interface{}{userVipLevel, userTags}

	if len(excludeIDs) > 0 {
		query += " AND c.id NOT IN (?)"
		args = append(args, excludeIDs)
	}

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	type contentTag struct {
		ID      int64
		Title   string
		TagName string
	}

	var results []contentTag
	for rows.Next() {
		var ct contentTag
		if err := rows.Scan(&ct.ID, &ct.Title, &ct.TagName); err != nil {
			continue
		}
		results = append(results, ct)
	}

	contentTags := make(map[int64][]string)
	for _, r := range results {
		contentTags[r.ID] = append(contentTags[r.ID], r.TagName)
	}

	for contentID, tags := range contentTags {
		intersection := 0
		matchedTag := ""
		for _, tag := range tags {
			for _, userTag := range userTags {
				if tag == userTag {
					intersection++
					matchedTag = tag
					break
				}
			}
		}

		union := len(tags) + len(userTags) - intersection
		jaccard := 0.0
		if union > 0 {
			jaccard = float64(intersection) / float64(union) * 100
		}

		if jaccard > 20 {
			scores[contentID] = jaccard
			reasons[contentID] = fmt.Sprintf("你可能喜欢「%s」类内容", matchedTag)
		}
	}

	return scores, reasons, nil
}

// favoriteExtension 收藏延伸推荐
func (l *ContentForYouLogic) favoriteExtension(userID int64, favoriteIDs, excludeIDs []int64, userVipLevel int16) (map[int64]float64, error) {
	scores := make(map[int64]float64)

	if len(favoriteIDs) == 0 || l.svcCtx.DB == nil {
		return scores, nil
	}

	query := `
		SELECT c2.id, COUNT(*) as similarity
		FROM content_tags ct1
		JOIN content_tags ct2 ON ct1.tag_id = ct2.tag_id AND ct1.content_id != ct2.content_id
		JOIN content c2 ON ct2.content_id = c2.id
		WHERE ct1.content_id IN (?)
		AND c2.status = 1
		AND c2.vip_level <= ?
	`
	args := []interface{}{favoriteIDs, userVipLevel}

	if len(excludeIDs) > 0 {
		query += " AND c2.id NOT IN (?)"
		args = append(args, excludeIDs)
	}

	query += " GROUP BY c2.id ORDER BY similarity DESC LIMIT 30"

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maxSim := float64(1)
	for rows.Next() {
		var contentID int64
		var sim int64
		if err := rows.Scan(&contentID, &sim); err != nil {
			continue
		}
		scores[contentID] = float64(sim)
		if float64(sim) > maxSim {
			maxSim = float64(sim)
		}
	}

	for id := range scores {
		scores[id] = scores[id] / maxSim * 80
	}

	return scores, nil
}

// hotContentSupplement 热门内容补足
func (l *ContentForYouLogic) hotContentSupplement(userVipLevel int16, excludeIDs1, excludeIDs2 []int64, limit int) (map[int64]float64, error) {
	scores := make(map[int64]float64)

	if l.svcCtx.DB == nil {
		return scores, nil
	}

	allExcludes := append(excludeIDs1, excludeIDs2...)
	uniqueExcludes := uniqueInt64(allExcludes)

	db := l.svcCtx.DB.Model(&dao.ContentCatalog{}).
		Select("c.id").
		Table("content c").
		Where("c.status = ?", 1).
		Where("c.vip_level <= ?", int(userVipLevel))

	if len(uniqueExcludes) > 0 {
		db = db.Where("c.id NOT IN (?)", uniqueExcludes)
	}

	rows, err := db.
		Joins("LEFT JOIN content_play_records pr ON c.id = pr.content_id").
		Order("COALESCE(pr.play_count, 0) DESC").
		Limit(limit).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rank := 0.0
	for rows.Next() {
		var contentID int64
		if err := rows.Scan(&contentID); err != nil {
			continue
		}
		scores[contentID] = 100 - rank
		rank += 10
	}

	return scores, nil
}

// getCandidateContents 获取候选内容详情
func (l *ContentForYouLogic) getCandidateContents(contentIDs []int64, userVipLevel int16) ([]candidateContent, error) {
	if len(contentIDs) == 0 || l.svcCtx.DB == nil {
		return []candidateContent{}, nil
	}

	rows, err := l.svcCtx.DB.Raw(`
		SELECT c.id, c.title, c.cover_url, c.artist, c.duration_sec, c.vip_level,
		COALESCE(array_agg(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tags
		FROM content c
		LEFT JOIN content_tags ct ON c.id = ct.content_id
		LEFT JOIN tags t ON ct.tag_id = t.id
		WHERE c.id IN (?)
		AND c.status = 1
		AND c.vip_level <= ?
		GROUP BY c.id
	`, contentIDs, userVipLevel).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []candidateContent
	for rows.Next() {
		var c candidateContent
		var tags interface{}
		if err := rows.Scan(&c.ID, &c.Title, &c.CoverURL, &c.Artist, &c.Duration, &c.VipLevel, &tags); err != nil {
			continue
		}

		if tagArr, ok := tags.([]string); ok {
			c.Tags = tagArr
		}
		contents = append(contents, c)
	}

	return contents, nil
}

// getRecentPlayHistory 获取用户最近播放历史
func (l *ContentForYouLogic) getRecentPlayHistory(userID int64, limit int) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	rows, err := l.svcCtx.DB.Raw(
		"SELECT DISTINCT content_id FROM user_play_record WHERE user_id = ? ORDER BY played_at DESC LIMIT ?",
		userID, limit,
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

// getFavoriteIDs 获取用户收藏的内容 ID
func (l *ContentForYouLogic) getFavoriteIDs(userID int64) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	rows, err := l.svcCtx.DB.Raw(
		"SELECT content_id FROM user_likes WHERE user_id = ?",
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

// getSubscribedArtistIDs 获取订阅的艺术家 ID
func (l *ContentForYouLogic) getSubscribedArtistIDs(userID int64) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	rows, err := l.svcCtx.DB.Raw(
		"SELECT target_id FROM user_subscriptions WHERE user_id = ? AND subscribe_type = 1",
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
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// getSubscribedSeriesIDs 获取订阅的系列 ID
func (l *ContentForYouLogic) getSubscribedSeriesIDs(userID int64) ([]int64, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	rows, err := l.svcCtx.DB.Raw(
		"SELECT target_id FROM user_subscriptions WHERE user_id = ? AND subscribe_type = 2",
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
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// getTagWeightsFromPlays 从播放历史中提取标签权重
func (l *ContentForYouLogic) getTagWeightsFromPlays(playedIDs []int64) (map[string]int, error) {
	weights := make(map[string]int)

	if len(playedIDs) == 0 || l.svcCtx.DB == nil {
		return weights, nil
	}

	rows, err := l.svcCtx.DB.Raw(`
		SELECT t.name, COUNT(*) as weight
		FROM content_tags ct
		JOIN tags t ON ct.tag_id = t.id
		WHERE ct.content_id IN (?)
		GROUP BY t.name
		ORDER BY weight DESC
	`, playedIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var weight int
		if err := rows.Scan(&name, &weight); err != nil {
			continue
		}
		weights[name] = weight
	}

	return weights, nil
}

// getPreferredArtists 获取偏好的艺术家
func (l *ContentForYouLogic) getPreferredArtists(playedIDs []int64) ([]string, error) {
	if len(playedIDs) == 0 || l.svcCtx.DB == nil {
		return []string{}, nil
	}

	rows, err := l.svcCtx.DB.Raw(`
		SELECT artist FROM content WHERE id IN (?) AND artist != '' GROUP BY artist ORDER BY COUNT(*) DESC LIMIT 10
	`, playedIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []string
	for rows.Next() {
		var artist string
		if err := rows.Scan(&artist); err != nil {
			continue
		}
		artists = append(artists, artist)
	}
	return artists, nil
}

// getUserVipLevel 获取用户会员等级
func (l *ContentForYouLogic) getUserVipLevel(userID int64) (int16, error) {
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

// 辅助函数
func parseExcludeIDs(excludeStr string) []int64 {
	if excludeStr == "" {
		return []int64{}
	}

	parts := strings.Split(excludeStr, ",")
	var ids []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func containsInt64(slice []int64, val int64) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func uniqueInt64(slice []int64) []int64 {
	seen := make(map[int64]bool)
	result := []int64{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func getMapKeys(m map[int64]float64) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getContentIDs(items []forYouScoredItem) []int64 {
	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ContentID
	}
	return ids
}

func mergeScoreMaps(maps ...map[int64]float64) map[int64]*scoreComponents {
	merged := make(map[int64]*scoreComponents)
	for _, m := range maps {
		for id, score := range m {
			if _, ok := merged[id]; !ok {
				merged[id] = &scoreComponents{}
			}
			switch len(maps) {
			case 1:
				merged[id].HotScore = score
			case 2:
				merged[id].FavScore = score
			case 3:
				merged[id].TagScore = score
			case 4:
				merged[id].CFScore = score
			}
		}
	}
	return merged
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
