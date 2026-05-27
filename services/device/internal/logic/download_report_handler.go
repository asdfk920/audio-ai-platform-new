package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DownloadReportHandler 下载状态上报处理器
// 处理设备通过WebSocket上报的下载进度和结果
type DownloadReportHandler struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDownloadReportHandler(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadReportHandler {
	return &DownloadReportHandler{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// HandleProgressReport 处理设备上报的下载进度
// 设备在下载过程中可选择性地上报进度信息
//
// 参数 report *types.WSDownloadProgressReport: 进度上报数据
// 返回 error: 错误信息（如果失败）
func (h *DownloadReportHandler) HandleProgressReport(report *types.WSDownloadProgressReport) error {
	if report == nil {
		return fmt.Errorf("进度上报数据不能为空")
	}

	logx.Infof("[DownloadProgress] 收到进度上报: task_id=%s, song_id=%d, progress=%.1f%%",
		report.TaskID, report.SongID, report.Percent)

	if report.Percent < 0 || report.Percent > 100 {
		return fmt.Errorf("进度百分比必须在0-100范围内")
	}

	logx.Infof("[DownloadProgress] 任务进度更新: task_id=%s, song_id=%d, percent=%.1f%%, downloaded=%d/%d bytes",
		report.TaskID, report.SongID, report.Percent,
		report.DownloadedSize, report.TotalSize)

	return nil
}

// HandleResultReport 处理设备上报的下载结果
// 设备下载完成或失败后必须上报最终结果
//
// 参数 report *types.WSDownloadResultReport: 结果上报数据
// 返回 error: 错误信息（如果失败）
func (h *DownloadReportHandler) HandleResultReport(report *types.WSDownloadResultReport) error {
	if report == nil {
		return fmt.Errorf("结果上报数据不能为空")
	}

	if report.Status != "success" && report.Status != "fail" {
		return fmt.Errorf("无效的状态值: %s (支持: success/fail)", report.Status)
	}

	if report.Status == "fail" && report.ErrorMsg == "" {
		return fmt.Errorf("下载失败时必须提供错误原因(error_msg)")
	}

	now := time.Now().Format(time.RFC3339)

	if report.Status == "success" {
		logx.Infof("[DownloadFinish] ✅ 下载成功: task_id=%s, song_id=%d, local_path=%s, size=%d bytes, duration=%dms",
			report.TaskID, report.SongID, report.LocalPath,
			report.FileSize, report.DurationMs)
	} else {
		logx.Errorf("[DownloadFinish] ❌ 下载失败: task_id=%s, song_id=%d, error=%s",
			report.TaskID, report.SongID, report.ErrorMsg)
	}

	err := h.saveDownloadResultToDB(report, now)
	if err != nil {
		logx.Errorf("[DownloadFinish] 保存下载记录失败: %v", err)
	}

	return nil
}

// saveDownloadResultToDB 将下载结果保存到数据库
// 实际应用中应该将下载任务记录持久化到数据库，便于：
//   - 前端查询下载历史
//   - 统计下载成功率
//   - 重试失败的下载任务
func (h *DownloadReportHandler) saveDownloadResultToDB(report *types.WSDownloadResultReport, timestamp string) error {

	logx.Infof("[DownloadDB] 准备保存下载记录: task_id=%s, song_id=%d, status=%s",
		report.TaskID, report.SongID, report.Status)

	recordJSON, _ := json.Marshal(map[string]interface{}{
		"task_id":     report.TaskID,
		"song_id":     report.SongID,
		"status":      report.Status,
		"error_msg":   report.ErrorMsg,
		"local_path":  report.LocalPath,
		"file_size":   report.FileSize,
		"duration_ms": report.DurationMs,
		"finished_at": timestamp,
	})

	logx.Debugf("[DownloadDB] 下载记录JSON: %s", string(recordJSON))

	return nil
}

// BuildDownloadInstruction 构造WebSocket下行下载指令
// 将下载指令转换为标准格式后下发给设备
//
// 参数:
//   - taskID: 任务ID
//   - songID: 歌曲ID
//   - songName: 歌曲名称
//   - downloadURL: CDN下载地址
//   - token: 设备鉴权Token（可选）
//
// 返回:
//   - []byte: JSON格式的指令字节
//   - error: 错误信息
func BuildDownloadInstruction(taskID string, songID int64, songName, downloadURL, token string) ([]byte, error) {
	instruction := types.WSSongDownloadInstruction{
		Cmd:         "download_song",
		TaskID:      taskID,
		SongID:      songID,
		SongName:    songName,
		DownloadURL: downloadURL,
		Token:       token,
		Timestamp:   time.Now().Unix(),
	}

	instructionJSON, err := json.Marshal(instruction)
	if err != nil {
		return nil, fmt.Errorf("序列化下载指令失败: %w", err)
	}

	logx.Infof("[BuildInstruction] 构造下载指令完成: task_id=%s, song_id=%d, url=%s...",
		taskID, songID, downloadURL[:min(50, len(downloadURL))])

	return instructionJSON, nil
}
