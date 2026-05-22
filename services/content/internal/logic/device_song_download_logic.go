package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/content/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// DeviceSongDownloadLogic 设备歌曲下载逻辑
type DeviceSongDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceSongDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceSongDownloadLogic {
	return &DeviceSongDownloadLogic{ctx: ctx, svcCtx: svcCtx}
}

// DownloadSong 处理设备歌曲下载请求
// 流程：
// 1. 校验参数和权限
// 2. 查询内容信息（获取音频URL）
// 3. 创建下载记录到数据库
// 4. 调用设备微服务下发下载指令
// 5. 返回任务ID
func (l *DeviceSongDownloadLogic) DownloadSong(req *types.DeviceSongDownloadReq, userID int64) (*types.DeviceSongDownloadResp, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	if sn == "" || len(sn) != 16 {
		return nil, fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	if req.ContentID <= 0 {
		return nil, fmt.Errorf("content_id 必须为正整数")
	}

	content, err := l.getContentByID(req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("内容不存在或已下架: %v", err)
	}

	userVipLevel := int16(0)
	vipLevel, err := l.getUserVipLevel(userID)
	if err != nil {
		logx.Errorf("[DeviceDownload] 获取用户会员等级失败: user_id=%d error=%v", userID, err)
	} else {
		userVipLevel = vipLevel
	}

	if content.VipLevel > userVipLevel {
		return nil, fmt.Errorf("该内容为VIP专属，请先升级会员")
	}

	taskID := uuid.New().String()
	songName := content.Title
	if songName == "" {
		songName = fmt.Sprintf("歌曲_%d", req.ContentID)
	}

	downloadRecord := &dao.DeviceSongDownload{
		TaskID:      taskID,
		UserID:      userID,
		DeviceSN:    sn,
		ContentID:   req.ContentID,
		SongName:    songName,
		DownloadURL: content.AudioURL,
		Status:      "pending",
		Progress:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	repo := repo.NewDeviceSongDownloadRepo(l.svcCtx.DB)
	if createErr := repo.Create(l.ctx, downloadRecord); createErr != nil {
		logx.Errorf("[DeviceDownload] 创建下载记录失败: task_id=%s error=%v", taskID, createErr)
		return nil, fmt.Errorf("创建下载记录失败: %v", createErr)
	}

	logx.Infof("[DeviceDownload] 下载记录已创建: task_id=%s user_id=%d device_sn=%s content_id=%d song_name=%s",
		taskID, userID, sn, req.ContentID, songName)

	deviceResult, callErr := l.callDeviceServiceDownload(sn, req.ContentID, songName, content.AudioURL, taskID)
	if callErr != nil {
		logx.Errorf("[DeviceDownload] 调用设备服务失败: task_id=%s error=%v", taskID, callErr)

		updateErr := repo.UpdateStatus(l.ctx, taskID, "failed", 0)
		if updateErr != nil {
			logx.Errorf("[DeviceDownload] 更新状态失败: task_id=%s error=%v", taskID, updateErr)
		}

		return &types.DeviceSongDownloadResp{
			TaskID:    taskID,
			ContentID: req.ContentID,
			DeviceSN:  sn,
			Status:    "failed",
			Message:   fmt.Sprintf("下载指令下发失败: %v", callErr),
		}, nil
	}

	if deviceResult.InstructionID != nil {
		updateErr := repo.UpdateInstructionID(l.ctx, taskID, *deviceResult.InstructionID)
		if updateErr != nil {
			logx.Slowf("[DeviceDownload] 更新指令ID失败: task_id=%s error=%v", taskID, updateErr)
		}
	}

	status := "sent"
	message := "下载指令已成功下发至设备"
	if deviceResult.Status == "cached" {
		status = "pending"
		message = "设备离线，指令已缓存，设备上线后将自动执行"
	}

	updateErr := repo.UpdateStatus(l.ctx, taskID, status, 0)
	if updateErr != nil {
		logx.Slowf("[DeviceDownload] 更新状态失败: task_id=%s error=%v", taskID, updateErr)
	}

	logx.Infof("[DeviceDownload] 歌曲下载流程完成: task_id=%s status=%s message=%s",
		taskID, status, message)

	return &types.DeviceSongDownloadResp{
		TaskID:    taskID,
		ContentID: req.ContentID,
		DeviceSN:  sn,
		Status:    status,
		Message:   message,
	}, nil
}

// HandleDownloadCallback 处理设备微服务的下载结果回调
// 设备下载完成后，设备微服务调用此接口通知内容微服务更新状态
func (l *DeviceSongDownloadLogic) HandleDownloadCallback(req *types.DeviceDownloadCallbackReq) (*types.DeviceDownloadCallbackResp, error) {
	if req.TaskID == "" {
		return nil, fmt.Errorf("task_id 不能为空")
	}
	if req.Status != "success" && req.Status != "fail" {
		return nil, fmt.Errorf("status 必须为 success 或 fail")
	}
	if req.Status == "fail" && req.ErrorMsg == "" {
		return nil, fmt.Errorf("下载失败时必须提供错误原因(error_msg)")
	}

	repo := repo.NewDeviceSongDownloadRepo(l.svcCtx.DB)

	record, findErr := repo.FindByTaskID(l.ctx, req.TaskID)
	if findErr != nil {
		logx.Errorf("[DeviceDownloadCallback] 查询记录失败: task_id=%s error=%v", req.TaskID, findErr)
		return nil, fmt.Errorf("查询下载记录失败")
	}
	if record == nil {
		logx.Errorf("[DeviceDownloadCallback] 记录不存在: task_id=%s", req.TaskID)
		return nil, fmt.Errorf("下载记录不存在")
	}

	localPath := ""
	fileSize := int64(0)
	durationMs := int64(0)

	if req.Status == "success" {
		localPath = req.LocalPath
		fileSize = req.FileSize
		durationMs = req.DurationMs

		logx.Infof("[DeviceDownloadCallback] ✅ 下载成功: task_id=%s content_id=%d local_path=%s size=%d duration=%dms",
			req.TaskID, req.ContentID, localPath, fileSize, durationMs)
	} else {
		logx.Errorf("[DeviceDownloadCallback] ❌ 下载失败: task_id=%s content_id=%d error=%s",
			req.TaskID, req.ContentID, req.ErrorMsg)
	}

	updateErr := repo.UpdateResult(
		l.ctx,
		req.TaskID,
		req.Status,
		req.ErrorMsg,
		localPath,
		fileSize,
		durationMs,
	)
	if updateErr != nil {
		logx.Errorf("[DeviceDownloadCallback] 更新结果失败: task_id=%s error=%v", req.TaskID, updateErr)
		return nil, fmt.Errorf("更新下载结果失败: %v", updateErr)
	}

	message := "下载成功"
	if req.Status == "fail" {
		message = fmt.Sprintf("下载失败: %s", req.ErrorMsg)
	}

	logx.Infof("[DeviceDownloadCallback] 回调处理完成: task_id=%s status=%s", req.TaskID, req.Status)

	return &types.DeviceDownloadCallbackResp{
		Success: true,
		Message: message,
	}, nil
}

// GetDownloadStatus 查询下载状态
func (l *DeviceSongDownloadLogic) GetDownloadStatus(taskID string, userID int64) (*types.DeviceDownloadStatusQueryResp, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id 不能为空")
	}

	repo := repo.NewDeviceSongDownloadRepo(l.svcCtx.DB)
	record, err := repo.FindByTaskID(l.ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}
	if record == nil {
		return nil, fmt.Errorf("下载记录不存在")
	}

	if record.UserID != userID {
		return nil, fmt.Errorf("无权查看此下载记录")
	}

	statusMessage := map[string]string{
		"pending":     "等待处理",
		"downloading": "下载中",
		"success":     "下载完成 ✅",
		"failed":      "下载失败 ❌",
		"cancelled":   "已取消",
	}

	finishedAt := ""
	if record.FinishedAt != nil {
		finishedAt = record.FinishedAt.Format("2006-01-02 15:04:05")
	}

	resp := &types.DeviceDownloadStatusQueryResp{
		TaskID:     record.TaskID,
		ContentID:  record.ContentID,
		SongName:   record.SongName,
		DeviceSN:   record.DeviceSN,
		Status:     record.Status,
		Progress:   record.Progress,
		Message:    statusMessage[record.Status],
		CreatedAt:  record.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  record.UpdatedAt.Format("2006-01-02 15:04:05"),
		FinishedAt: finishedAt,
	}

	return resp, nil
}

// GetDownloadList 查询用户的设备下载历史列表（支持模糊查询）
func (l *DeviceSongDownloadLogic) GetDownloadList(req *types.DeviceDownloadListReq, userID int64) (*types.DeviceDownloadListResp, error) {
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

	repo := repo.NewDeviceSongDownloadRepo(l.svcCtx.DB)

	songName := strings.TrimSpace(req.SongName)
	artist := strings.TrimSpace(req.Artist)
	deviceSN := strings.TrimSpace(req.DeviceSN)
	status := strings.TrimSpace(req.Status)

	logx.Infof("[DownloadList] 查询下载记录: user_id=%d page=%d page_size=%d song_name=%q artist=%q sn=%q status=%q",
		userID, page, pageSize, songName, artist, deviceSN, status)

	records, total, err := repo.ListByUserWithFilter(l.ctx, userID, page, pageSize, songName, artist, deviceSN, status)
	if err != nil {
		logx.Errorf("[DownloadList] 查询失败: user_id=%d error=%v", userID, err)
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}

	logx.Infof("[DownloadList] 查询结果: user_id=%d total=%d records_len=%d", userID, total, len(records))
	if len(records) > 0 {
		for i, r := range records {
			logx.Infof("[DownloadList] 记录%d: id=%d content_id=%d status=%q song_name=%q", i+1, r.ID, r.ContentID, r.LastStatus, r.SongName)
		}
	} else {
		logx.Infof("[DownloadList] 无匹配记录 user_id=%d：请确认 (1) JWT 用户与库中 user_downloads.user_id / device_song_downloads.user_id 一致 (2) content 服务连接的库与你在 Navicat 里看的是同一实例（见 etc Database.DataSource）", userID)
	}

	totalPages := int32(0)
	if total > 0 {
		totalPages = int32((total + int64(pageSize) - 1) / int64(pageSize))
	}

	list := make([]types.DeviceDownloadListItem, 0, len(records))
	for _, record := range records {
		item := types.DeviceDownloadListItem{
			DownloadID:  record.ID,
			ContentID:   record.ContentID,
			SongName:    record.SongName,
			Artist:      record.Artist,
			CoverURL:    record.CoverURL,
			DurationSec: record.DurationSec,
			Status:      record.LastStatus,
			LocalPath:   record.LastLocalPath,
			FileSize:    record.LastFileSize,
			ErrorMsg:    record.LastErrorMsg,
		}

		if record.LastDownloadAt != "" {
			item.DownloadTime = record.LastDownloadAt
		}

		list = append(list, item)
	}

	logx.Infof("[DownloadList] 查询成功: user_id=%d total=%d 当前页=%d 条数=%d",
		userID, total, page, len(list))

	return &types.DeviceDownloadListResp{
		Total:      total,
		List:       list,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalPages: totalPages,
	}, nil
}

// DeviceServiceResponse 设备服务响应结构体
type DeviceServiceResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data,omitempty"`
}

// DeviceServiceDownloadData 设备服务下载数据
type DeviceServiceDownloadData struct {
	TaskID        string `json:"task_id"`
	SongID        int64  `json:"song_id"`
	DeviceSN      string `json:"device_sn"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	InstructionID *int64 `json:"instruction_id,omitempty"`
}

// callDeviceServiceDownload 调用设备微服务的下载接口
func (l *DeviceSongDownloadLogic) callDeviceServiceDownload(deviceSN string, contentID int64, songName, downloadURL, taskID string) (*DeviceServiceDownloadData, error) {
	deviceServiceURL := "http://localhost:8002/api/v1/song/download"

	payload := map[string]interface{}{
		"sn":        deviceSN,
		"song_id":   contentID,
		"song_name": songName,
	}

	jsonData, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, fmt.Errorf("序列化请求数据失败: %w", marshalErr)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	request, requestErr := http.NewRequestWithContext(l.ctx, http.MethodPost, deviceServiceURL, bytes.NewBuffer(jsonData))
	if requestErr != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", requestErr)
	}

	request.Header.Set("Content-Type", "application/json")

	response, doErr := client.Do(request)
	if doErr != nil {
		return nil, fmt.Errorf("调用设备服务失败: %w", doErr)
	}
	defer response.Body.Close()

	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应数据失败: %w", readErr)
	}

	logx.Debugf("[DeviceDownload] 设备服务响应: status=%d body=%s", response.StatusCode, string(bodyBytes))

	var deviceResp DeviceServiceResponse
	unmarshalErr := json.Unmarshal(bodyBytes, &deviceResp)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("解析响应JSON失败: %w (body: %s)", unmarshalErr, string(bodyBytes))
	}

	if response.StatusCode != http.StatusOK || deviceResp.Code != 200 {
		errorMsg := deviceResp.Msg
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("设备服务返回错误: HTTP %d, Code %d", response.StatusCode, deviceResp.Code)
		}
		return nil, fmt.Errorf("%s", errorMsg)
	}

	if deviceResp.Data == nil || len(deviceResp.Data) == 0 || string(deviceResp.Data) == "null" {
		return &DeviceServiceDownloadData{
			TaskID:  taskID,
			Status:  "sent",
			Message: "指令下发成功",
		}, nil
	}

	var downloadData DeviceServiceDownloadData
	dataUnmarshalErr := json.Unmarshal(deviceResp.Data, &downloadData)
	if dataUnmarshalErr != nil {
		logx.Slowf("[DeviceDownload] 解析下载数据失败，使用默认值: error=%v body=%s", dataUnmarshalErr, string(deviceResp.Data))
		return &DeviceServiceDownloadData{
			TaskID:  taskID,
			Status:  "sent",
			Message: "指令下发成功",
		}, nil
	}

	downloadData.TaskID = taskID
	return &downloadData, nil
}

// getContentByID 根据内容ID获取信息
func (l *DeviceSongDownloadLogic) getContentByID(contentID int64) (*dao.ContentCatalog, error) {
	if l.svcCtx.DB == nil {
		return nil, fmt.Errorf("数据库未就绪")
	}

	var content dao.ContentCatalog
	err := l.svcCtx.DB.Where("id = ? AND status = ? AND is_deleted = ?", contentID, 1, 0).First(&content).Error
	if err != nil {
		return nil, fmt.Errorf("内容不存在或已下架")
	}
	return &content, nil
}

// getUserVipLevel 获取用户VIP等级
func (l *DeviceSongDownloadLogic) getUserVipLevel(userID int64) (int16, error) {
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
