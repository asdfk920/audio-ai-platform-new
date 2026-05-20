package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContentListLogic 内容列表逻辑
type ContentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentListLogic 创建内容列表逻辑实例
func NewContentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentListLogic {
	return &ContentListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ContentList 获取内容列表（完整版）
// 流程：
//  1. 接收并校验分页参数合法性
//  2. 从用户Token获取会员等级
//  3. 构建查询条件：已上架 + 符合权限 + 分类 + 标签 + 标题/关键词
//  4. 执行COUNT获取总数
//  5. 执行分页SELECT获取列表数据
//  6. 对每条内容做权限校验，组装完整字段
//  7. 封装统一返回格式：list + total + page + page_size
//  8. 空列表正常返回，不报错
func (l *ContentListLogic) ContentList(req *types.ContentListReq, userID int64) (*types.ContentListResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Content List] 开始处理内容列表请求...")
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("   Page:       %d", req.Page)
	logx.Infof("   PageSize:   %d", req.PageSize)
	if req.CategoryID > 0 {
		logx.Infof("   CategoryID: %d", req.CategoryID)
	}
	if req.TagIDs != "" {
		logx.Infof("   TagIDs:     %s", req.TagIDs)
	}
	if req.Title != "" {
		logx.Infof("   Title:      %s", req.Title)
	}
	if req.Keyword != "" {
		logx.Infof("   Keyword:    %s", req.Keyword)
	}
	logx.Infof("====================================")

	// ========== Step 1: 参数校验 ==========
	page, pageSize := l.validateParams(req)

	// ========== Step 2: 获取用户会员等级 ==========
	userVipLevel := int16(0)
	if userID > 0 {
		vipLevel, err := l.getUserVipLevel(userID)
		if err != nil {
			logx.Errorf("[Content List] 获取会员等级失败: user_id=%d, err=%v", userID, err)
		} else {
			userVipLevel = vipLevel
			logx.Infof("[Content List] 用户会员等级: %d", vipLevel)
		}
	} else {
		logx.Infof("[Content List] 游客模式（无Token）")
	}

	// ========== Step 3: 构建查询条件 ==========
	whereClause, args := l.buildWhereClause(req, userVipLevel)

	// ========== Step 4: 查询总数 ==========
	total, err := l.countTotal(whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("查询总数失败: %v", err)
	}
	logx.Infof("[Content List] 总记录数: %d", total)

	// ========== Step 5: 分页查询 ==========
	list, err := l.queryPageData(whereClause, args, page, pageSize, req.Sort, userVipLevel)
	if err != nil {
		return nil, fmt.Errorf("查询列表失败: %v", err)
	}

	// ========== Step 6: 计算分页信息 ==========
	offset := (page - 1) * pageSize
	hasMore := offset+len(list) < int(total)
	totalPages := (int(total) + pageSize - 1) / pageSize

	logx.Infof("\n====================================")
	logx.Infof("[Content List] ✅ 列表查询成功!")
	logx.Infof("====================================")
	logx.Infof("  当前页码:   %d / %d", page, totalPages)
	logx.Infof("  每页数量:   %d", pageSize)
	logx.Infof("  总记录数:   %d", total)
	logx.Infof("  本页条数:   %d", len(list))
	logx.Infof("  是否有更多: %v", hasMore)
	logx.Infof("====================================")

	// ========== Step 7: 组装响应 ==========
	resp := &types.ContentListResp{
		Total:      total,
		List:       list,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    hasMore,
		TotalPages: totalPages,
	}

	return resp, nil
}

// validateParams 校验参数合法性并设置默认值
func (l *ContentListLogic) validateParams(req *types.ContentListReq) (page int, pageSize int) {
	page = int(req.Page)
	pageSize = int(req.PageSize)

	// 页码校验
	if page <= 0 {
		page = 1
	}
	if page > 10000 {
		page = 10000 // 防止恶意请求
	}

	// 每页数量校验
	if pageSize <= 0 {
		pageSize = 20 // 默认20条
	}
	if pageSize > 100 {
		pageSize = 100 // 最大100条
	}

	logx.Infof("[Content List] 参数校验后: page=%d, page_size=%d", page, pageSize)
	return page, pageSize
}

// getUserVipLevel 获取用户会员等级
func (l *ContentListLogic) getUserVipLevel(userID int64) (int16, error) {
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

// buildWhereClause 构建SQL WHERE条件
// 核心规则：
//  - 只查已上架（status=1）+ 未删除（is_deleted=0）
//  - 权限过滤：vip_level <= 用户等级（或全部显示但标记权限）
//  - 支持分类、标签、标题模糊、关键词（标题或艺术家）、VIP专享过滤
func (l *ContentListLogic) buildWhereClause(req *types.ContentListReq, userVipLevel int16) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// 基础条件：未删除 + 已上架
	conditions = append(conditions, "c.is_deleted = 0")
	conditions = append(conditions, "c.status = 1")

	// 权限过滤：根据需求可选择严格过滤或宽松过滤
	// 方案A（当前）：宽松过滤 - 显示所有内容，前端根据权限字段判断是否可播放
	// 如需严格过滤（只返回有权限的），取消下面注释：
	// conditions = append(conditions, fmt.Sprintf("c.vip_level <= $%d", argIndex))
	// args = append(args, userVipLevel)
	// argIndex++

	// 分类过滤
	if req.CategoryID > 0 {
		conditions = append(conditions, fmt.Sprintf("c.category_id = $%d", argIndex))
		args = append(args, req.CategoryID)
		argIndex++
		logx.Infof("[Content List] 过滤条件: category_id=%d", req.CategoryID)
	}

	// 标签过滤（使用子查询）
	if req.TagIDs != "" {
		tagIDs := parseTagIDsToInt(req.TagIDs)
		if len(tagIDs) > 0 {
			placeholders := make([]string, len(tagIDs))
			for i, tid := range tagIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, tid)
				argIndex++
			}
			tagCondition := fmt.Sprintf(`c.id IN (
				SELECT ct.content_id FROM content_tag ct 
				WHERE ct.tag_id IN (%s)
				GROUP BY ct.content_id 
				HAVING COUNT(DISTINCT ct.tag_id) >= 1
			)`, strings.Join(placeholders, ","))
			conditions = append(conditions, tagCondition)
			logx.Infof("[Content List] 过滤条件: tag_ids=%s (%d个标签)", req.TagIDs, len(tagIDs))
		}
	}

	// 标题模糊（仅 title）
	if req.Title != "" {
		title := strings.TrimSpace(req.Title)
		if title != "" {
			conditions = append(conditions, fmt.Sprintf("c.title ILIKE $%d", argIndex))
			args = append(args, "%"+title+"%")
			argIndex++
			logx.Infof("[Content List] 过滤条件: title ILIKE '%%%s%%'", title)
		}
	}

	// 关键词搜索（标题/艺术家模糊匹配）
	if req.Keyword != "" {
		keyword := strings.TrimSpace(req.Keyword)
		if keyword != "" {
			conditions = append(conditions, fmt.Sprintf("(c.title ILIKE $%d OR c.artist ILIKE $%d)", argIndex, argIndex))
			args = append(args, "%"+keyword+"%")
			argIndex++
			logx.Infof("[Content List] 过滤条件: keyword='%s'", keyword)
		}
	}

	// 仅VIP专享
	if req.IsVip == 1 {
		conditions = append(conditions, "c.vip_level > 0")
		logx.Infof("[Content List] 过滤条件: is_vip=1（仅VIP）")
	}

	whereClause := strings.Join(conditions, " AND ")
	logx.Infof("[Content List] WHERE条件构建完成: %d个条件", len(conditions))

	return whereClause, args
}

// countTotal 查询符合条件的总记录数
func (l *ContentListLogic) countTotal(whereClause string, args []interface{}) (int64, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM content c WHERE %s", whereClause)

	var total int64
	err := l.svcCtx.DB.Raw(query, args...).Scan(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

// queryPageData 分页查询内容列表数据
func (l *ContentListLogic) queryPageData(whereClause string, args []interface{}, page, pageSize int, sort int32, userVipLevel int16) ([]types.ContentListItem, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	// 构造排序规则
	orderBy := l.buildOrderBy(sort)

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 主查询SQL（对可空列 COALESCE，避免 Scan 进非指针 Go 字段失败却被 continue 丢掉，造成 total 与 list 不一致）
	query := fmt.Sprintf(`
		SELECT 
			c.id,
			COALESCE(c.title, ''),
			COALESCE(c.cover_url, ''),
			COALESCE(c.artist, ''),
			COALESCE(c.duration_sec, 0),
			COALESCE(c.format, ''),
			COALESCE(c.size_bytes, 0),
			COALESCE(c.category_id, 0),
			COALESCE(c.vip_level, 0)::smallint,
			COALESCE(c.play_count, 0),
			c.created_at
		FROM content c
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)+1, len(args)+2)

	args = append(args, pageSize, offset)

	rows, err := l.svcCtx.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []types.ContentListItem

	for rows.Next() {
		var item contentListItemDB
		var createdAt time.Time

		err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.CoverURL,
			&item.Artist,
			&item.DurationSec,
			&item.Format,
			&item.SizeBytes,
			&item.CategoryID,
			&item.VipLevel,
			&item.PlayCount,
			&createdAt,
		)
		if err != nil {
			logx.Errorf("[Content List] 扫描行失败: %v", err)
			return nil, fmt.Errorf("扫描列表行失败: %w", err)
		}

		// 转换为API响应格式
		apiItem := l.convertToAPIItem(&item, createdAt, userVipLevel)
		list = append(list, apiItem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历列表结果失败: %w", err)
	}

	// 空列表正常返回（不报错）
	if list == nil {
		list = []types.ContentListItem{}
		logx.Infof("[Content List] 当前页无数据，返回空列表")
	}

	return list, nil
}

// buildOrderBy 构建排序规则
func (l *ContentListLogic) buildOrderBy(sort int32) string {
	switch sort {
	case 1:
		return "c.created_at DESC"           // 最新发布
	case 2:
		return "COALESCE(c.play_count, 0) DESC, c.created_at DESC"  // 最热（播放量）
	case 3:
		return "c.sort_order DESC, c.created_at DESC"  // 推荐
	default:
		return "c.sort_order DESC, c.created_at DESC"  // 综合（默认）
	}
}

// convertToAPIItem 将数据库记录转换为API响应格式
// 包含：基础信息 + 分类名称 + 标签 + 统计数据 + 权限信息
func (l *ContentListLogic) convertToAPIItem(dbItem *contentListItemDB, createdAt time.Time, userVipLevel int16) types.ContentListItem {
	item := types.ContentListItem{
		ContentID: dbItem.ID,
		Title:     dbItem.Title,
		CoverURL:  dbItem.CoverURL,
		Artist:    dbItem.Artist,
		Duration:  dbItem.DurationSec,
		Format:    dbItem.Format,
		FileSize:  dbItem.SizeBytes,
		ViewCount: dbItem.PlayCount,
		PublishedAt: createdAt.Format("2006-01-02 15:04:05"),
	}

	// 查询分类名称
	if dbItem.CategoryID > 0 {
		categoryName, _ := l.queryCategoryName(dbItem.CategoryID)
		item.Category = categoryName
	}

	// 查询标签
	tags, _ := l.queryContentTags(dbItem.ID)
	item.Tags = tags

	// 查询点赞数
	likeCount, _ := l.queryLikeCount(dbItem.ID)
	item.LikeCount = likeCount

	// 权限计算
	canPlay := userVipLevel >= dbItem.VipLevel
	item.Permission = types.ListItemPermission{
		NeedVIP:       dbItem.VipLevel > 0,
		RequiredLevel: dbItem.VipLevel,
		CanPlay:       canPlay,
	}

	return item
}

// contentListItemDB 数据库查询结果结构体
type contentListItemDB struct {
	ID          int64
	Title       string
	CoverURL    string
	Artist      string
	DurationSec int
	Format      string
	SizeBytes   int64
	CategoryID  int64
	VipLevel    int16
	PlayCount   int64
}

// queryCategoryName 查询分类名称
func (l *ContentListLogic) queryCategoryName(categoryID int64) (string, error) {
	if l.svcCtx.DB == nil || categoryID <= 0 {
		return "", fmt.Errorf("无效参数")
	}

	query := `SELECT name FROM content_category WHERE id = $1 AND status = 1 LIMIT 1`
	var name string
	err := l.svcCtx.DB.Raw(query, categoryID).Scan(&name).Error
	return name, err
}

// queryContentTags 查询内容标签
func (l *ContentListLogic) queryContentTags(contentID int64) ([]string, error) {
	if l.svcCtx.DB == nil || contentID <= 0 {
		return []string{}, nil
	}

	query := `
		SELECT t.name 
		FROM tag t
		INNER JOIN content_tag ct ON t.id = ct.tag_id
		WHERE ct.content_id = $1 AND t.status = 1
		ORDER BY t.name ASC
		LIMIT 10
	`

	var tags []string
	err := l.svcCtx.DB.Raw(query, contentID).Pluck("name", &tags).Error
	if err != nil {
		return []string{}, nil // 容错：表不存在时返回空数组
	}

	return tags, nil
}

// queryLikeCount 查询点赞数
func (l *ContentListLogic) queryLikeCount(contentID int64) (int64, error) {
	if l.svcCtx.DB == nil || contentID <= 0 {
		return 0, nil
	}

	query := `SELECT COUNT(*) FROM user_likes WHERE content_id = $1`
	var count int64
	err := l.svcCtx.DB.Raw(query, contentID).Scan(&count).Error
	if err != nil {
		return 0, nil // 容错
	}
	return count, nil
}

// parseTagIDsToInt 解析标签ID字符串为整数切片
func parseTagIDsToInt(tagIDsStr string) []int64 {
	if tagIDsStr == "" {
		return nil
	}

	parts := strings.Split(tagIDsStr, ",")
	var ids []int64
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := parseInt64(part)
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// parseInt64 安全解析整数
func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}