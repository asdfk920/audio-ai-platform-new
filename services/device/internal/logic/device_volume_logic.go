package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

type DeviceVolumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceVolumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceVolumeLogic {
	return &DeviceVolumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceVolume 下发设备音量调节指令（统一接口）
//
// Business flow:
//  1. Parse JWT Token and extract user ID
//  2. Validate request parameters (SN format, TargetVolume range 0-100)
//  3. Query device info from database
//  4. Verify user permission (check user_device_bind table)
//  5. Check device online status (WebSocket connection)
//  6. Query current volume value
//  7. Construct volume control command message
//  8. Send command via WebSocket real-time delivery
//  9. Log the event and return result
//
// Security features:
//   - JWT Token authentication to prevent unauthorized access
//   - Device binding verification to ensure only bound users can control devices
//   - Strict volume range validation (0-100) to prevent illegal values
//   - WebSocket long connection for real-time communication
//   - Detailed logging for troubleshooting and audit
func (l *DeviceVolumeLogic) DeviceVolume(req *types.DeviceVolumeReq) (*types.DeviceVolumeResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Volume] Start processing volume adjustment...")
	logx.Infof("[Device Volume] Device SN: %s", req.Sn)
	if req.TargetVolume > 0 {
		logx.Infof("[Device Volume] Target volume: %d%%", req.TargetVolume)
	}
	logx.Infof("[Device Volume] Action type: %s", req.Action)

	// ========== Step 1: Permission Pre-validation ==========

	logx.Infof("\n[Device Volume] Step 1: Authentication - Parsing JWT Token...")

	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		logx.Errorf("❌ [Device Volume] Authentication failed: Token invalid or missing")
		return nil, fmt.Errorf("Please login first")
	}

	logx.Infof("✅ [Device Volume] Token parsed successfully")
	logx.Infof("   User ID: %d", userID)

	logx.Infof("\n[Device Volume] Step 2: Parameter validation...")

	if err := validateDeviceVolumeReq(req); err != nil {
		logx.Errorf("❌ [Device Volume] Parameter validation failed: %v", err)
		return nil, fmt.Errorf("Parameter validation failed: %v", err)
	}

	sn := normalizeSN(req.Sn)
	action := strings.ToLower(strings.TrimSpace(req.Action))
	targetVolume := req.TargetVolume

	logx.Infof("✅ [Device Volume] Parameter validation passed")
	logx.Infof("   SN (normalized): %s", sn)
	logx.Infof("   Action:         %s", action)
	logx.Infof("   TargetVolume:   %d (range: 0-100)", targetVolume)

	logx.Infof("\n[Device Volume] Step 3: Querying device info...")

	deviceInfo, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		logx.Errorf("❌ [Device Volume] Database query failed: %v", err)
		return nil, fmt.Errorf("Query device failed: %v", err)
	}
	if deviceInfo == nil || deviceInfo.ID <= 0 {
		logx.Errorf("❌ [Device Volume] Device not found: %s", sn)
		return nil, fmt.Errorf("Device not found: %s", sn)
	}

	logx.Infof("✅ [Device Volume] Device query successful")
	logx.Infof("   Device ID:  %d", deviceInfo.ID)
	logx.Infof("   Device SN:  %s", deviceInfo.Sn)
	logx.Infof("   User ID:    %d", userID)

	logx.Infof("\n[Device Volume] Step 4: Permission check - Verifying device binding...")
	logx.Infof("   Query params: user_id=%d, device_id=%d", userID, deviceInfo.ID)

	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		logx.Errorf("❌ [Device Volume] Binding query exception: %v", err)
		return nil, fmt.Errorf("Query binding relationship failed: %v", err)
	}

	if bindInfo == nil {
		logx.Errorf("❌ [Device Volume] Permission denied - No control permission!")
		logx.Errorf("   user_id:   %d", userID)
		logx.Errorf("   device_id: %d", deviceInfo.ID)
		logx.Errorf("   sn:        %s", sn)
		return nil, fmt.Errorf("No permission to control this device (please bind it in App first)")
	}

	logx.Infof("✅ [Device Volume] Binding record found - Permission verified!")
	logx.Infof("   Bind ID:    %d", bindInfo.ID)
	logx.Infof("   Bind UserID: %d", bindInfo.UserID)
	logx.Infof("   Bind DeviceID: %d", bindInfo.DeviceID)
	logx.Infof("   Bind SN:     %s", bindInfo.SN)
	logx.Infof("   Bind Status: %d (1=normal)", bindInfo.Status)

	// ========== Step 2: Command Delivery Flow ==========

	deviceIDStr := fmt.Sprintf("%d", deviceInfo.ID)

	logx.Infof("\n[Device Volume] Step 5: Checking device online status...")

	isOnline := IsDeviceOnline(deviceIDStr)
	if !isOnline {
		logx.Infof("\n[Device Volume] Device offline, caching instruction for auto-delivery when online...")
		logx.Infof("   Device ID: %d", deviceInfo.ID)
		logx.Infof("   Device SN: %s", sn)

		cacheManager := NewDeviceCmdCacheManager(l.ctx, l.svcCtx)
		cachedCmd := &CachedCommand{
			Action: action,
			Cmd:    "volume_control",
			Params: map[string]interface{}{
				"action":        action,
				"sn":            sn,
				"target_volume": targetVolume,
			},
			Timestamp:     time.Now().UnixMilli(),
			InstructionID: time.Now().UnixMilli(),
			UserID:        userID,
			IssuedAt:      time.Now().Format(time.RFC3339),
		}

		cacheErr := cacheManager.CacheCommand(sn, cachedCmd)
		if cacheErr != nil {
			logx.Errorf("[Device Volume] Cache failed: %v", cacheErr)
			return nil, fmt.Errorf("Device offline and cache failed: %v", cacheErr)
		}

		l.logVolumeEvent(deviceInfo.ID, sn, userID, targetVolume, 0, cachedCmd.InstructionID)

		logx.Infof("\n====================================")
		logx.Infof("🔊 [Device Volume] ✅✅✅ Volume instruction cached! ✅✅✅")
		logx.Infof("====================================")
		logx.Infof("  Device SN:      %s", sn)
		logx.Infof("  User ID:        %d", userID)
		logx.Infof("  Action type:    %s", action)
		logx.Infof("  Target volume:  %d%%", targetVolume)
		logx.Infof("  Status:         cached (cached)")
		logx.Infof("  Instruction ID: %d", cachedCmd.InstructionID)
		logx.Infof("  Issue time:     %s", time.Now().Format(time.RFC3339))
		logx.Infof("  Expected behavior:")
		logx.Infof("    ✅ Auto-deliver when device reconnects")
		logx.Infof("    ✅ Execute in order (FIFO)")
		logx.Infof("    ✅ Clear cache after execution")
		logx.Infof("    ✅ Sync status to frontend in real-time")
		logx.Infof("  Cache validity: %.1f hours", deviceCmdCacheTTL.Hours())
		logx.Infof("====================================")

		return &types.DeviceVolumeResp{
			InstructionID: cachedCmd.InstructionID,
			Status:        "cached",
			Message:       fmt.Sprintf("Device offline, volume instruction cached (ID:%d), will auto-deliver on reconnection, target volume: %d%%", cachedCmd.InstructionID, targetVolume),
			TargetVolume:  targetVolume,
			CurrentVolume: -1,
		}, nil
	}

	logx.Infof("✅ [Device Volume] Device online - WebSocket connection established")

	logx.Infof("\n[Device Volume] Step 6: Querying current volume...")

	currentVolume := l.queryCurrentVolume(sn)
	logx.Infof("   Current volume: %d%%", currentVolume)

	instructionID := time.Now().UnixMilli()

	volumeCmd := l.constructVolumeCommand(sn, action, instructionID, targetVolume, currentVolume)

	logx.Infof("\n[Device Volume] Step 7: Constructing volume control instruction...")
	logx.Infof("   type:           cmd")
	logx.Infof("   cmd:            volume_control")
	logx.Infof("   instruction_id: %d", instructionID)
	logx.Infof("   action:         %s", action)
	logx.Infof("   target_volume:  %d%%", targetVolume)
	logx.Infof("   current_volume: %d%%", currentVolume)
	logx.Infof("   timestamp:      %d", time.Now().UnixMilli())

	logx.Infof("\n[Device Volume] Step 8: Sending via WebSocket real-time...")

	sendErr := SendCmdToDevice(deviceIDStr, volumeCmd)
	if sendErr != nil {
		logx.Errorf("❌ [Device Volume] Failed to send volume instruction: %v", sendErr)
		return nil, fmt.Errorf("Failed to send volume instruction: %v", sendErr)
	}

	l.logVolumeEvent(deviceInfo.ID, sn, userID, targetVolume, currentVolume, instructionID)

	logx.Infof("\n====================================")
	logx.Infof("🔊 [Device Volume] ✅✅✅ Volume adjustment instruction delivered! ✅✅✅")
	logx.Infof("====================================")
	logx.Infof("  Device SN:       %s", sn)
	logx.Infof("  User ID:         %d", userID)
	logx.Infof("  Action type:     %s", action)
	logx.Infof("  Current volume:  %d%% → Target volume: %d%%", currentVolume, targetVolume)
	logx.Infof("  Instruction ID:  %d", instructionID)
	logx.Infof("  Delivery time:   %s", time.Now().Format(time.RFC3339))
	logx.Infof("  Expected behavior:")
	logx.Infof("    1. Device receives instruction, validates format and signature")
	logx.Infof("    2. Device parses target volume: %d%%", targetVolume)
	logx.Infof("    3. Device adjusts audio playback volume")
	logx.Infof("    4. Device saves volume to local config (prevent loss on reboot)")
	logx.Infof("    5. Device reports 'volume adjusted' status with current volume")
	logx.Infof("    6. Cloud updates device running status")
	logx.Infof("    7. Frontend syncs latest volume display: %d%%", targetVolume)
	logx.Infof("====================================")

	return &types.DeviceVolumeResp{
		InstructionID: instructionID,
		Status:        "delivered",
		Message:       fmt.Sprintf("Volume instruction delivered to device %s, adjusting from %d%% to %d%%", sn, currentVolume, targetVolume),
		TargetVolume:  targetVolume,
		CurrentVolume: currentVolume,
	}, nil
}

// validateDeviceVolumeReq Validate volume adjustment request parameters
func validateDeviceVolumeReq(req *types.DeviceVolumeReq) error {
	if req == nil {
		return fmt.Errorf("Request cannot be empty")
	}

	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("Device serial number cannot be empty")
	}

	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{16}$`, sn)
	if !matched {
		return fmt.Errorf("Device serial number format error, must be 16 alphanumeric characters")
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	validActions := map[string]bool{"set_volume": true, "volume_up": true, "volume_down": true}
	if !validActions[action] {
		return fmt.Errorf("Invalid action type, must be one of: set_volume, volume_up, volume_down")
	}

	if req.TargetVolume < 0 || req.TargetVolume > 100 {
		return fmt.Errorf("Target volume must be between 0-100 (current: %d)", req.TargetVolume)
	}

	return nil
}

// constructVolumeCommand Construct volume control instruction message
func (l *DeviceVolumeLogic) constructVolumeCommand(sn string, action string, instructionID int64, targetVolume int, currentVolume int) map[string]interface{} {
	return map[string]interface{}{
		"type":           "cmd",
		"cmd":            "volume_control",
		"instruction_id": instructionID,
		"sn":             sn,
		"action":         action,
		"timestamp":      time.Now().UnixMilli(),
		"params": map[string]interface{}{
			"action":         action,
			"sn":             sn,
			"target_volume":  targetVolume,
			"current_volume": currentVolume,
			"persist_config": true,
		},
		"meta": map[string]interface{}{
			"source":    "user_app",
			"issued_at": time.Now().Format(time.RFC3339),
			"version":   "1.0",
		},
	}
}

// queryCurrentVolume Query device's current volume (from Redis or default 50%)
func (l *DeviceVolumeLogic) queryCurrentVolume(sn string) int {
	if l.svcCtx.Redis == nil {
		return 50 // Default volume if Redis not configured
	}

	volumeStr, err := l.svcCtx.Redis.Get(l.ctx, "device:volume:"+sn).Result()
	if err != nil || volumeStr == "" {
		return 50 // Default volume
	}

	var volume int
	_, scanErr := fmt.Sscanf(volumeStr, "%d", &volume)
	if scanErr != nil {
		return 50
	}

	if volume < 0 || volume > 100 {
		return 50
	}

	return volume
}

// logVolumeEvent Log volume adjustment event details
func (l *DeviceVolumeLogic) logVolumeEvent(deviceID int64, sn string, userID int64, targetVolume int, currentVolume int, instructionID int64) {
	logx.Infof("[VOLUME_EVENT] ========================================")
	logx.Infof("[VOLUME_EVENT] 🔊 Volume Adjustment Event Recorded")
	logx.Infof("[VOLUME_EVENT] ========================================")
	logx.Infof("[VOLUME_EVENT] Device ID:       %d", deviceID)
	logx.Infof("[VOLUME_EVENT] Device SN:       %s", sn)
	logx.Infof("[VOLUME_EVENT] User ID:         %d", userID)
	logx.Infof("[VOLUME_EVENT] Action:          volume_control")
	logx.Infof("[VOLUME_EVENT] Current volume:  %d%%", currentVolume)
	logx.Infof("[VOLUME_EVENT] Target volume:   %d%%", targetVolume)
	logx.Infof("[VOLUME_EVENT] Instruction ID:  %d", instructionID)
	logx.Infof("[VOLUME_EVENT] Timestamp:       %s", time.Now().Format(time.RFC3339Nano))
	logx.Infof("[VOLUME_EVENT] ========================================")
}
