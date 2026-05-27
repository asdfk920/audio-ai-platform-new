package logic

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// downloadTaskMeta 下载任务与用户侧通知元数据（task_id ↔ 谁在等结果）
type downloadTaskMeta struct {
	UserID            int64
	UserDownloadRowID int64
	ContentID         int64
}

var (
	downloadTaskByID sync.Map // taskID(string) -> downloadTaskMeta
)

// RegisterDownloadTaskForUserNotify 指令已成功创建并下发后登记，便于设备 WS 上报结束时推送给 App
func RegisterDownloadTaskForUserNotify(taskID string, meta downloadTaskMeta) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || meta.UserID <= 0 {
		return
	}
	downloadTaskByID.Store(taskID, meta)
}

func loadDownloadTaskMeta(taskID string) (downloadTaskMeta, bool) {
	v, ok := downloadTaskByID.Load(strings.TrimSpace(taskID))
	if !ok {
		return downloadTaskMeta{}, false
	}
	m, ok := v.(downloadTaskMeta)
	return m, ok
}

func deleteDownloadTaskMeta(taskID string) {
	downloadTaskByID.Delete(strings.TrimSpace(taskID))
}

// --------- App 用户 WebSocket 连接池（同一用户仅保留最后一条连接）---------

type wsUserConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (dc *wsUserConn) WriteJSON(v interface{}) error {
	if dc == nil || dc.conn == nil {
		return fmt.Errorf("用户连接为空")
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.conn.WriteJSON(v)
}

var (
	userWsConnMap = make(map[string]*wsUserConn)
	userWsConnMu  sync.RWMutex
)

func registerUserWsConn(userID int64, dc *wsUserConn) {
	key := strconv.FormatInt(userID, 10)
	userWsConnMu.Lock()
	userWsConnMap[key] = dc
	userWsConnMu.Unlock()
}

func unregisterUserWsConn(userID int64, dc *wsUserConn) {
	key := strconv.FormatInt(userID, 10)
	userWsConnMu.Lock()
	cur, ok := userWsConnMap[key]
	if ok && cur == dc {
		delete(userWsConnMap, key)
	}
	userWsConnMu.Unlock()
}

// NotifyUserWsJSON 向该用户当前 App WebSocket 推送 JSON（可能被防火墙丢弃，失败仅打日志）
func NotifyUserWsJSON(userID int64, v interface{}) {
	if userID <= 0 {
		return
	}
	key := strconv.FormatInt(userID, 10)
	userWsConnMu.RLock()
	dc, ok := userWsConnMap[key]
	userWsConnMu.RUnlock()
	if !ok || dc == nil {
		logx.Infof("[UserWSNotify] skip push: user_id=%d offline", userID)
		return
	}
	if err := dc.WriteJSON(v); err != nil {
		logx.Errorf("[UserWSNotify] 推送失败 user_id=%d err=%v", userID, err)
	}
}
