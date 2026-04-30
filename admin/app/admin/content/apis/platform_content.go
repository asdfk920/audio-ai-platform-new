package apis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/google/uuid"

	"go-admin/app/admin/models"
	"go-admin/common/file_store"
)

// PlatformContent 内容 API
type PlatformContent struct {
	api.Api
}

// List 服务初始化失败的
// @Summary 服务初始化失败的
// @Tags 平台内容
// @Param page query int false "ID(ID1)"
// @Param page_size query int false "内容的 (default10)"
// @Param title query string false "内容的"
// @Param artist query string false "内容"
// @Param category_id query int false "IDID"
// @Param vip_level query int false "内容的"
// @Param status query int false "状态 (0/1/2)"
// @Param is_delete query int false "内容的 (0/1;default0)"
// @Router /api/v1/platform-content/list [get]
// @Security Bearer
func (e PlatformContent) List(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	page, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("page", "1")))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("page_size", "10")))
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}

	title := strings.TrimSpace(c.Query("title"))
	artist := strings.TrimSpace(c.Query("artist"))
	categoryID := strings.TrimSpace(c.Query("category_id"))
	vipLevel := strings.TrimSpace(c.Query("vip_level"))
	statusFilter := strings.TrimSpace(c.Query("status"))
	isDelete := strings.TrimSpace(c.DefaultQuery("is_delete", "0"))

	base := e.Orm.Table("content")
	// 服务初始化失败is_delete=1 内容
	if isDelete == "" {
		isDelete = "0"
	}
	if isDelete == "0" || isDelete == "1" {
		base = base.Where("is_deleted = ?", isDelete)
	} else {
		e.Error(400, errors.New("bad request"), "is_delete 内容0 的1")
		return
	}

	if title != "" {
		base = base.Where("title ILIKE ?", "%"+title+"%")
	}
	if artist != "" {
		base = base.Where("artist ILIKE ?", "%"+artist+"%")
	}
	if categoryID != "" {
		n, err := strconv.ParseInt(categoryID, 10, 64)
		if err != nil || n < 0 {
			e.Error(400, err, "category_id ID")
			return
		}
		if n == 0 {
			base = base.Where("category_id IS NULL")
		} else {
			base = base.Where("category_id = ?", n)
		}
	}
	if vipLevel != "" {
		n, err := strconv.ParseInt(vipLevel, 10, 16)
		if err != nil || n < 0 || n > 3 {
			e.Error(400, err, "vip_level ID")
			return
		}
		base = base.Where("vip_level = ?", n)
	}
	if statusFilter != "" {
		n, err := strconv.ParseInt(statusFilter, 10, 16)
		if err != nil || n < 0 || n > 3 {
			e.Error(400, err, "status invalid")
			return
		}
		base = base.Where("status = ?", n)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	offset := (page - 1) * pageSize
	type row struct {
		ID              int64      `gorm:"column:id"`
		Title           string     `gorm:"column:title"`
		Artist          string     `gorm:"column:artist"`
		CoverURL        string     `gorm:"column:cover_url"`
		Duration        int        `gorm:"column:duration_sec"`
		VipLevel        int16      `gorm:"column:vip_level"`
		Status          int16      `gorm:"column:status"`
		IsDeleted       int16      `gorm:"column:is_deleted"`
		CategoryID      *int64     `gorm:"column:category_id"`
		SortOrder       int        `gorm:"column:sort_order"`
		AudioValidFrom  *time.Time `gorm:"column:audio_valid_from"`
		AudioValidUntil *time.Time `gorm:"column:audio_valid_until"`
		CreatedAt       time.Time  `gorm:"column:created_at"`
		UpdatedAt       time.Time  `gorm:"column:updated_at"`
	}
	var rows []row
	if err := base.
		Select("id,title,artist,cover_url,duration_sec,vip_level,status,is_deleted,category_id,sort_order,audio_valid_from,audio_valid_until,created_at,updated_at").
		Order("sort_order DESC, updated_at DESC, id DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	// 服务初始化失败
	catName := map[int64]string{}
	{
		var catIDs []int64
		seen := map[int64]bool{}
		for _, r := range rows {
			if r.CategoryID != nil && *r.CategoryID > 0 && !seen[*r.CategoryID] {
				seen[*r.CategoryID] = true
				catIDs = append(catIDs, *r.CategoryID)
			}
		}
		if len(catIDs) > 0 {
			var cats []struct {
				ID   int64  `gorm:"column:id"`
				Name string `gorm:"column:name"`
			}
			_ = e.Orm.Table("content_category").Select("id,name").Where("id IN ?", catIDs).Scan(&cats).Error
			for _, c2 := range cats {
				catName[c2.ID] = strings.TrimSpace(c2.Name)
			}
		}
	}

	// 服务初始化失败内容 string 内容
	tagMap := map[int64][]string{}
	{
		var cids []int64
		for _, r := range rows {
			cids = append(cids, r.ID)
		}
		if len(cids) > 0 {
			type tagRow struct {
				ContentID int64  `gorm:"column:content_id"`
				TagName   string `gorm:"column:tag_name"`
			}
			var trs []tagRow
			_ = e.Orm.Raw(`
SELECT ctr.content_id, ct.tag_name
FROM content_tag_relation ctr
INNER JOIN content_tag ct ON ct.id = ctr.tag_id
WHERE ctr.content_id IN ?
ORDER BY ct.id ASC`, cids).Scan(&trs).Error
			for _, tr := range trs {
				tagMap[tr.ContentID] = append(tagMap[tr.ContentID], strings.TrimSpace(tr.TagName))
			}
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var cat string
		if r.CategoryID != nil && *r.CategoryID > 0 {
			cat = catName[*r.CategoryID]
		}
		out = append(out, gin.H{
			"id":                r.ID,
			"title":             r.Title,
			"artist":            r.Artist,
			"cover_url":         r.CoverURL,
			"duration":          r.Duration,
			"duration_sec":      r.Duration,
			"vip_level":         r.VipLevel,
			"vip_text":          vipText(int32(r.VipLevel)),
			"status":            r.Status,
			"status_text":       statusText(int32(r.Status)),
			"is_delete":         r.IsDeleted,
			"category_name":     cat,
			"tag_list":          tagMap[r.ID],
			"audio_valid_from":  formatTimePtr(r.AudioValidFrom),
			"audio_valid_until": formatTimePtr(r.AudioValidUntil),
			"create_time":       r.CreatedAt.Format("2006-01-02 15:04:05"),
			"update_time":       r.UpdatedAt.Format("2006-01-02 15:04:05"),
			"operator":          "",
		})
	}

	e.OK(gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"list":      out,
	}, "success")
}

func statusText(st int32) string {
	switch st {
	case 0:
		return "待审核"
	case 1:
		return "已通过"
	case 2:
		return "已拒绝"
	case 3:
		return "已下架"
	default:
		return "未知"
	}
}

func vipText(v int32) string {
	if v <= 0 {
		return "免费"
	}
	return "VIP 内容"
}

// Detail 内容详情
// @Summary 内容详情
// @Tags 平台内容
// @Param content_id query int true "内容 ID"
// @Router /api/v1/platform-content/detail [get]
// @Security Bearer
func (e PlatformContent) Detail(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	s := strings.TrimSpace(c.Query("content_id"))
	if s == "" {
		e.Error(400, errors.New("bad request"), "content_id 内容的")
		return
	}
	contentID, err := strconv.ParseInt(s, 10, 64)
	if err != nil || contentID <= 0 {
		e.Error(400, err, "content_id ID")
		return
	}

	// 1) 服务初始化失败内容/内容的
	type contentRow struct {
		ID              int64           `gorm:"column:id"`
		Title           string          `gorm:"column:title"`
		Artist          string          `gorm:"column:artist"`
		Duration        int             `gorm:"column:duration_sec"`
		CoverURL        string          `gorm:"column:cover_url"`
		AudioURL        string          `gorm:"column:audio_url"`
		Format          string          `gorm:"column:format"`
		SizeBytes       int64           `gorm:"column:size_bytes"`
		CategoryID      *int64          `gorm:"column:category_id"`
		VipLevel        int16           `gorm:"column:vip_level"`
		Status          int16           `gorm:"column:status"`
		IsDeleted       int16           `gorm:"column:is_deleted"`
		SortOrder       int             `gorm:"column:sort_order"`
		SpatialRaw      json.RawMessage `gorm:"column:spatial_params"`
		AudioValidFrom  *time.Time      `gorm:"column:audio_valid_from"`
		AudioValidUntil *time.Time      `gorm:"column:audio_valid_until"`
		CreatedAt       time.Time       `gorm:"column:created_at"`
		UpdatedAt       time.Time       `gorm:"column:updated_at"`
	}
	var row contentRow
	if err := e.Orm.Table("content").
		Select("id,title,artist,duration_sec,cover_url,audio_url,format,COALESCE(size_bytes,0) AS size_bytes,category_id,vip_level,status,is_deleted,sort_order,spatial_params,audio_valid_from,audio_valid_until,created_at,updated_at").
		Where("id = ?", contentID).
		Limit(1).
		Scan(&row).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}
	if row.ID == 0 {
		e.Error(404, errors.New("not found"), "内容")
		return
	}

	// 2) 内容的
	categoryName := ""
	if row.CategoryID != nil && *row.CategoryID > 0 {
		var cat struct {
			Name string `gorm:"column:name"`
		}
		_ = e.Orm.Table("content_category").Select("name").Where("id = ?", *row.CategoryID).Limit(1).Scan(&cat).Error
		categoryName = strings.TrimSpace(cat.Name)
	}

	// 3) 内容的
	type tagItem struct {
		TagID   int64  `json:"tag_id" gorm:"column:tag_id"`
		TagName string `json:"tag_name" gorm:"column:tag_name"`
	}
	var tags []tagItem
	_ = e.Orm.Raw(`
SELECT ct.id AS tag_id, ct.tag_name
FROM content_tag_relation ctr
INNER JOIN content_tag ct ON ct.id = ctr.tag_id
WHERE ctr.content_id = ?
ORDER BY ct.id ASC`, contentID).Scan(&tags).Error

	// operator 内容的 content 服务初始化失败服务初始化失败内容
	operator := ""

	e.OK(gin.H{
		"id":            row.ID,
		"title":         row.Title,
		"artist":        row.Artist,
		"duration":      row.Duration,
		"duration_sec":  row.Duration,
		"cover_url":     row.CoverURL,
		"audio_url":     row.AudioURL,
		"format":        row.Format,
		"file_size":     row.SizeBytes,
		"category_id":   row.CategoryID,
		"category_name": categoryName,
		"vip_level":     row.VipLevel,
		"spatial_params": func() any {
			if len(row.SpatialRaw) == 0 {
				return nil
			}
			return row.SpatialRaw
		}(),
		"audio_valid_from":  formatTimePtr(row.AudioValidFrom),
		"audio_valid_until": formatTimePtr(row.AudioValidUntil),
		"status":            row.Status,
		"is_delete":         row.IsDeleted,
		"sort":              row.SortOrder,
		"tag_list":          tags,
		"create_time":       row.CreatedAt.Format("2006-01-02 15:04:05"),
		"update_time":       row.UpdatedAt.Format("2006-01-02 15:04:05"),
		"operator":          operator,
	}, "success")
}

// Online 服务初始化失败的
// @Summary 内容的
// @Tags 平台内容
// @Accept application/json
// @Accept multipart/form-data
// @Param content_id formData int true "IDID"
// @Router /api/v1/platform-content/online [post]
// @Security Bearer
func (e PlatformContent) Online(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 内容orm 的json
	var contentID int64
	if s := strings.TrimSpace(c.PostForm("content_id")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			contentID = n
		}
	} else {
		var body struct {
			ContentID int64 `json:"content_id"`
		}
		_ = c.ShouldBindJSON(&body)
		contentID = body.ContentID
	}
	if contentID <= 0 {
		e.Error(400, errors.New("bad request"), "content_id 内容的")
		return
	}

	// 服务初始化失败的
	var row struct {
		ID        int64 `gorm:"column:id"`
		Status    int32 `gorm:"column:status"`
		IsDeleted int16 `gorm:"column:is_deleted"`
	}
	if err := e.Orm.Table("content").
		Select("id, status, is_deleted").
		Where("id = ?", contentID).
		Limit(1).
		Scan(&row).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}
	if row.ID == 0 || row.IsDeleted != 0 {
		e.Error(404, errors.New("not found"), "服务初始化失败内容")
		return
	}

	// 服务初始化失败default0/2 -> 1的 服务初始化失败内容
	if row.Status == 1 {
		// 服务初始化失败服务初始化失败URL的
		go e.refreshContentCaches(contentID)
		e.OK(gin.H{"content_id": contentID}, "内容")
		return
	}
	if row.Status != 0 && row.Status != 2 {
		e.Error(400, errors.New("bad request"), "服务初始化失败内容")
		return
	}

	if err := e.Orm.Table("content").
		Where("id = ? AND is_deleted = 0", contentID).
		Updates(map[string]any{
			"status":     1,
			"updated_at": time.Now(),
		}).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	// 内容
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		log := models.ContentLog{
			ContentId:     contentID,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "publish",
			ChangedFields: `{"status":1}`,
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("服务初始化失败内容defaultv", err)
		}
	}()

	// 服务初始化失败+ default/服务初始化失败est-effort的
	go e.refreshContentCaches(contentID)

	e.OK(gin.H{"content_id": contentID}, "内容")
}

// Offline 服务初始化失败的
// @Summary 内容的
// @Tags 平台内容
// @Accept application/json
// @Accept multipart/form-data
// @Param content_id formData int true "IDID"
// @Router /api/v1/platform-content/offline [post]
// @Security Bearer
func (e PlatformContent) Offline(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 内容orm 的json
	var contentID int64
	if s := strings.TrimSpace(c.PostForm("content_id")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			contentID = n
		}
	} else {
		var body struct {
			ContentID int64 `json:"content_id"`
		}
		_ = c.ShouldBindJSON(&body)
		contentID = body.ContentID
	}
	if contentID <= 0 {
		e.Error(400, errors.New("bad request"), "content_id 内容的")
		return
	}

	// 服务初始化失败的
	var row struct {
		ID        int64 `gorm:"column:id"`
		Status    int32 `gorm:"column:status"`
		IsDeleted int16 `gorm:"column:is_deleted"`
	}
	if err := e.Orm.Table("content").
		Select("id, status, is_deleted").
		Where("id = ?", contentID).
		Limit(1).
		Scan(&row).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}
	if row.ID == 0 || row.IsDeleted != 0 {
		e.Error(404, errors.New("not found"), "服务初始化失败内容")
		return
	}

	// 服务初始化失败服务初始化失败
	if row.Status == 2 {
		go e.refreshContentCaches(contentID)
		e.OK(gin.H{"content_id": contentID}, "内容")
		return
	}

	if err := e.Orm.Table("content").
		Where("id = ? AND is_deleted = 0", contentID).
		Updates(map[string]any{
			"status":     2,
			"updated_at": time.Now(),
		}).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	// 内容
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		log := models.ContentLog{
			ContentId:     contentID,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "unpublish",
			ChangedFields: `{"status":2}`,
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("服务初始化失败内容defaultv", err)
		}
	}()

	go e.refreshContentCaches(contentID)
	e.OK(gin.H{"content_id": contentID}, "内容")
}

// Delete 服务初始化失败的
// @Summary 服务初始化失败的
// @Tags 平台内容
// @Accept application/json
// @Accept multipart/form-data
// @Param content_id formData int true "IDID"
// @Router /api/v1/platform-content/delete [post]
// @Security Bearer
func (e PlatformContent) Delete(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "service initialization failed")
		return
	}

	// 兼容：form 或 json
	var contentID int64
	if s := strings.TrimSpace(c.PostForm("content_id")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			contentID = n
		}
	} else {
		var body struct {
			ContentID int64 `json:"content_id"`
		}
		_ = c.ShouldBindJSON(&body)
		contentID = body.ContentID
	}
	if contentID <= 0 {
		e.Error(400, errors.New("bad request"), "content_id cannot be empty")
		return
	}

	// 查询内容是否存在及删除状态
	var row struct {
		ID        int64  `gorm:"column:id"`
		IsDeleted int16  `gorm:"column:is_deleted"`
		Title     string `gorm:"column:title"`
	}
	if err := e.Orm.Table("content").
		Select("id, is_deleted, title").
		Where("id = ?", contentID).
		Limit(1).
		Scan(&row).Error; err != nil {
		e.Error(500, err, "query content failed")
		return
	}
	if row.ID == 0 {
		e.OK(gin.H{"content_id": contentID}, "delete content success")
		return
	}
	if row.IsDeleted != 0 {
		// 幂等：已删除直接成功，但仍刷新缓存
		go e.refreshContentCaches(contentID)
		e.OK(gin.H{"content_id": contentID}, "delete content success")
		return
	}

	// 检查是否被收藏引用
	var favoriteCount int64
	if err := e.Orm.Table("user_favorite_content").
		Where("content_id = ? AND deleted_at IS NULL", contentID).
		Count(&favoriteCount).Error; err != nil {
		e.Error(500, err, "check favorite reference failed")
		return
	}
	if favoriteCount > 0 {
		e.Error(400, errors.New("bad request"),
			fmt.Sprintf("content has been favorited by %d users, cannot delete", favoriteCount))
		return
	}

	// 检查是否有下载记录
	var downloadCount int64
	if err := e.Orm.Table("user_download").
		Where("content_id = ? AND deleted_at IS NULL", contentID).
		Count(&downloadCount).Error; err != nil {
		e.Error(500, err, "check download reference failed")
		return
	}
	if downloadCount > 0 {
		e.Error(400, errors.New("bad request"),
			fmt.Sprintf("content has been downloaded %d times, cannot delete", downloadCount))
		return
	}

	// 检查是否有播放记录（最近 30 天）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var playCount int64
	if err := e.Orm.Table("user_play_record").
		Where("content_id = ? AND play_time >= ?", contentID, thirtyDaysAgo).
		Count(&playCount).Error; err != nil {
		e.Error(500, err, "check play reference failed")
		return
	}
	if playCount > 0 {
		e.Error(400, errors.New("bad request"),
			fmt.Sprintf("content has %d play records in last 30 days, cannot delete", playCount))
		return
	}

	// 执行软删除
	if err := e.Orm.Table("content").
		Where("id = ? AND is_deleted = 0", contentID).
		Updates(map[string]any{
			"is_deleted": 1,
			"status":     2,
			"updated_at": time.Now(),
		}).Error; err != nil {
		e.Error(500, err, "delete content failed")
		return
	}

	// 记录操作日志
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		log := models.ContentLog{
			ContentId:     contentID,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "delete",
			ChangedFields: fmt.Sprintf(`{"is_deleted":1,"status":2,"title":"%s"}`, row.Title),
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("record content delete log failed: %v", err)
		}
	}()

	// 刷新内容服务缓存
	go e.refreshContentCaches(contentID)

	e.OK(gin.H{"content_id": contentID}, "delete content success")
}

// Restore 恢复已删除内容（幂等）
// @Summary 恢复已删除内容
// @Tags 平台内容
// @Accept application/json
// @Accept multipart/form-data
// @Param content_id formData int true "内容 ID"
// @Router /api/v1/platform-content/restore [post]
// @Security Bearer
func (e PlatformContent) Restore(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 兼容：form 或 json
	var contentID int64
	if s := strings.TrimSpace(c.PostForm("content_id")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			contentID = n
		}
	} else {
		var body struct {
			ContentID int64 `json:"content_id"`
		}
		_ = c.ShouldBindJSON(&body)
		contentID = body.ContentID
	}
	if contentID <= 0 {
		e.Error(400, errors.New("bad request"), "content_id cannot be empty")
		return
	}

	// 查询已被软删除的内容
	var row struct {
		ID        int64  `gorm:"column:id"`
		IsDeleted int16  `gorm:"column:is_deleted"`
		Title     string `gorm:"column:title"`
		Status    int32  `gorm:"column:status"`
	}
	if err := e.Orm.Table("content").
		Select("id, is_deleted, title, status").
		Where("id = ?", contentID).
		Limit(1).
		Scan(&row).Error; err != nil {
		e.Error(500, err, "query content failed")
		return
	}
	if row.ID == 0 {
		e.Error(404, errors.New("not found"), "content does not exist")
		return
	}
	if row.IsDeleted == 0 {
		// 幂等：如果内容未被删除，直接返回成功
		e.OK(gin.H{"content_id": contentID}, "content is already active, restore success")
		return
	}

	// 检查标题是否与现有正常内容冲突
	var conflictCount int64
	if err := e.Orm.Table("content").
		Where("title = ? AND id != ? AND is_deleted = 0", row.Title, contentID).
		Count(&conflictCount).Error; err != nil {
		e.Error(500, err, "check title conflict failed")
		return
	}
	if conflictCount > 0 {
		e.Error(400, errors.New("bad request"),
			fmt.Sprintf("title '%s' conflicts with existing content, cannot restore", row.Title))
		return
	}

	// 执行恢复操作：重置删除标识，恢复为正常状态
	if err := e.Orm.Table("content").
		Where("id = ? AND is_deleted = 1", contentID).
		Updates(map[string]any{
			"is_deleted": 0,
			"status":     2, // 恢复后保持下架状态，需要管理员手动上架
			"updated_at": time.Now(),
		}).Error; err != nil {
		e.Error(500, err, "restore content failed")
		return
	}

	// 记录操作日志
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		log := models.ContentLog{
			ContentId:     contentID,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "restore",
			ChangedFields: fmt.Sprintf(`{"is_deleted":0,"status":2,"title":"%s"}`, row.Title),
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("record content restore log failed: %v", err)
		}
	}()

	// 刷新内容服务缓存
	go e.refreshContentCaches(contentID)

	e.OK(gin.H{"content_id": contentID}, "restore content success")
}

// Update 服务初始化失败内容
// @Summary 内容的
// @Tags 平台内容
// @Accept multipart/form-data
// @Param content_id formData int true "IDID"
// @Param title formData string false "ID"
// @Param artist formData string false "内容"
// @Param category_id formData int false "IDID"
// @Param tag_ids formData string false "defaultIDdefault(服务初始化失败)"
// @Param vip_level formData int false "内容的(0/1/2/3)"
// @Param duration formData int false "ID(的"
// @Param spatial_params formData string false "内容的JSON"
// @Param cover formData file false "内容的"
// @Param audio formData file false "内容的"
// @Param cover_url formData string false "defaultURL(内容)"
// @Param audio_url formData string false "defaultURL(内容)"
// @Param status formData int false "状态(0/1/2)"
// @Router /api/v1/platform-content/update [post]
// @Security Bearer
func (e PlatformContent) Update(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	cidStr, ok := c.GetPostForm("content_id")
	if !ok || strings.TrimSpace(cidStr) == "" {
		e.Error(400, errors.New("bad request"), "content_id 内容的")
		return
	}
	contentID, err := strconv.ParseInt(strings.TrimSpace(cidStr), 10, 64)
	if err != nil || contentID <= 0 {
		e.Error(400, err, "content_id ID")
		return
	}

	// 服务初始化失败的
	var exists int64
	if err := e.Orm.Table("content").Where("id = ? AND is_deleted = 0", contentID).Count(&exists).Error; err != nil {
		e.Error(500, err, "内容")
		return
	}
	if exists == 0 {
		e.Error(404, errors.New("not found"), "服务初始化失败内容")
		return
	}

	updates := make(map[string]any)

	if v, has := c.GetPostForm("title"); has {
		v = strings.TrimSpace(v)
		if v == "" {
			e.Error(400, errors.New("bad request"), "title 内容")
			return
		}
		updates["title"] = v
	}
	if v, has := c.GetPostForm("artist"); has {
		v = strings.TrimSpace(v)
		if v == "" {
			e.Error(400, errors.New("bad request"), "artist 内容")
			return
		}
		updates["artist"] = v
	}
	if v, has := c.GetPostForm("category_id"); has {
		v = strings.TrimSpace(v)
		if v == "" {
			updates["category_id"] = nil
		} else {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				e.Error(400, err, "category_id ID")
				return
			}
			updates["category_id"] = n
		}
	}
	if v, has := c.GetPostForm("vip_level"); has {
		n, err := parseIntRange(v, 0, 3)
		if err != nil {
			e.Error(400, err, "vip_level 内容0/1/2/3")
			return
		}
		updates["vip_level"] = n
	}
	if v, has := c.GetPostForm("duration"); has {
		n, err := parseIntRange(v, 1, 24*60*60)
		if err != nil {
			e.Error(400, err, "duration ID > 0")
			return
		}
		updates["duration_sec"] = n
	}
	if v, has := c.GetPostForm("status"); has {
		n, err := parseIntRange(v, 0, 2)
		if err != nil {
			e.Error(400, err, "status 0/1/2")
			return
		}
		updates["status"] = n
	}
	if v, has := c.GetPostForm("spatial_params"); has {
		v = strings.TrimSpace(v)
		if v != "" {
			var anyVal any
			if err := json.Unmarshal([]byte(v), &anyVal); err != nil {
				e.Error(400, err, "spatial_params 内容JSON")
				return
			}
			updates["spatial_params"] = json.RawMessage(v)
		} else {
			updates["spatial_params"] = nil
		}
	} else if flat, err := buildSpatialJSONFromFlatPost(c); err != nil {
		e.Error(400, err, "空间/渲染参数无效")
		return
	} else if flat != "" {
		updates["spatial_params"] = json.RawMessage(flat)
	}

	avUp, avErr := parseAudioValidity(c, true)
	if avErr != nil {
		e.Error(400, avErr, avErr.Error())
		return
	}
	if !avUp.Skip {
		if avUp.Clear {
			updates["audio_valid_from"] = nil
			updates["audio_valid_until"] = nil
		} else if avUp.From != nil {
			updates["audio_valid_from"] = *avUp.From
			updates["audio_valid_until"] = *avUp.Until
		}
	}

	coverURL, coverURLHas := c.GetPostForm("cover_url")
	audioURL, audioURLHas := c.GetPostForm("audio_url")
	coverURL = strings.TrimSpace(coverURL)
	audioURL = strings.TrimSpace(audioURL)

	// 服务初始化失败 cover/audio default
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		u, upErr := e.uploadFormFile(c, fh, "content/cover")
		if upErr != nil {
			e.Error(400, upErr, upErr.Error())
			return
		}
		coverURL = u
		coverURLHas = true
	}
	if fh, err := c.FormFile("audio"); err == nil && fh != nil {
		u, upErr := e.uploadFormFile(c, fh, "content/audio")
		if upErr != nil {
			e.Error(400, upErr, upErr.Error())
			return
		}
		audioURL = u
		audioURLHas = true
	}

	if coverURLHas {
		if coverURL == "" {
			e.Error(400, errors.New("bad request"), "cover_url 内容")
			return
		}
		if !looksLikeURL(coverURL) {
			e.Error(400, errors.New("bad request"), "cover_url 内容的")
			return
		}
		updates["cover_url"] = coverURL
	}
	if audioURLHas {
		if audioURL == "" {
			e.Error(400, errors.New("bad request"), "audio_url 内容")
			return
		}
		if !looksLikeURL(audioURL) {
			e.Error(400, errors.New("bad request"), "audio_url 内容的")
			return
		}
		updates["audio_url"] = audioURL
	}

	tagIDsRaw, tagIDsHas := c.GetPostForm("tag_ids")

	if len(updates) == 0 && !tagIDsHas {
		e.Error(400, errors.New("bad request"), "服务初始化失败内容")
		return
	}

	updates["updated_at"] = time.Now()

	tx := e.Orm.Begin()
	if tx.Error != nil {
		e.Error(500, tx.Error, "服务初始化失败")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if len(updates) > 0 {
		if err := tx.Table("content").Where("id = ? AND is_deleted = 0", contentID).Updates(updates).Error; err != nil {
			tx.Rollback()
			e.Error(500, err, "内容")
			return
		}
	}

	// 服务初始化失败tag_ids服务初始化失败内容
	if tagIDsHas {
		if err := tx.Exec(`DELETE FROM content_tag_relation WHERE content_id = ?`, contentID).Error; err != nil {
			tx.Rollback()
			e.Error(500, err, "服务初始化失败")
			return
		}
		tagIDs := parseCSVInt64(tagIDsRaw)
		for _, tid := range tagIDs {
			_ = tx.Exec(`INSERT INTO content_tag_relation (content_id, tag_id, created_at) VALUES (?, ?, NOW())
ON CONFLICT (content_id, tag_id) DO NOTHING`, contentID, tid).Error
		}
	}

	if err := tx.Commit().Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	// 内容
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 服务初始化失败内容defaultupdated_at的
		changedFields := make(map[string]interface{})
		for k, v := range updates {
			if k != "updated_at" {
				changedFields[k] = v
			}
		}

		// 服务初始化失败的
		changedFieldsJSON, _ := json.Marshal(changedFields)

		// 内容的
		log := models.ContentLog{
			ContentId:     contentID,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "update",
			ChangedFields: string(changedFieldsJSON),
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("服务初始化失败内容defaultv", err)
		}
	}()

	// 服务初始化失败+ default/服务初始化失败est-effort的
	go e.refreshContentCaches(contentID)

	e.OK(gin.H{"content_id": contentID}, "内容")
}

func looksLikeURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "/")
}

func (e PlatformContent) refreshContentCaches(contentID int64) {
	base := strings.TrimSpace(os.Getenv("CONTENT_SERVICE_BASE_URL"))
	secret := strings.TrimSpace(os.Getenv("CONTENT_INTERNAL_SECRET"))
	if base == "" || secret == "" {
		return
	}
	base = strings.TrimRight(base, "/")

	// 1) 内容
	body, _ := json.Marshal(map[string]any{"content_id": contentID})
	req1, _ := http.NewRequest(http.MethodPost, base+"/api/v1/content/internal/del-detail-cache", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Internal-Secret", secret)

	// 2) bump default/内容
	req2, _ := http.NewRequest(http.MethodPost, base+"/api/v1/content/internal/bump-list-cache", nil)
	req2.Header.Set("X-Internal-Secret", secret)

	client := &http.Client{Timeout: 2 * time.Second}
	_, _ = client.Do(req1)
	_, _ = client.Do(req2)
}

// Add 内容的
// @Summary 内容的
// @Tags 平台内容
// @Accept multipart/form-data
// @Param title formData string true "ID"
// @Param artist formData string true "内容"
// @Param category_id formData int false "IDID"
// @Param tag_ids formData string false "defaultIDdefault(内容的)"
// @Param vip_level formData int true "内容的(0/1/2/3)"
// @Param duration formData int true "ID(的)"
// @Param spatial_params formData string false "内容的JSON"
// @Param cover formData file false "内容的"
// @Param audio formData file false "内容的"
// @Param cover_url formData string false "defaultURL(内容)"
// @Param audio_url formData string false "defaultURL(内容)"
// @Param status formData int true "状态(0/1)"
// @Router /api/v1/platform-content/add [post]
// @Security Bearer
func (e PlatformContent) Add(c *gin.Context) {
	if err := e.MakeContext(c).MakeOrm().Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// #region agent log
	debugSessionLog167bb8("H1", "platform_content.go:Add:entry", "request_headers", map[string]any{
		"contentType": c.GetHeader("Content-Type"),
		"method":      c.Request.Method,
	})
	// #endregion

	title := strings.TrimSpace(c.PostForm("title"))
	artist := strings.TrimSpace(c.PostForm("artist"))
	if title == "" || artist == "" {
		// #region agent log
		debugSessionLog167bb8("H1", "platform_content.go:Add:title_artist", "validation_fail", map[string]any{
			"titleLen": len(title), "artistLen": len(artist),
		})
		// #endregion
		e.Error(400, errors.New("bad request"), "title/artist 内容的")
		return
	}

	vipLevel, err := parseIntRange(c.PostForm("vip_level"), 0, 3)
	if err != nil {
		e.Error(400, err, "vip_level 内容0/1/2/3")
		return
	}
	duration, err := parseIntRange(c.PostForm("duration"), 1, 24*60*60)
	if err != nil {
		e.Error(400, err, "duration ID > 0")
		return
	}
	// 与 public.content.status 及 Update 一致：0 草稿 1 上架 2 下架；缺省为 0（运营「确认创建」可不传）
	statusStr := strings.TrimSpace(c.PostForm("status"))
	statusVal := 0
	if statusStr != "" {
		statusVal, err = parseIntRange(statusStr, 0, 2)
		if err != nil {
			e.Error(400, err, "status 应为 0/1/2")
			return
		}
	}

	var categoryID *int64
	if s := strings.TrimSpace(c.PostForm("category_id")); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < 0 {
			e.Error(400, err, "category_id ID")
			return
		}
		categoryID = &n
	}

	spatial, err := spatialParamsFromRequest(c)
	if err != nil {
		e.Error(400, err, "spatial_params 格式无效")
		return
	}

	avAdd, avErr := parseAudioValidity(c, false)
	if avErr != nil {
		e.Error(400, avErr, avErr.Error())
		return
	}

	coverURL := strings.TrimSpace(c.PostForm("cover_url"))
	audioURL := strings.TrimSpace(c.PostForm("audio_url"))
	audioKey := strings.TrimSpace(c.PostForm("audio_key"))

	// 服务初始化失败的cover/audio
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		u, upErr := e.uploadFormFile(c, fh, "content/cover")
		if upErr != nil {
			e.Error(400, upErr, upErr.Error())
			return
		}
		coverURL = u
	}
	if fh, err := c.FormFile("audio"); err == nil && fh != nil {
		u, upErr := e.uploadFormFile(c, fh, "content/audio")
		if upErr != nil {
			e.Error(400, upErr, upErr.Error())
			return
		}
		audioURL = u
	}

	if coverURL == "" || audioURL == "" {
		// #region agent log
		debugSessionLog167bb8("H4", "platform_content.go:Add:cover_audio", "urls_missing_or_empty", map[string]any{
			"coverLen": len(coverURL), "audioLen": len(audioURL),
		})
		// #endregion
		e.Error(400, errors.New("bad request"), "cover_url/audio_url 服务初始化失败的 cover/audio 内容")
		return
	}

	tagIDs := parseCSVInt64(c.PostForm("tag_ids"))

	now := time.Now()
	op := user.GetUserId(c)

	// #region agent log
	debugSessionLog167bb8("H1", "platform_content.go:Add:validated", "fields_ok_before_tx", map[string]any{
		"duration": duration, "vipLevel": vipLevel, "status": statusVal,
		"titleLen": len(title), "artistLen": len(artist), "coverLen": len(coverURL), "audioLen": len(audioURL),
	})
	// #endregion

	// 的content IDpublic.content的
	type contentRow struct {
		ID int64 `gorm:"column:id"`
	}
	// format/size 服务初始化失败defaultheader 服务初始化失败的
	insert := map[string]any{
		"title":          title,
		"artist":         artist,
		"cover_url":      coverURL,
		"audio_url":      audioURL,
		"duration_sec":   duration,
		"vip_level":      vipLevel,
		"sort_order":     0,
		"status":         statusVal,
		"is_deleted":     0,
		"created_at":     now,
		"updated_at":     now,
		"spatial_params": nullJSON(spatial),
	}
	if categoryID != nil && *categoryID > 0 {
		insert["category_id"] = *categoryID
	}
	if avAdd.From != nil {
		insert["audio_valid_from"] = *avAdd.From
		insert["audio_valid_until"] = *avAdd.Until
	}
	if audioKey != "" {
		insert["audio_key"] = audioKey
	}
	if s := strings.TrimSpace(c.PostForm("audio_id")); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
			insert["audio_id"] = n
		}
	}
	if s := strings.TrimSpace(c.PostForm("cover_id")); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
			insert["cover_id"] = n
		}
	}

	tx := e.Orm.Begin()
	if tx.Error != nil {
		e.Error(500, tx.Error, "服务初始化失败")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Table("content").Create(insert).Error; err != nil {
		tx.Rollback()
		// #region agent log
		debugSessionLog167bb8("H4", "platform_content.go:Add:db_create", "insert_error", map[string]any{
			"err": err.Error(),
		})
		// #endregion
		e.Error(500, err, "内容")
		return
	}
	var cid int64
	_ = tx.Raw("SELECT currval(pg_get_serial_sequence('content','id'))").Scan(&cid).Error
	if cid <= 0 {
		var cr contentRow
		if err := tx.Table("content").Select("id").Where("title = ?", title).Order("id DESC").Limit(1).Scan(&cr).Error; err == nil {
			cid = cr.ID
		}
	}

	// 服务初始化失败内容default
	for _, tid := range tagIDs {
		_ = tx.Exec(`INSERT INTO content_tag_relation (content_id, tag_id, created_at) VALUES (?, ?, NOW())
ON CONFLICT (content_id, tag_id) DO NOTHING`, cid, tid).Error
	}

	if err := tx.Commit().Error; err != nil {
		e.Error(500, err, "内容")
		return
	}

	// 内容
	go func() {
		operator := user.GetUserId(c)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 服务初始化失败的
		createdFields := map[string]interface{}{
			"title":        title,
			"artist":       artist,
			"cover_url":    coverURL,
			"audio_url":    audioURL,
			"duration_sec": duration,
			"vip_level":    vipLevel,
			"status":       statusVal,
		}
		if categoryID != nil && *categoryID > 0 {
			createdFields["category_id"] = *categoryID
		}
		if spatial != "" {
			createdFields["spatial_params"] = json.RawMessage(spatial)
		}
		if avAdd.From != nil {
			createdFields["audio_valid_from"] = avAdd.From.Format(time.RFC3339)
			createdFields["audio_valid_until"] = avAdd.Until.Format(time.RFC3339)
		}

		// 内容
		createdFieldsJSON, _ := json.Marshal(createdFields)

		// 内容的
		log := models.ContentLog{
			ContentId:     cid,
			Operator:      fmt.Sprintf("%d", operator),
			Operation:     "create",
			ChangedFields: string(createdFieldsJSON),
			IpAddress:     ipAddress,
			UserAgent:     userAgent,
			CreateBy:      operator,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			e.Logger.Errorf("服务初始化失败内容defaultv", err)
		}
	}()

	// 服务初始化失败内容est-effort服务初始化失败的
	go e.bumpContentServiceCache()

	// #region agent log
	debugSessionLog167bb8("H4", "platform_content.go:Add:success", "insert_ok", map[string]any{
		"contentId": cid, "operator": op,
	})
	// #endregion

	e.OK(gin.H{"content_id": cid, "operator": op}, "内容")
}

// spatialParamsFromRequest 优先使用 spatial_params JSON；否则根据扁平表单字段组装（与后台「添加音频内容」表单一致）。
func spatialParamsFromRequest(c *gin.Context) (string, error) {
	raw := strings.TrimSpace(c.PostForm("spatial_params"))
	if raw != "" {
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return "", err
		}
		return raw, nil
	}
	return buildSpatialJSONFromFlatPost(c)
}

func buildSpatialJSONFromFlatPost(c *gin.Context) (string, error) {
	keys := []string{"pos_x", "pos_y", "pos_z", "yaw", "pitch", "roll", "render_distance", "render_gain", "render_filter"}
	anySet := false
	for _, k := range keys {
		if strings.TrimSpace(c.PostForm(k)) != "" {
			anySet = true
			break
		}
	}
	if !anySet {
		return "", nil
	}
	px := parseFloatPostForm(c, "pos_x", 1.5)
	py := parseFloatPostForm(c, "pos_y", 0)
	pz := parseFloatPostForm(c, "pos_z", 2.0)
	yaw := parseFloatPostForm(c, "yaw", 90)
	pitch := parseFloatPostForm(c, "pitch", 0)
	roll := parseFloatPostForm(c, "roll", 0)
	dist := parseFloatPostForm(c, "render_distance", 10)
	gain := parseFloatPostForm(c, "render_gain", 1.0)
	filt := strings.TrimSpace(c.PostForm("render_filter"))
	if filt == "" {
		filt = "lowpass"
	}
	doc := map[string]any{
		"position":    map[string]float64{"x": px, "y": py, "z": pz},
		"orientation": map[string]float64{"yaw": yaw, "pitch": pitch, "roll": roll},
		"render":      map[string]any{"distance": dist, "gain": gain, "filter": filt},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseFloatPostForm(c *gin.Context, key string, def float64) float64 {
	s := strings.TrimSpace(c.PostForm(key))
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}

// audioValidityParsed 音频有效期：none=不限制；range=起止时间；Skip=更新时未带 audio_validity_mode（兼容旧客户端）。
type audioValidityParsed struct {
	From  *time.Time
	Until *time.Time
	Skip  bool
	Clear bool
}

func parseAdminDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", time.RFC3339}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

func parseAudioValidity(c *gin.Context, isUpdate bool) (audioValidityParsed, error) {
	var out audioValidityParsed
	mode := strings.TrimSpace(strings.ToLower(c.PostForm("audio_validity_mode")))
	if isUpdate && mode == "" {
		out.Skip = true
		return out, nil
	}
	if mode == "" {
		mode = "none"
	}
	if mode == "none" {
		if isUpdate {
			out.Clear = true
		}
		return out, nil
	}
	if mode != "range" {
		return out, fmt.Errorf("audio_validity_mode 仅支持 none 或 range")
	}
	vfStr := strings.TrimSpace(c.PostForm("audio_valid_from"))
	vuStr := strings.TrimSpace(c.PostForm("audio_valid_until"))
	if vfStr == "" || vuStr == "" {
		return out, fmt.Errorf("请选择音频有效期的开始与结束时间")
	}
	tf, err := parseAdminDateTime(vfStr)
	if err != nil {
		return out, fmt.Errorf("audio_valid_from 格式无效")
	}
	tu, err := parseAdminDateTime(vuStr)
	if err != nil {
		return out, fmt.Errorf("audio_valid_until 格式无效")
	}
	if tu.Before(tf) {
		return out, fmt.Errorf("有效期结束须晚于或等于开始时间")
	}
	out.From = &tf
	out.Until = &tu
	return out, nil
}

func parseIntRange(s string, min, max int) (int, error) {
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("invalid")
	}
	return n, nil
}

func parseCSVInt64(s string) []int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	seen := map[int64]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func nullJSON(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

// uploadFormFile 服务初始化失败内容 local/oss 服务初始化失败内容 URL的
func (e PlatformContent) uploadFormFile(c *gin.Context, fh *multipart.FileHeader, prefix string) (string, error) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	size := fh.Size

	// 文件扩展名和大小校验
	if strings.Contains(prefix, "cover") {
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			return "", fmt.Errorf("invalid cover format: jpg/png/webp only")
		}
		if size > 10<<20 {
			return "", fmt.Errorf("cover size exceeds 10MB")
		}
	} else {
		if ext != ".mp3" && ext != ".wav" && ext != ".flac" && ext != ".aac" {
			return "", fmt.Errorf("invalid audio format: mp3/wav/flac/aac only")
		}
		if size > 100<<20 {
			return "", fmt.Errorf("audio size exceeds 100MB")
		}
	}

	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, f); err != nil {
		return "", err
	}

	day := time.Now().Format("20060102")
	name := uuid.NewString()
	objectKey := fmt.Sprintf("%s/%s/%s%s", strings.TrimRight(prefix, "/"), day, name, ext)

	driver := strings.ToLower(strings.TrimSpace(os.Getenv("UPLOAD_STORAGE_DRIVER")))
	if driver == "" {
		driver = strings.ToLower(strings.TrimSpace(e.sysConfigValue("upload_storage_driver")))
		if driver == "" {
			driver = "local"
		}
	}

	switch driver {
	case "oss":
		// 内容OSS服务初始化失败内容 env/sys_config default
		endpoint := firstNonEmpty(os.Getenv("OSS_ENDPOINT"), e.sysConfigValue("upload_storage_oss_endpoint"))
		bucket := firstNonEmpty(os.Getenv("OSS_BUCKET_NAME"), e.sysConfigValue("upload_storage_oss_bucket"))
		ak := strings.TrimSpace(os.Getenv("OSS_ACCESS_KEY_ID"))
		sk := strings.TrimSpace(os.Getenv("OSS_ACCESS_KEY_SECRET"))
		if endpoint == "" || bucket == "" || ak == "" || sk == "" {
			return "", fmt.Errorf("oss 内容的")
		}

		tmp, err := os.CreateTemp("", "admin-content-upload-*"+ext)
		if err != nil {
			return "", err
		}
		tmpPath := tmp.Name()
		if _, err := tmp.Write(buf.Bytes()); err != nil {
			tmp.Close()
			_ = os.Remove(tmpPath)
			return "", err
		}
		_ = tmp.Close()
		defer os.Remove(tmpPath)

		var eng file_store.ALiYunOSS
		if err := eng.Setup(endpoint, ak, sk, bucket); err != nil {
			return "", err
		}
		if err := eng.UpLoad(objectKey, tmpPath); err != nil {
			return "", err
		}
		return e.publicURL(objectKey, c), nil
	default:
		// local内容static/uploadfile/ 的
		baseDir := "static/uploadfile"
		full := filepath.Join(baseDir, filepath.FromSlash(objectKey))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(full, buf.Bytes(), 0o644); err != nil {
			return "", err
		}
		// public URL
		return e.publicURL("/"+strings.TrimLeft(filepath.ToSlash(full), "/"), c), nil
	}
}

func (e PlatformContent) sysConfigValue(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || e.Orm == nil {
		return ""
	}
	type row struct {
		ConfigValue string `gorm:"column:config_value"`
	}
	var r row
	_ = e.Orm.Table("sys_config").Select("config_value").
		Where("deleted_at IS NULL AND config_key = ?", key).
		Limit(1).
		Scan(&r).Error
	return strings.TrimSpace(r.ConfigValue)
}

func (e PlatformContent) publicURL(objectKey string, c *gin.Context) string {
	base := strings.TrimSpace(os.Getenv("OSS_PUBLIC_BASE_URL"))
	if base == "" {
		base = e.sysConfigValue("upload_storage_public_base_url")
	}
	if base == "" {
		// 服务初始化失败host default
		base = fmt.Sprintf("%s://%s", "http", c.Request.Host)
	}
	base = strings.TrimRight(base, "/")
	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	return base + "/" + objectKey
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func (e PlatformContent) bumpContentServiceCache() {
	// 服务初始化失败defaultsecret内容的 secret default repo的
	base := strings.TrimSpace(os.Getenv("CONTENT_SERVICE_BASE_URL"))
	secret := strings.TrimSpace(os.Getenv("CONTENT_INTERNAL_SECRET"))
	if base == "" || secret == "" {
		return
	}
	base = strings.TrimRight(base, "/")
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/content/internal/bump-list-cache", nil)
	req.Header.Set("X-Internal-Secret", secret)
	client := &http.Client{Timeout: 2 * time.Second}
	_, _ = client.Do(req)
}
