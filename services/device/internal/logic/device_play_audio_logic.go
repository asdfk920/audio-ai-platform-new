package logic

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

var validPlayAudioModes = map[string]bool{
	"sequential":  true,
	"list_loop":   true,
	"single_loop": true,
	"random":      true,
}

// DevicePlayAudioLogic 点播（URL 播放）指令
type DevicePlayAudioLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDevicePlayAudioLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DevicePlayAudioLogic {
	return &DevicePlayAudioLogic{ctx: ctx, svcCtx: svcCtx}
}

// DevicePlayAudio 点播：入库 + 在线 WebSocket 下发（与 Pause 链路一致）。
func (l *DevicePlayAudioLogic) DevicePlayAudio(req *types.DevicePlayAudioReq) (*types.DevicePlayAudioResp, error) {
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// sn 优先；兼容 JSON 字段 device_sn
	if strings.TrimSpace(req.Sn) == "" && strings.TrimSpace(req.DeviceSn) != "" {
		req.Sn = req.DeviceSn
	}

	if err := validateDevicePlayAudioReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.TrimSpace(req.Sn)
	audioURL := strings.TrimSpace(req.AudioURL)

	startPos := req.StartPos
	if startPos < 0 {
		startPos = 0
	}

	volume := req.Volume
	if volume <= 0 || volume > 100 {
		volume = 50
	}

	playMode := strings.ToLower(strings.TrimSpace(req.PlayMode))
	if playMode == "" {
		playMode = "sequential"
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
	params := map[string]interface{}{
		"task_id":      taskID,
		"content_id":   req.ContentID,
		"audio_url":    audioURL,
		"start_pos":    startPos,
		"volume":       volume,
		"play_mode":    playMode,
		"playback_cmd": "play_audio",
	}

	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "play_audio",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户点播(content_id=%d)", req.ContentID),
	})
	if err != nil {
		return nil, fmt.Errorf("创建点播指令失败: %v", err)
	}

	status, message := MapInstructionDispatchOutcome(result.Status, "点播", "")
	logx.Infof("设备点播指令已创建: user_id=%d sn=%s instruction_id=%d status=%s", userID, sn, result.InstructionID, status)

	return &types.DevicePlayAudioResp{
		TaskID:        taskID,
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
	}, nil
}

func validateDevicePlayAudioReq(req *types.DevicePlayAudioReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if err := validateReqSN(req.Sn); err != nil {
		return err
	}

	if req.ContentID <= 0 {
		return fmt.Errorf("content_id 必须为正整数")
	}

	audioURL := strings.TrimSpace(req.AudioURL)
	if audioURL == "" {
		return fmt.Errorf("audio_url 不能为空")
	}
	if _, err := url.ParseRequestURI(audioURL); err != nil {
		return fmt.Errorf("audio_url 格式无效")
	}

	if req.Volume < 0 || req.Volume > 100 {
		return fmt.Errorf("volume 必须在 0-100 范围内")
	}

	playMode := strings.ToLower(strings.TrimSpace(req.PlayMode))
	if playMode == "" {
		return nil
	}
	if !validPlayAudioModes[playMode] {
		return fmt.Errorf("无效的播放模式: %s (支持: sequential/list_loop/single_loop/random)", req.PlayMode)
	}

	return nil
}
