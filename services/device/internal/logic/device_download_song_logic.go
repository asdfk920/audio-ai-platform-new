package logic

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

var validDownloadQualities = map[string]bool{
	"standard": true,
	"higher":   true,
	"exhigh":   true,
	"lossless": true,
	"hires":    true,
}

// DeviceDownloadSongLogic 设备下载歌曲指令
type DeviceDownloadSongLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceDownloadSongLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceDownloadSongLogic {
	return &DeviceDownloadSongLogic{ctx: ctx, svcCtx: svcCtx}
}

// DownloadSong 创建下载歌曲指令并通过WebSocket下发
// 完整流程：参数校验 → 用户鉴权 → 设备验证 → 权限检查 → 创建指令 → WebSocket下发 → 返回结果
func (l *DeviceDownloadSongLogic) DownloadSong(req *types.DeviceDownloadSongReq, authz string) (*types.DeviceDownloadSongResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Download] 开始处理下载歌曲请求")
	logx.Infof("====================================")

	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	if err := validateDownloadSongReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.TrimSpace(req.Sn)
	contentID := req.ContentID

	quality := strings.ToLower(strings.TrimSpace(req.Quality))
	if quality == "" {
		quality = "standard"
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
		return nil, fmt.Errorf("无权限控制该设备（请先绑定）")
	}

	sourceURL := strings.TrimSpace(req.SourceURL)
	songName := strings.TrimSpace(req.SongName)

	if sourceURL == "" {
		baseURL := strings.TrimSpace(l.svcCtx.Config.Content.BaseURL)
		if baseURL == "" {
			return nil, fmt.Errorf("未配置 Content.BaseURL，无法根据 content_id 解析下载地址（可在请求中提供 source_url）")
		}
		if strings.TrimSpace(authz) == "" {
			return nil, fmt.Errorf("缺少 Authorization，无法拉取内容与下载权限")
		}
		title, playURL, canDL, err := fetchDownloadableContent(l.ctx, baseURL, authz, contentID)
		if err != nil {
			return nil, fmt.Errorf("获取内容详情失败: %w", err)
		}
		if !canDL {
			return nil, fmt.Errorf("当前账号对该内容无下载权限")
		}
		sourceURL = playURL
		if songName == "" {
			songName = title
		}
	}

	if songName == "" {
		songName = fmt.Sprintf("歌曲_%d", contentID)
	}
	if sourceURL == "" {
		return nil, fmt.Errorf("无法确定音频地址，内容详情缺少 play_url 或未提供 source_url")
	}

	taskID := uuid.New().String()

	enableDRM := true
	if req.EnableDRM != nil {
		enableDRM = *req.EnableDRM
	}

	params := map[string]interface{}{
		"task_id":    taskID,
		"content_id": contentID,
		"song_name":  songName,
		"quality":    quality,
		"source_url": sourceURL,
		"auth_token": authz,
		"enable_drm": enableDRM,
	}

	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "download_song",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下载歌曲(content_id=%d, quality=%s, drm=%v)", contentID, quality, enableDRM),
	})
	if err != nil {
		return nil, fmt.Errorf("创建下载指令失败: %v", err)
	}

	RegisterDownloadTaskForUserNotify(taskID, downloadTaskMeta{
		UserID:    userID,
		ContentID: contentID,
	})

	status, message := MapInstructionDispatchOutcome(result.Status, "下载", "")

	logx.Infof("====================================")
	logx.Infof("[Device Download] ✅ 下载指令创建成功")
	logx.Infof("   Task ID: %s", taskID)
	logx.Infof("   Instruction ID: %d", result.InstructionID)
	logx.Infof("   User ID: %d", userID)
	logx.Infof("   Device SN: %s", sn)
	logx.Infof("   Content ID: %d", contentID)
	logx.Infof("   Song Name: %s", songName)
	logx.Infof("   Quality: %s", quality)
	logx.Infof("   Enable DRM: %v (合规保护)", enableDRM)
	logx.Infof("   Status: %s", status)
	logx.Infof("   Message: %s", message)
	logx.Infof("====================================")

	return &types.DeviceDownloadSongResp{
		TaskID:        taskID,
		InstructionID: &result.InstructionID,
		Status:        status,
		Message:       message,
	}, nil
}

// CancelDownload 取消正在进行的下载任务
func (l *DeviceDownloadSongLogic) CancelDownload(taskID string, userID int64) error {
	logx.Infof("[Device Download] 用户取消下载任务: task_id=%s, user_id=%d", taskID, userID)

	deleteDownloadTaskMeta(taskID)

	logx.Infof("[Device Download] 任务已取消并清理资源: task_id=%s", taskID)
	return nil
}

// GetDownloadStatus 查询下载任务状态
func (l *DeviceDownloadSongLogic) GetDownloadStatus(taskID string, userID int64) (*types.DownloadStatusQueryResp, error) {
	meta, ok := loadDownloadTaskMeta(taskID)
	if !ok {
		return nil, fmt.Errorf("任务不存在或已过期: %s", taskID)
	}

	if meta.UserID != userID {
		return nil, fmt.Errorf("无权查看该任务状态")
	}

	resp := &types.DownloadStatusQueryResp{
		TaskID:    taskID,
		SongID:    meta.ContentID,
		Status:    "pending",
		Message:   "等待设备响应",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	logx.Infof("[Device Download] 查询任务状态: task_id=%s, status=%s", taskID, resp.Status)

	return resp, nil
}

// validateDownloadSongReq 校验下载请求参数（必填 sn + content_id）。
func validateDownloadSongReq(req *types.DeviceDownloadSongReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if err := validateReqSN(req.Sn); err != nil {
		return err
	}

	if req.ContentID <= 0 {
		return fmt.Errorf("content_id 必须为正整数")
	}

	quality := strings.ToLower(strings.TrimSpace(req.Quality))
	if quality == "" {
		return nil
	}

	if !validDownloadQualities[quality] {
		return fmt.Errorf("无效的音质等级: %s (支持: standard/higher/exhigh/lossless/hires)", req.Quality)
	}

	if req.SourceURL != "" {
		if _, err := url.ParseRequestURI(req.SourceURL); err != nil {
			return fmt.Errorf("source_url 格式无效")
		}
	}

	return nil
}
