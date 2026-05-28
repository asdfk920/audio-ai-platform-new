package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// MessageLogic 站内消息通知业务逻辑
type MessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewMessageLogic 创建消息逻辑实例
func NewMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MessageLogic {
	return &MessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetMessageList 获取用户消息列表
// 完整流程：
// 1. 校验用户是否登录
// 2. 查询消息列表（支持分页、类型筛选、已读/未读筛选）
// 3. 统计未读消息数量
// 4. 返回消息列表和分页信息
func (l *MessageLogic) GetMessageList(userID int64, req *types.MessageListReq) (*types.MessageListResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Message] 开始查询消息列表...")
	logx.Infof("   User ID: %d", userID)
	logx.Infof("   Page:     %d", req.Page)
	logx.Infof("   PageSize: %d", req.PageSize)
	logx.Infof("====================================")

	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := l.svcCtx.DB.Table("user_messages").
		Where("user_id = ?", userID)

	if req.MessageType != "" {
		query = query.Where("message_type = ?", req.MessageType)
	}
	if req.IsRead != nil {
		query = query.Where("is_read = ?", *req.IsRead)
	}

	var total int64
	countQuery := l.svcCtx.DB.Table("user_messages").
		Where("user_id = ?", userID)
	if req.MessageType != "" {
		countQuery = countQuery.Where("message_type = ?", req.MessageType)
	}
	if req.IsRead != nil {
		countQuery = countQuery.Where("is_read = ?", *req.IsRead)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		logx.Errorf("[Message] 查询消息总数失败: error=%v", err)
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	offset := (req.Page - 1) * req.PageSize

	var items []struct {
		ID          int64
		MessageType string
		Title       string
		Content     string
		RelatedType string
		RelatedID   int64
		ActionURL   string
		IsRead      int16
		ReadAt      *time.Time
		CreatedAt   time.Time
	}

	result := query.
		Select("id, message_type, title, content, related_type, related_id, action_url, is_read, read_at, created_at").
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(offset)).
		Find(&items)

	if result.Error != nil {
		logx.Errorf("[Message] 查询消息列表失败: error=%v", result.Error)
		return nil, fmt.Errorf("查询失败: %v", result.Error)
	}

	list := make([]types.MessageListItem, 0, len(items))
	for _, item := range items {
		readAt := ""
		if item.ReadAt != nil {
			readAt = item.ReadAt.Format("2006-01-02 15:04:05")
		}
		list = append(list, types.MessageListItem{
			ID:          item.ID,
			MessageType: item.MessageType,
			Title:       item.Title,
			Content:     item.Content,
			RelatedType: item.RelatedType,
			RelatedID:   item.RelatedID,
			ActionURL:   item.ActionURL,
			IsRead:      item.IsRead == 1,
			ReadAt:      readAt,
			CreatedAt:   item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	unreadCount, _ := l.GetUnreadCount(userID)

	totalPages := int32(total / int64(req.PageSize))
	if total%int64(req.PageSize) > 0 {
		totalPages++
	}

	resp := &types.MessageListResp{
		Total:       total,
		UnreadCount: unreadCount,
		List:        list,
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalPages:  totalPages,
	}

	logx.Infof("[Message] ✅ 查询完成: total=%d unreadCount=%d page=%d",
		total, unreadCount, req.Page)

	return resp, nil
}

// MarkMessageRead 标记消息为已读
// 支持两种模式：
// 1. 批量标记：传入具体的 message_ids 列表
// 2. 全部标记：不传或传空数组，则标记该用户所有未读消息为已读
func (l *MessageLogic) MarkMessageRead(userID int64, messageIDs []int64) (*types.MarkMessageReadResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Message] 开始标记消息已读...")
	logx.Infof("   User ID:    %d", userID)
	logx.Infof("   MessageIDs: %v", messageIDs)
	logx.Infof("====================================")

	if userID <= 0 {
		return nil, fmt.Errorf("用户未登录")
	}

	now := time.Now()
	var affectedRows int64

	query := l.svcCtx.DB.Table("user_messages").
		Where("user_id = ? AND is_read = 0", userID)

	if len(messageIDs) > 0 {
		query = query.Where("id IN (?)", messageIDs)
	}

	result := query.Updates(map[string]interface{}{
		"is_read": 1,
		"read_at": now,
	})
	if result.Error != nil {
		logx.Errorf("[Message] 标记已读失败: error=%v", result.Error)
		return nil, fmt.Errorf("操作失败: %v", result.Error)
	}

	affectedRows = result.RowsAffected

	unreadCount, _ := l.GetUnreadCount(userID)

	resp := &types.MarkMessageReadResp{
		Success:      true,
		Message:      fmt.Sprintf("成功标记 %d 条消息为已读", affectedRows),
		AffectedRows: affectedRows,
		UnreadCount:  unreadCount,
	}

	logx.Infof("[Message] ✅ 标记已读完成: affectedRows=%d remainingUnread=%d",
		affectedRows, unreadCount)

	return resp, nil
}

// GetUnreadCount 获取用户未读消息数量
func (l *MessageLogic) GetUnreadCount(userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}

	var count int64
	err := l.svcCtx.DB.Table("user_messages").
		Where("user_id = ? AND is_read = 0", userID).
		Count(&count).Error

	if err != nil {
		logx.Errorf("[Message] 查询未读数失败: error=%v", err)
		return 0, err
	}

	return count, nil
}

// CreateMessage 创建新消息（内部调用）
// 用于业务系统生成通知：
// - 用户订阅艺术家后，艺术家发布新内容时调用
// - 系统公告推送时调用
// - 其他需要通知用户的场景
func (l *MessageLogic) CreateMessage(userID int64, messageType, title, content, relatedType string, relatedID int64, actionURL string) error {
	if userID <= 0 {
		return fmt.Errorf("用户ID无效")
	}

	msg := map[string]interface{}{
		"user_id":      userID,
		"message_type": messageType,
		"title":        title,
		"content":      content,
		"related_type": relatedType,
		"related_id":   relatedID,
		"action_url":   actionURL,
		"is_read":      0,
		"created_at":   time.Now(),
	}

	err := l.svcCtx.DB.Table("user_messages").Create(msg).Error
	if err != nil {
		logx.Errorf("[Message] 创建消息失败: error=%v", err)
		return fmt.Errorf("创建消息失败: %v", err)
	}

	logx.Infof("[Message] ✅ 消息创建成功: userID=%d type=%s title=%s",
		userID, messageType, title)

	return nil
}

// NotifySubscribersAboutNewContent 当艺术家发布新内容时，通知所有订阅者
// 这是核心触发函数，在内容发布时调用
func (l *MessageLogic) NotifySubscribersAboutNewContent(artistID int64, artistName string, contentType string, contentID int64, contentTitle string) error {
	logx.Infof("====================================")
	logx.Infof("[Message] 开始通知订阅者...")
	logx.Infof("   Artist ID:    %d", artistID)
	logx.Infof("   Artist Name:  %s", artistName)
	logx.Infof("   Content Type: %s", contentType)
	logx.Infof("   Content ID:   %d", contentID)
	logx.Infof("   Content Title:%s", contentTitle)
	logx.Infof("====================================")

	if artistID <= 0 || contentID <= 0 {
		return fmt.Errorf("参数无效")
	}

	var subscribers []struct {
		UserID int64
	}

	err := l.svcCtx.DB.Table("artist_subscriptions").
		Select("user_id").
		Where("artist_id = ? AND status = 1", artistID).
		Find(&subscribers).Error

	if err != nil {
		logx.Errorf("[Message] 查询订阅者失败: error=%v", err)
		return fmt.Errorf("查询订阅者失败: %v", err)
	}

	if len(subscribers) == 0 {
		logx.Infof("[Message] 该艺术家暂无订阅者，跳过通知")
		return nil
	}

	contentTypeCN := ""
	switch contentType {
	case "song":
		contentTypeCN = "歌曲"
	case "album":
		contentTypeCN = "专辑"
	default:
		contentTypeCN = "内容"
	}

	title := fmt.Sprintf("%s发布了新%s《%s》", artistName, contentTypeCN, contentTitle)
	actionURL := fmt.Sprintf("/content/%d", contentID)

	successCount := 0
	failCount := 0

	for _, sub := range subscribers {
		err := l.CreateMessage(
			sub.UserID,
			"new_content",
			title,
			fmt.Sprintf("您关注的艺术家%s发布了新的%s《%s》，快来收听吧！", artistName, contentTypeCN, contentTitle),
			contentType,
			contentID,
			actionURL,
		)
		if err != nil {
			logx.Errorf("[Message] 通知用户%d失败: error=%v", sub.UserID, err)
			failCount++
		} else {
			successCount++
		}
	}

	logx.Infof("[Message] ✅ 通知完成: total=%d success=%d failed=%d",
		len(subscribers), successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("部分通知发送失败: 成功%d 失败%d", successCount, failCount)
	}

	return nil
}
