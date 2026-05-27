package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// handleDeviceDownloadEvent 处理设备 JSON 中的 event=download_progress | download_finish（与 types 中 WSDownload* 对齐）
func (l *DeviceWsLogic) handleDeviceDownloadEvent(deviceKey string, raw map[string]interface{}) {
	ev, _ := raw["event"].(string)
	switch strings.TrimSpace(ev) {
	case "download_progress":
		l.handleDeviceDownloadProgress(deviceKey, raw)
	case "download_finish":
		l.handleDeviceDownloadFinish(deviceKey, raw)
	default:
		logx.Infof("[WS] 设备 %s 未知 download event=%q", deviceKey, ev)
	}
}

func (l *DeviceWsLogic) handleDeviceDownloadProgress(deviceKey string, raw map[string]interface{}) {
	b, _ := json.Marshal(raw)
	var report types.WSDownloadProgressReport
	if err := json.Unmarshal(b, &report); err != nil {
		logx.Errorf("[WS] download_progress 解析失败 device=%s err=%v body=%s", deviceKey, err, string(b))
		return
	}
	_ = NewDownloadReportHandler(l.ctx, l.svcCtx).HandleProgressReport(&report)

	if meta, ok := loadDownloadTaskMeta(report.TaskID); ok {
		NotifyUserWsJSON(meta.UserID, map[string]interface{}{
			"type":       "download_progress",
			"task_id":    report.TaskID,
			"content_id": meta.ContentID,
			"song_id":    report.SongID,
			"percent":    report.Percent,
		})
	}
}

func (l *DeviceWsLogic) handleDeviceDownloadFinish(deviceKey string, raw map[string]interface{}) {
	b, _ := json.Marshal(raw)
	var report types.WSDownloadResultReport
	if err := json.Unmarshal(b, &report); err != nil {
		logx.Errorf("[WS] download_finish 解析失败 device=%s err=%v body=%s", deviceKey, err, string(b))
		return
	}

	if _, err := ProcessDownloadFinishReport(l.ctx, l.svcCtx, report, deviceKey); err != nil {
		logx.Errorf("[WS] download_finish 处理失败 device=%s task_id=%s err=%v", deviceKey, report.TaskID, err)
	}
}

// ProcessDownloadFinishReport 统一处理设备下载完成/失败回调（WebSocket 与 HTTP 回调共用）。
// 完成后会更新统计/用户下载记录，并通过用户 WebSocket 推送 type=download_result 给前端。
func ProcessDownloadFinishReport(ctx context.Context, svcCtx *svc.ServiceContext, report types.WSDownloadResultReport, deviceKey string) (map[string]interface{}, error) {
	report.Status = normalizeDownloadReportStatus(report.Status)
	if report.Status == "fail" {
		report.ErrorMsg = strings.TrimSpace(report.ErrorMsg)
	} else {
		report.ErrorMsg = ""
	}

	taskID := strings.TrimSpace(report.TaskID)
	if taskID == "" {
		return nil, fmt.Errorf("task_id 不能为空")
	}
	meta, hasMeta := loadDownloadTaskMeta(taskID)
	contentID := report.SongID
	if hasMeta && meta.ContentID > 0 {
		contentID = meta.ContentID
	}

	if err := NewDownloadReportHandler(ctx, svcCtx).HandleResultReport(&report); err != nil {
		return nil, err
	}

	dh := svcDownloadFinishHelper{ctx: ctx, svcCtx: svcCtx}
	dbStatus := "success"
	if report.Status != "success" {
		dbStatus = "failed"
	}

	if err := dh.updateStatsAndUserRow(contentID, hasMeta, meta, report, dbStatus); err != nil {
		logx.Errorf("[DownloadFinish] 更新库失败 task_id=%s err=%v", taskID, err)
	}

	payload := map[string]interface{}{
		"type":        "download_result",
		"task_id":     taskID,
		"content_id":  contentID,
		"status":      dbStatus,
		"error_msg":   report.ErrorMsg,
		"local_path":  report.LocalPath,
		"file_size":   report.FileSize,
		"duration_ms": report.DurationMs,
		"message":     dh.resultMessage(report),
	}

	if hasMeta && taskID != "" {
		deleteDownloadTaskMeta(taskID)
		NotifyUserWsJSON(meta.UserID, payload)
		return payload, nil
	}

	logx.Infof("[DownloadFinish] 无监听用户任务 task_id=%s content_id=%d（可能非 App WS 链路）", taskID, contentID)
	return payload, nil
}

type svcDownloadFinishHelper struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func (h svcDownloadFinishHelper) resultMessage(report types.WSDownloadResultReport) string {
	if report.Status == "success" {
		return "下载成功"
	}
	return fmt.Sprintf("下载失败: %s", strings.TrimSpace(report.ErrorMsg))
}

func (h svcDownloadFinishHelper) updateStatsAndUserRow(contentID int64, hasMeta bool, meta downloadTaskMeta, report types.WSDownloadResultReport, dbStatus string) error {
	statsRepo := h.svcCtx.ContentDownloadStatsRepo
	if statsRepo != nil && contentID > 0 {
		errMsg := report.ErrorMsg
		if dbStatus != "failed" {
			errMsg = ""
		}
		if upErr := statsRepo.UpdateDownloadResult(
			h.ctx,
			contentID,
			dbStatus,
			errMsg,
			report.LocalPath,
			report.FileSize,
			report.DurationMs,
		); upErr != nil {
			logx.Errorf("[DownloadFinish] content_download_stats 更新失败 content_id=%d err=%v", contentID, upErr)
		}
	}

	if !hasMeta || h.svcCtx.UserDownloadsRepo == nil || meta.UserDownloadRowID <= 0 {
		return nil
	}
	return h.svcCtx.UserDownloadsRepo.UpdateStatus(h.ctx, meta.UserDownloadRowID, dbStatus)
}

func normalizeDownloadReportStatus(status string) string {
	s := strings.TrimSpace(strings.ToLower(status))
	switch s {
	case "ok", "completed", "done":
		return "success"
	case "failed", "error", "fail":
		return "fail"
	default:
		return s
	}
}
