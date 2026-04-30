package logic

import (
	"context"
	"fmt"
	"strconv"
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

// ContentList 获取内容列表
// 流程：
//  1. 解析请求参数
//  2. 获取用户会员等级
//  3. 构造 SQL 查询条件
//  4. 执行 COUNT 查询获取总数
//  5. 执行 SELECT 查询获取内容列表
//  6. 计算分页信息
//  7. 组装响应返回
func (l *ContentListLogic) ContentList(req *types.ContentListReq, userID int64) (*types.ContentListResp, error) {
	// 1. 解析请求参数
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
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

	// 3. 构造 SQL 查询条件
	whereClause, args := l.buildWhereClause(req, userVipLevel)

	// 4. 执行 COUNT 查询获取总数
	total, err := l.countContent(whereClause, args)
	if err != nil {
		return nil, fmt.Errorf("查询内容总数失败: %v", err)
	}

	// 5. 执行 SELECT 查询获取内容列表
	list, err := l.queryContentList(whereClause, args, page, pageSize, req.Sort)
	if err != nil {
		return nil, fmt.Errorf("查询内容列表失败: %v", err)
	}

	// 6. 计算分页信息
	offset := (page - 1) * pageSize
	hasMore := offset+len(list) < int(total)

	// 7. 组装响应返回
	return &types.ContentListResp{
		Total:    total,
		List:     list,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	}, nil
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

// buildWhereClause 构造 SQL 查询条件
func (l *ContentListLogic) buildWhereClause(req *types.ContentListReq, userVipLevel int16) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// 基础条件：未删除、已上架
	conditions = append(conditions, "is_deleted = 0")
	conditions = append(conditions, "status = 1")

	// 会员等级过滤：用户只能看到 vip_level <= user_vip_level 的内容
	conditions = append(conditions, fmt.Sprintf("vip_level <= $%d", argIndex))
	args = append(args, userVipLevel)
	argIndex++

	// 分类过滤
	if req.CategoryID > 0 {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, req.CategoryID)
		argIndex++
	}

	// 标签过滤
	if req.TagIDs != "" {
		tagIDs := strings.Split(req.TagIDs, ",")
		var validTagIDs []string
		for _, tid := range tagIDs {
			tid = strings.TrimSpace(tid)
			if tid != "" {
				validTagIDs = append(validTagIDs, tid)
			}
		}
		if len(validTagIDs) > 0 {
			// 使用子查询过滤有指定标签的内容
			conditions = append(conditions, fmt.Sprintf(`id IN (
				SELECT content_id FROM content_tag_relation 
				WHERE tag_id IN (%s)
			)`, strings.Join(validTagIDs, ",")))
		}
	}

	// 关键词搜索
	if req.Keyword != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR artist ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+req.Keyword+"%")
		argIndex++
	}

	// 仅会员专享
	if req.IsVip == 1 {
		conditions = append(conditions, "vip_level > 0")
	}

	whereClause := strings.Join(conditions, " AND ")
	return whereClause, args
}

// countContent 查询内容总数
func (l *ContentListLogic) countContent(whereClause string, args []interface{}) (int64, error) {
	if l.svcCtx.DB == nil {
		return 0, fmt.Errorf("数据库未就绪")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM content WHERE %s", whereClause)

	var total int64
	err := l.svcCtx.DB.Raw(query, args...).Scan(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

// queryContentList 查询内容列表
func (l *ContentListLogic) queryContentList(whereClause string, args []interface{}, page, pageSize int, sort int32) ([]types.ContentListItem, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	// 构造排序条件
	orderBy := "sort_order DESC, created_at DESC"
	switch sort {
	case 1:
		orderBy = "sort_order DESC"
	case 2:
		orderBy = "created_at DESC"
	case 3:
		orderBy = "sort_order DESC"
	case 4:
		orderBy = "updated_at DESC"
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, title, cover_url, artist, duration_sec, vip_level, 
			   format, sort_order, created_at
		FROM content 
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
		var item types.ContentListItem
		var createdAt time.Time
		err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.CoverURL,
			&item.Artist,
			&item.Duration,
			&item.VipLevel,
			&item.Format,
			&item.Sort,
			&createdAt,
		)
		if err != nil {
			logx.Errorf("扫描内容列表行失败: %v", err)
			continue
		}

		item.IsVipContent = item.VipLevel > 0
		item.CanPlay = true
		item.PublishedAt = createdAt.Format("2006-01-02T15:04:05Z")

		list = append(list, item)
	}

	return list, nil
}

// parseTagIDs 解析标签 ID 字符串
func parseTagIDs(tagIDsStr string) []int64 {
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
		id, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
