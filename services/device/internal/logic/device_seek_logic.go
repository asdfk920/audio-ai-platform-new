package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceSeekLogic 进度跳转指令
type DeviceSeekLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceSeekLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceSeekLogic {
	return &DeviceSeekLogic{ctx: ctx, svcCtx: svcCtx}
}

// DeviceSeek 进度条跳转：入库 + WebSocket 下发。
func (l *DeviceSeekLogic) DeviceSeek(req *types.DeviceSeekReq) (*types.DeviceSeekResp, error) {
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	if err := validateDeviceSeekReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

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

	corrID := strings.TrimSpace(req.TaskID)
	if corrID == "" {
		corrID = uuid.New().String()
	}

	params := map[string]interface{}{
		"task_id":        corrID,
		"position":       req.Position,
		"playback_cmd":   "seek",
	}

	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "seek",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("进度跳转 %.2fs", req.Position),
	})
	if err != nil {
		return nil, fmt.Errorf("创建跳转指令失败: %v", err)
	}

	status, message := MapInstructionDispatchOutcome(result.Status, "进度跳转", "")
	logx.Infof("设备跳转指令已创建: user_id=%d sn=%s instruction_id=%d position=%.2f", userID, sn, result.InstructionID, req.Position)

	return &types.DeviceSeekResp{
		TaskID:        corrID,
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
	}, nil
}

func validateDeviceSeekReq(req *types.DeviceSeekReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{16}$`, sn)
	if !matched {
		return fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	if req.Position < 0 {
		return fmt.Errorf("position 不能为负数")
	}

	return nil
}
