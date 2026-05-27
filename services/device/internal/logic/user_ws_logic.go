package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// UserWsLogic App 用户 WebSocket：通过用户 JWT 连接，发送下载指令并接收进度/结果推送
type UserWsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserWsLogic 创建逻辑实例
func NewUserWsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserWsLogic {
	return &UserWsLogic{ctx: ctx, svcCtx: svcCtx}
}

// UserAppWs App 长连接：GET /ws/user ，握手使用与 HTTP 相同的用户 AccessSecret
func (l *UserWsLogic) UserAppWs(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(l.svcCtx.Config.Auth.AccessSecret)
	if secret == "" {
		http.Error(w, `{"code":500,"msg":"JWT secret 未配置"}`, http.StatusInternalServerError)
		return
	}
	tok := jwt.ExtractAppWSBearerToken(r)
	if tok == "" {
		http.Error(w, `{"code":401,"msg":"请在握手头携带 Authorization: Bearer <token> 或使用 ?token="}`, http.StatusUnauthorized)
		return
	}
	claims, err := jwtx.ParseAccessToken(secret, tok)
	if err != nil || claims == nil || claims.UserID <= 0 {
		logx.Errorf("[UserWS] JWT 校验失败: %v", err)
		http.Error(w, `{"code":401,"msg":"token 无效或已过期"}`, http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("[UserWS] Upgrade 失败 user_id=%d err=%v", userID, err)
		return
	}
	defer conn.Close()

	dc := &wsUserConn{conn: conn}
	registerUserWsConn(userID, dc)
	defer func() {
		unregisterUserWsConn(userID, dc)
		UnregisterAllDeviceSubscriptions(userID)
	}()

	_ = dc.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "已连接用户通道，可发送 {\"type\":\"subscribe\",\"device_sn\":\"<SN>\"} 或 {\"type\":\"download_song\",...}",
		"user_id": userID,
	})

	ctx := jwt.WithUserID(r.Context(), userID)
	authz := tok
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		authz = "Bearer " + authz
	}

	logx.Infof("[UserWS] App 已连接 user_id=%s", fmtInt(userID))

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			logx.Infof("[UserWS] user_id=%d 读结束或断开: %v", userID, err)
			break
		}
		msg, err := parseUserWSMessage(data)
		if err != nil {
			logx.Errorf("[UserWS] JSON 解析失败 user_id=%d raw=%q err=%v", userID, truncateStringForLog(string(data), 200), err)
			_ = dc.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "JSON 解析失败，请发送 UTF-8 文本帧 JSON，例如 {\"type\":\"subscribe\",\"device_sn\":\"AUSP2605000002Y2\"}",
			})
			continue
		}
		t := strings.TrimSpace(wsMsgType(msg))
		cmd := strings.TrimSpace(wsMsgCmd(msg))
		switch t {
		case "ping":
			_ = dc.WriteJSON(map[string]interface{}{"type": "pong"})
		case "download_song":
			l.handleUserDownloadSong(dc, ctx, authz, msg)
		case "subscribe":
			l.handleDeviceSubscribe(dc, userID, ctx, authz, msg)
		case "unsubscribe":
			l.handleDeviceUnsubscribe(dc, userID, msg)
		default:
			switch cmd {
			case "subscribe":
				l.handleDeviceSubscribe(dc, userID, ctx, authz, msg)
			case "unsubscribe":
				l.handleDeviceUnsubscribe(dc, userID, msg)
			default:
				_ = dc.WriteJSON(map[string]interface{}{
					"type":    "error",
					"message": wsUnknownCmdHint(t, cmd),
				})
			}
		}
	}
}

func fmtInt(v int64) string {
	return fmt.Sprintf("%d", v)
}

func wsReadInt64(msg map[string]interface{}, key string) (int64, bool) {
	raw, ok := msg[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch x := raw.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	case json.Number:
		i, err := x.Int64()
		return i, err == nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		var n int64
		_, err := fmt.Sscanf(s, "%d", &n)
		return n, err == nil && n > 0
	default:
		return 0, false
	}
}

func (l *UserWsLogic) handleUserDownloadSong(dc *wsUserConn, ctx context.Context, authz string, msg map[string]interface{}) {
	contentID, okC := wsReadInt64(msg, "content_id")
	deviceID, okD := wsReadInt64(msg, "device_id")
	if !okC || contentID <= 0 {
		_ = dc.WriteJSON(map[string]interface{}{"type": "download_error", "message": "缺少 content_id"})
		return
	}
	if (!okD || deviceID <= 0) && strings.TrimSpace(asString(msg["sn"])) == "" {
		_ = dc.WriteJSON(map[string]interface{}{"type": "download_error", "message": "请提供 device_id 或 sn"})
		return
	}

	req := types.DeviceDownloadSongReq{
		DeviceID:  deviceID,
		ContentID: contentID,
		SongName:  strings.TrimSpace(asString(msg["song_name"])),
		Quality:   strings.TrimSpace(asString(msg["quality"])),
		SourceURL: strings.TrimSpace(asString(msg["source_url"])),
		Sn:        strings.TrimSpace(asString(msg["sn"])),
	}

	dl := NewDeviceDownloadSongLogic(ctx, l.svcCtx)
	resp, err := dl.DownloadSong(&req, authz)
	if err != nil {
		_ = dc.WriteJSON(map[string]interface{}{
			"type":    "download_error",
			"message": err.Error(),
		})
		return
	}
	_ = dc.WriteJSON(map[string]interface{}{
		"type":            "download_accepted",
		"task_id":         resp.TaskID,
		"dispatch_status": resp.Status,
		"message":         resp.Message,
		"instruction_id":  resp.InstructionID,
	})
}

func (l *UserWsLogic) handleDeviceSubscribe(dc *wsUserConn, userID int64, ctx context.Context, authz string, msg map[string]interface{}) {
	deviceSN := strings.TrimSpace(wsStringField(msg, "device_sn", "sn"))
	if deviceSN == "" {
		_ = dc.WriteJSON(map[string]interface{}{
			"type":    "subscribe_error",
			"cmd":     "subscribe_error",
			"message": "缺少 device_sn 或 sn 参数",
		})
		return
	}

	token, _ := msg["token"].(string)
	if token == "" {
		token = authz
	}

	if err := l.verifyDeviceOwnership(ctx, userID, deviceSN, token); err != nil {
		logx.Errorf("[DeviceSub] 鉴权失败 user_id=%d device_sn=%s err=%v", userID, deviceSN, err)
		_ = dc.WriteJSON(map[string]interface{}{
			"cmd":   "subscribe_error",
			"error": err.Error(),
		})
		return
	}

	RegisterDeviceSubscription(userID, deviceSN, dc)

	store := l.svcCtx.GetRedisShadowStore()
	if store != nil {
		shadow, err := store.GetShadow(ctx, deviceSN)
		if err != nil {
			logx.Slowf("[DeviceSub] 查询设备影子失败 device_sn=%s err=%v", deviceSN, err)
		} else if shadow != nil {
			_ = dc.WriteJSON(map[string]interface{}{
				"cmd":       "subscribe_success",
				"device_sn": deviceSN,
				"data":      shadow,
			})
			logx.Infof("[DeviceSub] 订阅成功并推送当前状态 user_id=%d device_sn=%s", userID, deviceSN)
			return
		}
	}

	_ = dc.WriteJSON(map[string]interface{}{
		"cmd":       "subscribe_success",
		"device_sn": deviceSN,
	})

	logx.Infof("[DeviceSub] 订阅成功 user_id=%d device_sn=%s", userID, deviceSN)
}

func (l *UserWsLogic) handleDeviceUnsubscribe(dc *wsUserConn, userID int64, msg map[string]interface{}) {
	deviceSN := strings.TrimSpace(wsStringField(msg, "device_sn", "sn"))
	if deviceSN == "" {
		_ = dc.WriteJSON(map[string]interface{}{
			"type":    "unsubscribe_error",
			"cmd":     "unsubscribe_error",
			"message": "缺少 device_sn 或 sn 参数",
		})
		return
	}
	UnregisterDeviceSubscription(userID, deviceSN)

	_ = dc.WriteJSON(map[string]interface{}{
		"cmd":       "unsubscribe_success",
		"device_sn": deviceSN,
	})

	logx.Infof("[DeviceSub] 取消订阅成功 user_id=%d device_sn=%s", userID, deviceSN)
}

func (l *UserWsLogic) verifyDeviceOwnership(ctx context.Context, userID int64, deviceSN string, _ string) error {
	if l.svcCtx == nil || l.svcCtx.UserDeviceBindRepo == nil {
		return fmt.Errorf("系统错误：数据库未初始化")
	}
	ok, err := l.svcCtx.UserDeviceBindRepo.ExistsActiveByUserAndSN(ctx, userID, deviceSN)
	if err != nil {
		logx.Errorf("[DeviceSub] 查询设备绑定关系失败 user_id=%d device_sn=%s err=%v", userID, deviceSN, err)
		return fmt.Errorf("查询设备权限失败")
	}
	if !ok {
		return fmt.Errorf("无权访问该设备或设备不存在")
	}
	return nil
}

// parseUserWSMessage 解析用户 WS 文本帧（兼容 BOM、智能引号、cmd/type 混用等）
func parseUserWSMessage(data []byte) (map[string]interface{}, error) {
	raw := bytes.TrimSpace(data)
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty message")
	}
	s := normalizeWSJSONText(string(raw))
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(s), &msg); err != nil {
		var wrapped string
		if err2 := json.Unmarshal([]byte(s), &wrapped); err2 == nil {
			inner := strings.TrimSpace(normalizeWSJSONText(wrapped))
			if strings.HasPrefix(inner, "{") {
				if err3 := json.Unmarshal([]byte(inner), &msg); err3 == nil {
					return msg, nil
				}
			}
		}
		return nil, err
	}
	return msg, nil
}

func normalizeWSJSONText(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		var b strings.Builder
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line == "```" || strings.HasPrefix(line, "```") {
				continue
			}
			b.WriteString(line)
		}
		s = b.String()
	}
	return strings.NewReplacer(
		"\u201c", `"`, "\u201d", `"`,
		"\u2018", `'`, "\u2019", `'`,
		"\uff1a", ":",
		"\u00a0", " ",
	).Replace(s)
}

func wsMsgType(msg map[string]interface{}) string {
	if t := strings.TrimSpace(asString(msg["type"])); t != "" {
		return t
	}
	return strings.TrimSpace(asString(msg["Type"]))
}

func wsMsgCmd(msg map[string]interface{}) string {
	if c := strings.TrimSpace(asString(msg["cmd"])); c != "" {
		return c
	}
	return strings.TrimSpace(asString(msg["Cmd"]))
}

func wsStringField(msg map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(asString(msg[k])); v != "" {
			return v
		}
	}
	return ""
}

// wsUnknownCmdHint 未知指令时的可读提示（避免把下行 subscribe_success 当成上行命令）
func wsUnknownCmdHint(t, cmd string) string {
	key := strings.ToLower(strings.TrimSpace(cmd))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(t))
	}
	switch key {
	case "subscribe_success", "subscribe_error":
		return "subscribe_success 是服务端下行，订阅请发送 {\"type\":\"subscribe\",\"device_sn\":\"<SN>\"}"
	case "unsubscribe_success", "unsubscribe_error":
		return "请发送 {\"type\":\"unsubscribe\",\"device_sn\":\"<SN>\"} 取消订阅"
	}
	if t != "" && cmd != "" {
		return fmt.Sprintf("未知 type/cmd: %s / %s；订阅请用 type=subscribe", t, cmd)
	}
	if cmd != "" {
		return fmt.Sprintf("未知 cmd: %s；订阅请发送 {\"type\":\"subscribe\",\"device_sn\":\"<SN>\"}", cmd)
	}
	if t != "" {
		return fmt.Sprintf("未知 type: %s；订阅请发送 {\"type\":\"subscribe\",\"device_sn\":\"<SN>\"}", t)
	}
	return "未知消息；订阅请发送 {\"type\":\"subscribe\",\"device_sn\":\"<SN>\"}"
}
