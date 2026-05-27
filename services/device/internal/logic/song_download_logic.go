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

// SongDownloadLogic 歌曲下载指令
type SongDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSongDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SongDownloadLogic {
	return &SongDownloadLogic{ctx: ctx, svcCtx: svcCtx}
}

// SongDownload 处理歌曲下载请求：
//   - 校验用户登录状态和权限
//   - 校验设备绑定关系
//   - 生成CDN下载地址（这里使用audio_url作为示例，实际应从内容服务获取）
//   - 构造下载指令并通过WebSocket下发到设备
//   - 返回任务ID供前端追踪状态
func (l *SongDownloadLogic) SongDownload(req *types.SongDownloadReq) (*types.SongDownloadResp, error) {
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	if err := validateSongDownloadReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	songName := strings.TrimSpace(req.SongName)
	if songName == "" {
		songName = fmt.Sprintf("歌曲_%d", req.SongID)
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

	taskID := uuid.New().String()

	statsRepo := l.svcCtx.ContentDownloadStatsRepo
	if statsErr := statsRepo.IncrementDownloads(l.ctx, req.SongID); statsErr != nil {
		logx.Errorf("[SongDownload] 更新下载统计失败: song_id=%d error=%v", req.SongID, statsErr)
		return nil, fmt.Errorf("更新下载统计失败: %v", statsErr)
	}
	logx.Infof("[SongDownload] ✅ 已写入下载统计到 content_download_stats: song_id=%d user_id=%d device_sn=%s",
		req.SongID, userID, sn)

	downloadURL, err := l.generateDownloadURL(req.SongID)
	if err != nil {
		logx.Errorf("[SongDownload] 生成下载地址失败: song_id=%d, error=%v", req.SongID, err)
		return nil, fmt.Errorf("生成下载地址失败: %v", err)
	}

	params := map[string]interface{}{
		"task_id":      taskID,
		"song_id":      req.SongID,
		"song_name":    songName,
		"download_url": downloadURL,
		"download_cmd": "download_song",
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
		Reason:          fmt.Sprintf("用户下载歌曲(song_id=%d, name=%s)", req.SongID, songName),
	})
	if err != nil {
		logx.Errorf("[SongDownload] 创建下载指令失败: user_id=%d sn=%s song_id=%d error=%v",
			userID, sn, req.SongID, err)
		return nil, fmt.Errorf("创建下载指令失败: %v", err)
	}

	status, message := MapInstructionDispatchOutcome(result.Status, "下载", "")
	logx.Infof("[SongDownload] 歌曲下载指令已创建: task_id=%s user_id=%d sn=%s song_id=%d instruction_id=%d status=%s",
		taskID, userID, sn, req.SongID, result.InstructionID, status)

	return &types.SongDownloadResp{
		TaskID:        taskID,
		SongID:        req.SongID,
		DeviceSN:      sn,
		Status:        status,
		Message:       message,
		InstructionID: &result.InstructionID,
	}, nil
}

// generateDownloadURL 根据song_id生成CDN下载地址
// 实际应用中应该：
//  1. 调用内容服务API获取真实的CDN地址
//  2. 或从数据库中查询音频文件的存储路径
//  3. 生成带签名的临时下载链接（防盗链）
//
// 这里提供基础实现，可根据实际需求扩展
func (l *SongDownloadLogic) generateDownloadURL(songID int64) (string, error) {

	baseURL := "https://your-cdn-domain.com/music"

	downloadPath := fmt.Sprintf("/songs/%d.mp3", songID)

	fullURL := baseURL + downloadPath

	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("URL解析失败: %w", err)
	}

	query := parsedURL.Query()
	query.Set("t", fmt.Sprintf("%d", time.Now().Unix()))
	parsedURL.RawQuery = query.Encode()

	logx.Infof("[SongDownload] 生成下载地址: song_id=%d, url=%s", songID, parsedURL.String())

	return parsedURL.String(), nil
}

func validateSongDownloadReq(req *types.SongDownloadReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if err := validateReqSN(req.Sn); err != nil {
		return err
	}

	if req.SongID <= 0 {
		return fmt.Errorf("song_id 必须为正整数")
	}

	if len([]rune(req.SongName)) > 200 {
		return fmt.Errorf("歌曲名称长度不能超过200字符")
	}

	return nil
}
