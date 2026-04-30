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

// ContentDownloadLogic 下载逻辑
type ContentDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewContentDownloadLogic 创建下载逻辑实例
func NewContentDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentDownloadLogic {
	return &ContentDownloadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Download 处理内容下载请求
func (l *ContentDownloadLogic) Download(req *types.DownloadReq, userID int64, clientIP, userAgent string) (*types.DownloadResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	content, err := l.getContentByID(req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("内容不存在或已下架")
	}

	userVipLevel := int16(0)
	vipLevel, err := l.getUserVipLevel(userID)
	if err != nil {
		logx.Errorf("获取用户会员等级失败: %v", err)
	} else {
		userVipLevel = vipLevel
	}

	if content.VipLevel > userVipLevel {
		return nil, fmt.Errorf("该内容为VIP专属，请先升级会员")
	}

	now := time.Now()

	existingRecord, err := l.checkExistingDownload(userID, req.ContentID)
	isFirst := existingRecord.ID == 0 || existingRecord.Status != 1

	if existingRecord.ID > 0 && existingRecord.Status == 1 {
		if updateErr := l.updateExistingDownload(existingRecord.ID, now, clientIP, userAgent, 0); updateErr != nil {
			logx.Errorf("更新下载记录失败: %v", updateErr)
		}
	} else {
		if insertErr := l.insertDownloadRecord(userID, req.ContentID, content.Title, content.AudioURL, now, clientIP, userAgent, 0); insertErr != nil {
			logx.Errorf("创建下载记录失败: %v", insertErr)
			return nil, fmt.Errorf("下载记录创建失败")
		}
	}

	if isFirst {
		l.recordFirstDownload(userID, req.ContentID)
	}

	l.updateDownloadStats(req.ContentID, now)

	totalDownloads := l.getContentDownloadCount(req.ContentID)

	fileName := fmt.Sprintf("%s.mp3", content.Title)
	contentType := l.getContentType(content.AudioURL)
	expiresAt := now.Add(24 * time.Hour)

	return &types.DownloadResp{
		ID:             req.ContentID,
		DownloadURL:    content.AudioURL,
		FileName:       fileName,
		FileSize:       0,
		DownloadTime:   now.Format("2006-01-02 15:04:05"),
		IsFirst:        isFirst,
		ContentType:    contentType,
		ExpiresAt:      expiresAt.Format("2006-01-02 15:04:05"),
		TotalDownloads: totalDownloads,
	}, nil
}

// GetDownloadList 获取用户下载历史列表
func (l *ContentDownloadLogic) GetDownloadList(req *types.DownloadListReq, userID int64) (*types.DownloadListResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	page := int(req.Page)
	size := int(req.Size)
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}

	offset := (page - 1) * size

	var total int64
	countErr := l.svcCtx.DB.Table("user_downloads").
		Where("user_id = ?", userID).
		Count(&total).Error
	if countErr != nil {
		total = 0
	}

	totalPages := int32(0)
	if total > 0 {
		totalPages = int32((total + int64(size) - 1) / int64(size))
	}

	rows, err := l.svcCtx.DB.Raw(`
		SELECT ud.id, ud.content_id, COALESCE(ud.content_title, ''),
		COALESCE(c.cover_url, ''), ud.download_time, ud.status, ud.file_size,
		CASE WHEN c.status = 1 AND c.is_deleted = 0 THEN false ELSE true END as is_offline
		FROM user_downloads ud
		LEFT JOIN content c ON ud.content_id = c.id
		WHERE ud.user_id = ?
		ORDER BY ud.download_time DESC
		LIMIT ? OFFSET ?
	`, userID, size, offset).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}
	defer rows.Close()

	var list []types.DownloadItem
	for rows.Next() {
		var item types.DownloadItem
		var downloadTime time.Time
		var fileSizeBytes int64
		var statusInt int16
		if err := rows.Scan(
			&item.RecordID, &item.ContentID, &item.ContentName,
			&item.CoverURL, &downloadTime, &statusInt, &fileSizeBytes, &item.IsOffline,
		); err != nil {
			continue
		}
		item.DownloadTime = downloadTime.Format("2006-01-02 15:04:05")
		item.Status = l.formatStatus(statusInt)
		item.FileSize = l.formatFileSize(fileSizeBytes)
		list = append(list, item)
	}

	return &types.DownloadListResp{
		Total:      total,
		Page:       int32(page),
		PageSize:   int32(size),
		TotalPages: totalPages,
		List:       list,
	}, nil
}

// formatStatus 格式化状态
func (l *ContentDownloadLogic) formatStatus(status int16) string {
	switch status {
	case 0:
		return "待下载"
	case 1:
		return "下载中"
	case 2:
		return "下载中"
	case 3:
		return "已下载"
	default:
		return "未知"
	}
}

// formatFileSize 格式化文件大小
func (l *ContentDownloadLogic) formatFileSize(bytes int64) string {
	if bytes <= 0 {
		return "0B"
	}
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	if bytes >= GB {
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	}
	if bytes >= KB {
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%dB", bytes)
}

// getContentByID 根据ID获取内容信息
func (l *ContentDownloadLogic) getContentByID(contentID int64) (*dao.ContentCatalog, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	var content dao.ContentCatalog
	err := l.svcCtx.DB.Where("id = ? AND status = ? AND is_deleted = ?", contentID, 1, 0).First(&content).Error
	if err != nil {
		return nil, fmt.Errorf("内容不存在")
	}
	return &content, nil
}

// checkExistingDownload 检查是否已有下载记录
func (l *ContentDownloadLogic) checkExistingDownload(userID, contentID int64) (*downloadRecord, error) {
	if l.svcCtx.DB == nil {
		return &downloadRecord{}, nil
	}

	var record downloadRecord
	err := l.svcCtx.DB.Table("user_downloads").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Order("id DESC").
		Limit(1).
		Find(&record).Error
	if err != nil || record.ID == 0 {
		return &downloadRecord{}, nil
	}
	return &record, nil
}

// insertDownloadRecord 插入下载记录
func (l *ContentDownloadLogic) insertDownloadRecord(userID, contentID int64, title, fileURL string, downloadTime time.Time, ip, device string, fileSize int64) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	record := map[string]interface{}{
		"user_id":       userID,
		"content_id":    contentID,
		"content_title": title,
		"file_url":      fileURL,
		"download_time": downloadTime,
		"status":        1,
		"ip_address":    ip,
		"device_info":   device,
		"file_size":     fileSize,
		"created_at":    downloadTime,
	}

	result := l.svcCtx.DB.Table("user_downloads").Create(record)
	return result.Error
}

// updateExistingDownload 更新已有下载记录
func (l *ContentDownloadLogic) updateExistingDownload(downloadID int64, downloadTime time.Time, ip, device string, fileSize int64) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	updates := map[string]interface{}{
		"download_time": downloadTime,
		"status":        1,
		"ip_address":    ip,
		"device_info":   device,
		"file_size":     fileSize,
	}

	return l.svcCtx.DB.Table("user_downloads").Where("id = ?", downloadID).Updates(updates).Error
}

// DeleteDownloadRecords 删除下载记录（支持单条和批量删除）
// 删除后自动加入同步队列，等待用户确认后删除云端记录
func (l *ContentDownloadLogic) DeleteDownloadRecords(req *types.DownloadDeleteReq, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("请先登录")
	}

	if len(req.RecordIDs) == 0 {
		return fmt.Errorf("请选择要删除的记录")
	}

	if len(req.RecordIDs) > 100 {
		return fmt.Errorf("单次最多删除100条记录")
	}

	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	for _, recordID := range req.RecordIDs {
		if recordID <= 0 {
			return fmt.Errorf("记录 ID 格式无效")
		}
	}

	var count int64
	err := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 0", req.RecordIDs, userID).
		Count(&count).Error
	if err != nil {
		logx.Errorf("查询下载记录失败: %v", err)
		return fmt.Errorf("删除失败")
	}

	if count == 0 {
		return fmt.Errorf("删除失败，记录不存在或已处于同步状态")
	}

	if count != int64(len(req.RecordIDs)) {
		logx.Infof("部分记录不存在或已处于同步状态: 请求%d条，验证通过%d条", len(req.RecordIDs), count)
	}

	result := l.svcCtx.DB.Table("user_downloads").
		Where("id IN ? AND user_id = ? AND sync_status = 0", req.RecordIDs, userID).
		Update("sync_status", 1)
	if result.Error != nil {
		logx.Errorf("更新同步状态失败: %v", result.Error)
		return fmt.Errorf("删除失败")
	}

	logx.Infof("删除下载记录成功，已加入同步队列: userID=%d, recordIDs=%v, count=%d", userID, req.RecordIDs, result.RowsAffected)

	return nil
}

// getUserVipLevel 获取用户VIP等级
func (l *ContentDownloadLogic) getUserVipLevel(userID int64) (int16, error) {
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

// recordFirstDownload 记录首次下载业务逻辑
func (l *ContentDownloadLogic) recordFirstDownload(userID, contentID int64) {
	logx.Infof("首次下载: userID=%d, contentID=%d", userID, contentID)
}

// UpdateDownloadStatus 更新下载状态
func (l *ContentDownloadLogic) UpdateDownloadStatus(recordID int64, status int16, fileSize int64) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	updates := map[string]interface{}{
		"status":    status,
		"file_size": fileSize,
	}

	if status == 1 {
		updates["download_time"] = time.Now()
	}

	return l.svcCtx.DB.Table("user_downloads").Where("id = ?", recordID).Updates(updates).Error
}

// updateDownloadStats 更新下载统计
func (l *ContentDownloadLogic) updateDownloadStats(contentID int64, downloadTime time.Time) {
	if l.svcCtx.DB == nil {
		return
	}

	l.svcCtx.DB.Exec(`
		INSERT INTO content_download_stats (content_id, total_downloads, today_downloads, week_downloads, last_download_time, updated_at)
		VALUES (?, 1, 1, 1, ?, NOW())
		ON CONFLICT (content_id) DO UPDATE SET
			total_downloads = content_download_stats.total_downloads + 1,
			today_downloads = content_download_stats.today_downloads + 1,
			week_downloads = content_download_stats.week_downloads + 1,
			last_download_time = EXCLUDED.last_download_time,
			updated_at = NOW()
	`, contentID, downloadTime)
}

// getContentDownloadCount 获取内容总下载次数
func (l *ContentDownloadLogic) getContentDownloadCount(contentID int64) int64 {
	if l.svcCtx.DB == nil {
		return 0
	}

	var count int64
	l.svcCtx.DB.Raw("SELECT COALESCE(total_downloads, 0) FROM content_download_stats WHERE content_id = ?", contentID).Scan(&count)
	return count
}

// getContentType 根据文件URL获取内容类型
func (l *ContentDownloadLogic) getContentType(url string) string {
	if len(url) == 0 {
		return "application/octet-stream"
	}

	ext := ""
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '.' {
			ext = url[i:]
			break
		}
		if url[i] == '/' || url[i] == '?' {
			break
		}
	}

	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// downloadRecord 下载记录结构体
type downloadRecord struct {
	ID           int64 `gorm:"column:id"`
	UserID       int64 `gorm:"column:user_id"`
	ContentID    int64 `gorm:"column:content_id"`
	ContentTitle string
	FileURL      string
	DownloadTime time.Time
	Status       int16
	IPAddress    string
	DeviceInfo   string
	FileSize     int64
	CreatedAt    time.Time
}
