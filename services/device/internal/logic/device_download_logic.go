package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceDownloadLogic 设备下载完整流程逻辑
type DeviceDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceDownloadLogic {
	return &DeviceDownloadLogic{ctx: ctx, svcCtx: svcCtx}
}

// Download 处理设备下载请求（完整流程）
// 流程：
// 1. JWT鉴权 + 参数校验
// 2. 调用内容服务获取歌曲信息（audio_url, title）
// 3. 校验用户权限和设备绑定关系
// 4. 更新 content_download_stats 下载统计
// 5. 通过WebSocket下发下载指令到设备
// 6. 返回任务ID
func (l *DeviceDownloadLogic) Download(req *types.DeviceDownloadReq) (*types.DeviceDownloadResp, error) {

	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	if err := validateReqSN(sn); err != nil {
		return nil, err
	}

	if req.ContentID <= 0 {
		return nil, fmt.Errorf("content_id 必须为正整数")
	}

	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("无权限控制该设备")
	}

	contentInfo, fetchErr := l.fetchContentFromContentService(req.ContentID)
	if fetchErr != nil {
		logx.Errorf("[DeviceDownload] 获取内容信息失败: content_id=%d error=%v", req.ContentID, fetchErr)
		return nil, fmt.Errorf("获取内容信息失败: %v", fetchErr)
	}

	taskID := uuid.New().String()
	songName := contentInfo.Title
	if songName == "" {
		songName = fmt.Sprintf("歌曲_%d", req.ContentID)
	}

	statsRepo := l.svcCtx.ContentDownloadStatsRepo
	if statsErr := statsRepo.IncrementDownloads(l.ctx, req.ContentID); statsErr != nil {
		logx.Errorf("[DeviceDownload] 更新下载统计失败: content_id=%d error=%v", req.ContentID, statsErr)
		return nil, fmt.Errorf("更新下载统计失败: %v", statsErr)
	}

	logx.Infof("[DeviceDownload] ✅ 下载统计已记录到 content_download_stats 表: content_id=%d user_id=%d device_sn=%s song_name=%s",
		req.ContentID, userID, sn, songName)

	userDownloadsRepo := l.svcCtx.UserDownloadsRepo
	downloadRecord := &model.UserDownload{
		UserID:       userID,
		ContentID:    req.ContentID,
		ContentTitle: songName,
		FileURL:      contentInfo.AudioURL,
		Status:       "downloading",
	}

	if createErr := userDownloadsRepo.Create(l.ctx, downloadRecord); createErr != nil {
		logx.Errorf("[DeviceDownload] 预创建下载记录失败: content_id=%d error=%v", req.ContentID, createErr)
		return nil, fmt.Errorf("预创建下载记录失败: %v", createErr)
	}

	logx.Infof("[DeviceDownload] ✅ 下载记录已预创建到 user_downloads 表: content_id=%d user_id=%d status=downloading",
		req.ContentID, userID)

	params := map[string]interface{}{
		"task_id":      taskID,
		"song_id":      req.ContentID,
		"song_name":    songName,
		"download_url": contentInfo.AudioURL,
		"download_cmd": "download_song",
	}

	cmdSvc := commandsvc.New(l.svcCtx)
	result, cmdErr := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "download_song",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下载歌曲(content_id=%d, name=%s)", req.ContentID, songName),
	})

	if cmdErr != nil {
		logx.Errorf("[DeviceDownload] 创建下载指令失败: task_id=%s error=%v", taskID, cmdErr)

		return &types.DeviceDownloadResp{
			TaskID:    taskID,
			ContentID: req.ContentID,
			DeviceSN:  sn,
			Status:    "failed",
			Message:   fmt.Sprintf("下载指令下发失败: %v", cmdErr),
		}, nil
	}

	status, message := MapInstructionDispatchOutcome(result.Status, "下载", "")

	logx.Infof("[DeviceDownload] 歌曲下载流程完成: task_id=%s status=%s message=%s",
		taskID, status, message)

	return &types.DeviceDownloadResp{
		TaskID:        taskID,
		ContentID:     req.ContentID,
		DeviceSN:      sn,
		Status:        status,
		Message:       message,
		InstructionID: func() *int64 { id := result.InstructionID; return &id }(),
	}, nil
}

// HandleCallback 处理设备下载回调（设备→设备服务）
func (l *DeviceDownloadLogic) HandleCallback(req *types.DeviceDownloadCallbackReq) (*types.DeviceDownloadCallbackResp, error) {
	logx.Infof("[DeviceDownloadCallback] 📥 收到下载回调: task_id=%s device_sn=%s content_id=%d status=%s",
		req.TaskID, req.DeviceSN, req.ContentID, req.Status)

	if req.TaskID == "" {
		logx.Errorf("[DeviceDownloadCallback] ❌ 参数错误: task_id 为空")
		return nil, fmt.Errorf("task_id 不能为空")
	}
	if req.DeviceSN == "" {
		logx.Errorf("[DeviceDownloadCallback] ❌ 参数错误: device_sn 为空")
		return nil, fmt.Errorf("device_sn 不能为空")
	}
	if req.ContentID <= 0 {
		logx.Errorf("[DeviceDownloadCallback] ❌ 参数错误: content_id 无效 (%d)", req.ContentID)
		return nil, fmt.Errorf("content_id 必须为正整数")
	}
	if req.Status != "success" && req.Status != "fail" {
		logx.Errorf("[DeviceDownloadCallback] ❌ 参数错误: status 无效 (%s)", req.Status)
		return nil, fmt.Errorf("status 必须为 success 或 fail")
	}
	if req.Status == "fail" && req.ErrorMsg == "" {
		logx.Errorf("[DeviceDownloadCallback] ❌ 参数错误: 下载失败但未提供原因")
		return nil, fmt.Errorf("下载失败时必须提供错误原因(error_msg)")
	}

	statsRepo := l.svcCtx.ContentDownloadStatsRepo

	existingRecord, findErr := statsRepo.FindByContentID(l.ctx, req.ContentID)
	if findErr != nil {
		logx.Errorf("[DeviceDownloadCallback] ❌ 查询预记录失败: content_id=%d error=%v", req.ContentID, findErr)
		return nil, fmt.Errorf("查询下载记录失败: %v", findErr)
	}

	if existingRecord == nil {
		logx.Errorf("[DeviceDownloadCallback] ❌ 未找到预记录: content_id=%d (task_id=%s)", req.ContentID, req.TaskID)
		return nil, fmt.Errorf("未找到该内容的下载任务(content_id=%d)，请确认已下发下载指令", req.ContentID)
	}

	logx.Infof("[DeviceDownloadCallback] ✅ 找到预记录: content_id=%d 当前状态=%s 总下载次数=%d",
		req.ContentID, existingRecord.LastStatus, existingRecord.TotalDownloads)

	finalStatus := "success"
	if req.Status == "fail" {
		finalStatus = "failed"
	}

	updateErr := statsRepo.UpdateDownloadResult(
		l.ctx,
		req.ContentID,
		finalStatus,
		req.ErrorMsg,
		"",
		0,
		0,
	)
	if updateErr != nil {
		logx.Errorf("[DeviceDownloadCallback] ❌ 更新状态失败: content_id=%d status=%s error=%v",
			req.ContentID, finalStatus, updateErr)
		return nil, fmt.Errorf("更新下载状态失败: %v", updateErr)
	}

	userDownloadsRepo := l.svcCtx.UserDownloadsRepo
	downloadRecord, findErr := userDownloadsRepo.FindByContentIDAndUserID(l.ctx, req.ContentID, 0)
	if findErr != nil {
		logx.Errorf("[DeviceDownloadCallback] ⚠️ 查询下载记录失败: content_id=%d error=%v", req.ContentID, findErr)
	} else if downloadRecord != nil {
		if statusErr := userDownloadsRepo.UpdateStatus(l.ctx, downloadRecord.ID, finalStatus); statusErr != nil {
			logx.Errorf("[DeviceDownloadCallback] ⚠️ 更新user_downloads状态失败: id=%d error=%v", downloadRecord.ID, statusErr)
		} else {
			logx.Infof("[DeviceDownloadCallback] ✅ user_downloads状态已更新: id=%d status=%s", downloadRecord.ID, finalStatus)
		}
	}

	var responseMessage string
	if req.Status == "success" {
		responseMessage = "✅ 下载成功"
		logx.Infof("[DeviceDownloadCallback] 🎉 下载成功！task_id=%s content_id=%d device_sn=%s",
			req.TaskID, req.ContentID, req.DeviceSN)
	} else {
		responseMessage = fmt.Sprintf("❌ 下载失败: %s", req.ErrorMsg)
		logx.Errorf("[DeviceDownloadCallback] 💥 下载失败！task_id=%s content_id=%d device_sn=%s 原因=%s",
			req.TaskID, req.ContentID, req.DeviceSN, req.ErrorMsg)
	}

	logx.Infof("[DeviceDownloadCallback] 📝 回调处理完成: task_id=%s content_id=%d 状态从 %s → %s",
		req.TaskID, req.ContentID, existingRecord.LastStatus, finalStatus)

	return &types.DeviceDownloadCallbackResp{
		Success: true,
		Message: responseMessage,
	}, nil
}

// GetDownloadStatus 查询下载状态（基于 content_download_stats 统计表）
func (l *DeviceDownloadLogic) GetDownloadStatus(contentID int64, userID int64) (*types.DeviceDownloadStatusQueryResp, error) {
	if contentID <= 0 {
		return nil, fmt.Errorf("content_id 必须为正整数")
	}

	statsRepo := l.svcCtx.ContentDownloadStatsRepo
	record, err := statsRepo.FindByContentID(l.ctx, contentID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("查询下载统计失败: %v", err)
	}

	if record == nil {
		return &types.DeviceDownloadStatusQueryResp{
			ContentID: contentID,
			Status:    "pending",
			Progress:  0,
			Message:   "暂无下载记录",
		}, nil
	}

	resp := &types.DeviceDownloadStatusQueryResp{
		ContentID: record.ContentID,
		Status:    "success",
		Progress:  100,
		Message:   fmt.Sprintf("累计下载 %d 次", record.TotalDownloads),
		CreatedAt: record.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if record.LastDownloadAt != nil {
		resp.FinishedAt = record.LastDownloadAt.Format("2006-01-02 15:04:05")
	}

	return resp, nil
}

// GetDownloadList 查询下载统计列表（基于 content_download_stats 表）
func (l *DeviceDownloadLogic) GetDownloadList(req *types.DeviceDownloadListReq, userID int64) (*types.DeviceDownloadListResp, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	query := `
		SELECT content_id, total_downloads, today_downloads, week_downloads, last_download_at, created_at, updated_at 
		FROM content_download_stats 
		ORDER BY total_downloads DESC 
		LIMIT $1 OFFSET $2
	`

	countQuery := `SELECT COUNT(*) FROM content_download_stats`

	var total int64
	countErr := l.svcCtx.DB.QueryRowContext(l.ctx, countQuery).Scan(&total)
	if countErr != nil {
		return nil, fmt.Errorf("查询总数失败: %v", countErr)
	}

	offset := (page - 1) * pageSize
	rows, queryErr := l.svcCtx.DB.QueryContext(l.ctx, query, pageSize, offset)
	if queryErr != nil {
		return nil, fmt.Errorf("查询列表失败: %v", queryErr)
	}
	defer rows.Close()

	var list []types.DeviceDownloadListItem
	for rows.Next() {
		var item types.DeviceDownloadListItem
		var lastDownloadAt *time.Time

		scanErr := rows.Scan(
			&item.ContentID,
			&item.FileSize,
			&item.Progress,
			&item.Status,
			&lastDownloadAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if scanErr != nil {
			continue
		}

		item.SongName = fmt.Sprintf("内容_%d", item.ContentID)
		item.ErrorMsg = fmt.Sprintf("总下载: %d 次", int(item.FileSize))

		if lastDownloadAt != nil {
			item.FinishedAt = lastDownloadAt.Format("2006-01-02 15:04:05")
		}

		list = append(list, item)
	}

	totalPages := int32(0)
	if total > 0 {
		totalPages = int32((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &types.DeviceDownloadListResp{
		Total:      total,
		List:       list,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalPages: totalPages,
	}, nil
}

// ContentServiceResponse 内容服务响应结构体
type ContentServiceResponse struct {
	Code int                `json:"code"`
	Data ContentServiceData `json:"data"`
}

// ContentServiceData 内容服务数据
type ContentServiceData struct {
	ContentID int64  `json:"content_id"`
	Title     string `json:"title"`
	AudioURL  string `json:"play_url"`
	VipLevel  int16  `json:"vip_level"`
}

// fetchContentFromContentService 从内容服务获取歌曲信息
func (l *DeviceDownloadLogic) fetchContentFromContentService(contentID int64) (*ContentServiceData, error) {
	contentServiceURL := fmt.Sprintf("http://localhost:8888/api/v1/content/%d", contentID)

	client := &http.Client{Timeout: 10 * time.Second}
	request, requestErr := http.NewRequestWithContext(l.ctx, http.MethodGet, contentServiceURL, nil)
	if requestErr != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", requestErr)
	}

	request.Header.Set("Accept", "application/json")

	response, doErr := client.Do(request)
	if doErr != nil {
		return nil, fmt.Errorf("调用内容服务失败: %w", doErr)
	}
	defer response.Body.Close()

	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应数据失败: %w", readErr)
	}

	logx.Debugf("[FetchContent] 内容服务响应: status=%d body=%s", response.StatusCode, string(bodyBytes))

	var contentResp ContentServiceResponse
	unmarshalErr := json.Unmarshal(bodyBytes, &contentResp)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("解析响应JSON失败: %w (body: %s)", unmarshalErr, string(bodyBytes))
	}

	if response.StatusCode != http.StatusOK || contentResp.Code != 0 && contentResp.Code != 200 {
		errorMsg := fmt.Sprintf("内容服务返回错误: HTTP %d, Code %d", response.StatusCode, contentResp.Code)
		return nil, fmt.Errorf("%s", errorMsg)
	}

	return &contentResp.Data, nil
}
