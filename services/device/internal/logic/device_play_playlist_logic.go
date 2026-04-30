package logic

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/zeromicro/go-zero/core/logx"
)

// DevicePlayPlaylistLogic 设备播放歌单指令逻辑
// 处理用户通过 App 向设备下发播放歌单指令的业务逻辑
// 支持在线设备立即下发，离线设备缓存等待上线
// 包含用户身份验证、设备权限校验、歌单查询、歌曲列表查询等

type DevicePlayPlaylistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDevicePlayPlaylistLogic 创建设备播放歌单指令逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DevicePlayPlaylistLogic: 设备播放歌单指令逻辑实例
func NewDevicePlayPlaylistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DevicePlayPlaylistLogic {
	return &DevicePlayPlaylistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DevicePlayPlaylist 下发设备播放歌单指令
// 接收播放歌单指令请求，验证用户权限和设备状态，查询歌单和歌曲列表，下发播放歌单指令
// 参数 req *types.DevicePlayPlaylistReq: 播放歌单指令请求
// 返回 *types.DevicePlayPlaylistResp: 播放歌单指令响应
// 返回 error: 错误信息
func (l *DevicePlayPlaylistLogic) DevicePlayPlaylist(req *types.DevicePlayPlaylistReq) (*types.DevicePlayPlaylistResp, error) {
	// 1. 校验 Token 是否存在
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDevicePlayPlaylistReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	action := strings.ToLower(strings.TrimSpace(req.Action))
	playlistID := strings.TrimSpace(req.Params.PlaylistID)
	startIndex := req.Params.StartIndex
	volume := req.Params.Volume

	// 3. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	// 4. 验证用户权限：查询 user_device_bind 绑定表
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("无权限控制该设备")
	}

	// 5. 查询歌单信息
	playlistInfo, err := l.queryPlaylistInfo(playlistID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询歌单失败: %v", err)
	}

	// 6. 查询歌曲列表
	songs, totalCount, err := l.queryPlaylistSongs(playlistID)
	if err != nil {
		return nil, fmt.Errorf("查询歌曲列表失败: %v", err)
	}

	// 7. 构造播放歌单指令参数
	params := map[string]interface{}{
		"action":     action,
		"playlist_id": playlistID,
		"playlist_name": playlistInfo.Name,
		"start_index": startIndex,
		"total_count": totalCount,
		"songs":       songs,
	}

	if volume > 0 {
		params["volume"] = volume
	}

	// 8. 通过 commandsvc 创建并下发播放歌单指令
	// commandsvc 内部会：
	//   - 检查设备在线状态（Redis + MySQL）
	//   - 在线时通过 MQTT 立即下发
	//   - 离线时缓存为 pending 状态，设备上线后自动推送
	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "play_playlist",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下发播放歌单指令: action=%s, playlist_id=%s, total_count=%d", action, playlistID, totalCount),
	})
	if err != nil {
		return nil, fmt.Errorf("创建播放歌单指令失败: %v", err)
	}

	// 9. 组装响应
	status := "cached"
	message := "设备离线，指令已缓存，设备上线后将自动执行"
	if result.Status == "dispatched" || result.Status == "delivered" {
		status = "delivered"
		message = "播放歌单指令已下发"
	}

	// 10. 组装歌单信息
	playlistResp := &types.DevicePlaylistInfo{
		ID:         playlistID,
		Name:       playlistInfo.Name,
		TotalCount: totalCount,
	}

	// 如果有歌曲，添加第一首歌曲信息
	if len(songs) > 0 {
		firstSong := songs[0]
		playlistResp.FirstSong = &types.SongInfo{
			SongName: firstSong["song_name"].(string),
			Artist:   firstSong["artist"].(string),
			Duration: firstSong["duration"].(int),
		}
	}

	logx.Infof("设备播放歌单指令已下发: user_id=%d, sn=%s, action=%s, playlist_id=%s, total_count=%d, instruction_id=%d, status=%s",
		userID, sn, action, playlistID, totalCount, result.InstructionID, status)

	return &types.DevicePlayPlaylistResp{
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
		Playlist:      playlistResp,
	}, nil
}

// queryPlaylistInfo 查询歌单信息
// 参数 playlistID string: 歌单ID
// 参数 userID int64: 用户ID
// 返回 *model.Playlist: 歌单信息
// 返回 error: 查询错误
func (l *DevicePlayPlaylistLogic) queryPlaylistInfo(playlistID string, userID int64) (*model.Playlist, error) {
	// 将字符串playlistID转换为int64
	playlistIDInt, err := strconv.ParseInt(playlistID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("歌单ID格式错误: %v", err)
	}

	// 查询歌单信息（这里简化处理，实际应该查询user_playlists表）
	// 由于当前项目结构中没有直接的user_playlists表查询方法，我们使用PlaylistRepo查询
	playlist, err := l.svcCtx.PlaylistRepo.FindByUserId(l.ctx, userID, model.PlaylistTypeCustom)
	if err != nil {
		return nil, fmt.Errorf("查询歌单失败: %v", err)
	}
	if playlist == nil {
		return nil, fmt.Errorf("歌单不存在或已被删除")
	}

	// 检查歌单ID是否匹配
	if playlist.ID != playlistIDInt {
		return nil, fmt.Errorf("歌单不存在或不属于当前用户")
	}

	return playlist, nil
}

// queryPlaylistSongs 查询歌单歌曲列表
// 参数 playlistID string: 歌单ID
// 返回 []map[string]interface{}: 歌曲列表
// 返回 int: 歌曲总数
// 返回 error: 查询错误
func (l *DevicePlayPlaylistLogic) queryPlaylistSongs(playlistID string) ([]map[string]interface{}, int, error) {
	// 将字符串playlistID转换为int64
	playlistIDInt, err := strconv.ParseInt(playlistID, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("歌单ID格式错误: %v", err)
	}

	// 查询歌单项
	playlistItems, err := l.svcCtx.PlaylistItemRepo.FindByPlaylistId(l.ctx, playlistIDInt)
	if err != nil {
		return nil, 0, fmt.Errorf("查询歌单项失败: %v", err)
	}

	if len(playlistItems) == 0 {
		return nil, 0, fmt.Errorf("歌单为空")
	}

	// 提取音频ID列表
	audioIDs := make([]int64, 0, len(playlistItems))
	for _, item := range playlistItems {
		audioIDs = append(audioIDs, item.AudioID)
	}

	// 查询音频资源信息
	audioMap, err := l.svcCtx.AudioResourceRepo.FindByIds(l.ctx, audioIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("查询音频资源失败: %v", err)
	}

	// 组装歌曲列表
	songs := make([]map[string]interface{}, 0, len(playlistItems))
	for _, item := range playlistItems {
		audio, exists := audioMap[item.AudioID]
		if !exists {
			continue // 跳过不存在的音频
		}

		song := map[string]interface{}{
			"song_id":   audio.ID,
			"song_name": audio.Title,
			"artist":    "未知歌手", // 实际项目中应该从音频资源中获取歌手信息
			"duration":  audio.Duration,
			"media_url": audio.PlayUrl,
		}
		songs = append(songs, song)
	}

	return songs, len(songs), nil
}

// validateDevicePlayPlaylistReq 校验播放歌单指令请求参数
// 参数 req *types.DevicePlayPlaylistReq: 播放歌单指令请求
// 返回 error: 校验错误
func validateDevicePlayPlaylistReq(req *types.DevicePlayPlaylistReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	// 校验 SN 格式：16位字母数字组合
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}
	
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{16}$`, sn)
	if !matched {
		return fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	// 校验 action 参数
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "play_playlist" {
		return fmt.Errorf("操作类型必须为 play_playlist")
	}

	// 校验 params 参数
	if req.Params.PlaylistID == "" {
		return fmt.Errorf("歌单ID不能为空")
	}

	// 校验 start_index 参数
	if req.Params.StartIndex < 0 {
		return fmt.Errorf("起始播放索引不能小于0")
	}

	// 校验 volume 参数
	if req.Params.Volume < 0 || req.Params.Volume > 100 {
		return fmt.Errorf("音量必须在0-100范围内")
	}

	return nil
}