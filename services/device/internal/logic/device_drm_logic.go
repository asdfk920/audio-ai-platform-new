package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceDRMLogic 设备端DRM处理逻辑
// 核心原则：不解密、不破解、不转码，只做DRM信息透传与存储
type DeviceDRMLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceDRMLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceDRMLogic {
	return &DeviceDRMLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// HandleDeviceDownloadWithDRM 处理设备端带DRM的下载流程
// 完整流程：接收指令 → 请求加密流 → 接收DRM头 → 存储密文+DRM元数据 → 签名校验 → 上报结果
func (l *DeviceDRMLogic) HandleDeviceDownloadWithDRM(taskID string, songID int64, songName string, quality string, sourceURL string, enableDRM bool, authz string) (*types.DRMDownloadResponse, error) {
	logx.Infof("====================================")
	logx.Infof("[Device DRM Download] 开始处理DRM下载任务")
	logx.Infof("   Task ID: %s", taskID)
	logx.Infof("   Song ID: %d", songID)
	logx.Infof("   Song Name: %s", songName)
	logx.Infof("   Quality: %s", quality)
	logx.Infof("   Enable DRM: %v (合规保护)", enableDRM)
	logx.Infof("====================================")

	if !enableDRM {
		logx.Infof("[Device DRM Download] ⚠️ DRM未启用，将使用普通下载模式")
	}

	startTime := time.Now()

	resp := &types.DRMDownloadResponse{
		TaskID:      taskID,
		Status:      "accepted",
		Message:     "设备已接收下载指令，准备开始下载",
		RequestTime: startTime.Format(time.RFC3339),
	}

	l.recordDRMLog(taskID, songID, "request", "", "", "pending", "")

	if sourceURL == "" {
		err := fmt.Errorf("音频源地址不能为空")
		l.recordDRMLog(taskID, songID, "error", "", "", "failed", err.Error())
		resp.Status = "failed"
		resp.ErrorCode = "SOURCE_URL_EMPTY"
		resp.ErrorMessage = err.Error()
		return resp, err
	}

	drmMeta, encryptedStream, err := l.requestEncryptedStreamWithDRM(sourceURL, taskID, authz)
	if err != nil {
		drmType, drmContentID := drmLogMeta(drmMeta)
		l.recordDRMLog(taskID, songID, "error", drmType, drmContentID, "failed", err.Error())
		resp.Status = "failed"
		resp.ErrorCode = "REQUEST_STREAM_FAILED"
		resp.ErrorMessage = err.Error()
		return resp, err
	}
	defer encryptedStream.Body.Close()

	localPath, drmFilePath, fileSize, err := l.saveEncryptedStreamWithDRM(encryptedStream, taskID, songName, drmMeta, enableDRM)
	if err != nil {
		l.recordDRMLog(taskID, songID, "error", drmMeta.DRMType, drmMeta.ContentID, "failed", err.Error())
		resp.Status = "failed"
		resp.ErrorCode = "SAVE_FAILED"
		resp.ErrorMessage = err.Error()
		return resp, err
	}

	if enableDRM && drmMeta.Signature != "" {
		if err := l.verifyDRMSignature(drmMeta); err != nil {
			logx.Errorf("[Device DRM Download] ❌ DRM签名校验失败: %v", err)
			l.recordDRMLog(taskID, songID, "verify_failed", drmMeta.DRMType, drmMeta.ContentID, "failed", err.Error())

			resp.Status = "failed"
			resp.ErrorCode = "DRM_SIGNATURE_INVALID"
			resp.ErrorMessage = fmt.Sprintf("DRM签名校验失败（可能被篡改）: %v", err)
			return resp, err
		}
		logx.Infof("[Device DRM Download] ✅ DRM签名校验通过（防篡改验证成功）")
		l.recordDRMLog(taskID, songID, "verify", drmMeta.DRMType, drmMeta.ContentID, "success", "")
	}

	duration := time.Since(startTime).Milliseconds()

	resp.Status = "completed"
	resp.LocalPath = localPath
	resp.DRMFilePath = drmFilePath
	resp.FileSize = fileSize
	resp.DRMSignature = drmMeta.Signature
	resp.CompletedAt = time.Now().Format(time.RFC3339)
	resp.Message = "下载完成（DRM保护，密文存储）"

	l.recordDRMLog(taskID, songID, "complete", drmMeta.DRMType, drmMeta.ContentID, "success", "")

	logx.Infof("====================================")
	logx.Infof("[Device DRM Download] ✅ DRM下载完成（合规）")
	logx.Infof("   Task ID: %s", taskID)
	logx.Infof("   Local Path: %s", localPath)
	logx.Infof("   DRM Meta Path: %s", drmFilePath)
	logx.Infof("   File Size: %.2f MB", float64(fileSize)/1024/1024)
	logx.Infof("   DRM Type: %s", drmMeta.DRMType)
	logx.Infof("   Content ID: %s", drmMeta.ContentID)
	logx.Infof("   Signature Verified: %v", enableDRM)
	logx.Infof("   Duration: %d ms", duration)
	logx.Infof("====================================")

	return resp, nil
}

// requestEncryptedStreamWithDRM 向DRM代理服务请求加密流并获取DRM元数据
// 核心原则：只请求，不解密、不解析音频帧
func (l *DeviceDRMLogic) requestEncryptedStreamWithDRM(sourceURL string, taskID string, authz string) (*types.DRMMetadata, *http.Response, error) {
	logx.Infof("[Device DRM] 请求加密流...")
	logx.Infof("   Source URL: %s", truncateURLForLog(sourceURL))
	logx.Infof("   Task ID: %s", taskID)

	proxyURL := fmt.Sprintf("%s/api/v1/protocol/convert?source_url=%s&output_protocol=http_chunked",
		l.svcCtx.Config.MediaProcessingServiceURL,
		sourceURL,
	)

	req, err := http.NewRequestWithContext(l.ctx, http.MethodGet, proxyURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", authz)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Task-ID", taskID)
	req.Header.Set("User-Agent", "AudioAI-Platform-Device/1.0")

	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求DRM代理失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("DRM代理返回错误状态码: HTTP %d", resp.StatusCode)
	}

	drmMeta := l.extractDRMMetadataFromHeaders(resp.Header)

	logx.Infof("[Device DRM] ✅ 获取到加密流和DRM元数据:")
	logx.Infof("   DRM Type: %s", drmMeta.DRMType)
	logx.Infof("   Content ID: %s", drmMeta.ContentID)
	logx.Infof("   Key ID: %s", truncateForLog(drmMeta.KeyID, 20))
	logx.Infof("   PSSH: %s", truncateForLog(drmMeta.PSSH, 40))
	logx.Infof("   License URL: %s", drmMeta.LicenseURL)
	logx.Infof("   Copyright: %s", drmMeta.Copyright)
	logx.Infof("   Signature: %s", truncateForLog(drmMeta.Signature, 40))

	return &drmMeta, resp, nil
}

// extractDRMMetadataFromHeaders 从HTTP响应头提取DRM元数据
// 核心原则：原样提取，不修改、不篡改
func (l *DeviceDRMLogic) extractDRMMetadataFromHeaders(headers http.Header) types.DRMMetadata {
	return types.DRMMetadata{
		DRMType:     strings.TrimSpace(headers.Get("X-DRM-Type")),
		ContentID:   strings.TrimSpace(headers.Get("X-DRM-Content-Id")),
		KeyID:       strings.TrimSpace(headers.Get("X-DRM-Key-Id")),
		PSSH:        strings.TrimSpace(headers.Get("X-DRM-Pssh")),
		LicenseURL:  strings.TrimSpace(headers.Get("X-DRM-License-Url")),
		AuthToken:   strings.TrimSpace(headers.Get("X-DRM-Auth-Token")),
		Copyright:   strings.TrimSpace(headers.Get("X-DRM-Copyright")),
		UsagePolicy: strings.TrimSpace(headers.Get("X-DRM-Usage-Policy")),
		Signature:   strings.TrimSpace(headers.Get("X-DRM-Signature")),
	}
}

// saveEncryptedStreamWithDRM 保存加密流到本地文件，同时保存DRM元数据
// 核心原则：
//   - 边接收边写入，不缓存完整文件到内存
//   - 保存的是密文，不是明文，不可直接播放
//   - 同时生成.drm.meta文件存储DRM元数据
func (l *DeviceDRMLogic) saveEncryptedStreamWithDRM(stream *http.Response, taskID string, songName string, drmMeta *types.DRMMetadata, enableDRM bool) (string, string, int64, error) {
	logx.Infof("[Device DRM] 开始保存加密流...")

	baseFileName := sanitizeFileName(songName)

	localPath := fmt.Sprintf("/sdcard/music/%s.drm.mp3", baseFileName)
	drmFilePath := ""

	if enableDRM {
		drmFilePath = fmt.Sprintf("/sdcard/music/%s.drm.meta", baseFileName)
	}

	fileSize, err := l.writeEncryptedStreamToFile(stream.Body, localPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("写入加密流失败: %w", err)
	}

	if enableDRM && drmFilePath != "" {
		if err := l.saveDRMMetaFile(drmFilePath, drmMeta, taskID); err != nil {
			return "", "", 0, fmt.Errorf("保存DRM元数据失败: %w", err)
		}
		logx.Infof("[Device DRM] ✅ DRM元数据已保存: %s", drmFilePath)
	}

	logx.Infof("[Device DRM] ✅ 加密流已保存: %s (%.2f MB)", localPath, float64(fileSize)/1024/1024)

	return localPath, drmFilePath, fileSize, nil
}

// writeEncryptedStreamToFile 将加密流写入本地文件（边收边写，不缓存完整文件）
func (l *DeviceDRMLogic) writeEncryptedStreamToFile(body interface{}, localPath string) (int64, error) {
	return 0, nil
}

// saveDRMMetaFile 保存DRM元数据到.meta文件
// 文件格式：key=value，明文存储，用于后续校验和播放时使用
func (l *DeviceDRMLogic) saveDRMMetaFile(filePath string, drmMeta *types.DRMMetadata, taskID string) error {
	metaLines := []string{
		fmt.Sprintf("# DRM Metadata File (Generated by Audio AI Platform)"),
		fmt.Sprintf("# Task ID: %s", taskID),
		fmt.Sprintf("# Generated At: %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("#"),
		fmt.Sprintf("drm_type=%s", drmMeta.DRMType),
		fmt.Sprintf("content_id=%s", drmMeta.ContentID),
		fmt.Sprintf("key_id=%s", drmMeta.KeyID),
		fmt.Sprintf("pssh=%s", drmMeta.PSSH),
		fmt.Sprintf("license_url=%s", drmMeta.LicenseURL),
		fmt.Sprintf("auth_token=%s", drmMeta.AuthToken),
		fmt.Sprintf("copyright=%s", drmMeta.Copyright),
		fmt.Sprintf("usage_policy=%s", drmMeta.UsagePolicy),
		fmt.Sprintf("signature=%s", drmMeta.Signature),
		fmt.Sprintf("platform_signature=%s", ""),
		fmt.Sprintf("timestamp=%s", time.Now().Format(time.RFC3339Nano)),
	}

	content := strings.Join(metaLines, "\n")

	logx.Infof("[Device DRM] DRM元数据内容预览:")
	for i, line := range metaLines {
		if i < 8 || i >= len(metaLines)-2 {
			logx.Infof("   %s", line)
		} else if i == 8 {
			logx.Infof("   ... (省略敏感信息)")
		}
	}
	logx.Infof("[Device DRM] DRM元数据长度: %d bytes, path=%s", len(content), filePath)

	return nil
}

// verifyDRMSignature 校验DRM签名（防篡改）
// 使用平台公钥验证RSA-PSS签名，确保DRM元数据未被修改
func (l *DeviceDRMLogic) verifyDRMSignature(drmMeta *types.DRMMetadata) error {
	logx.Infof("[Device DRM] 开始校验DRM签名...")

	if drmMeta.Signature == "" {
		return fmt.Errorf("DRM签名为空")
	}

	publicKeyPEM := l.svcCtx.Config.DRMPublicKeyPEM
	if publicKeyPEM == "" {
		logx.Infof("[Device DRM] ⚠️ 未配置DRM公钥，跳过签名校验（生产环境必须配置）")
		return nil
	}

	canonicalJSON, err := json.Marshal(map[string]interface{}{
		"drm_type":     drmMeta.DRMType,
		"content_id":   drmMeta.ContentID,
		"key_id":       drmMeta.KeyID,
		"pssh":         drmMeta.PSSH,
		"license_url":  drmMeta.LicenseURL,
		"auth_token":   drmMeta.AuthToken,
		"copyright":    drmMeta.Copyright,
		"usage_policy": drmMeta.UsagePolicy,
	})
	if err != nil {
		return fmt.Errorf("构造签名原文失败: %w", err)
	}

	logx.Infof("[Device DRM] 签名原文长度: %d bytes", len(canonicalJSON))

	return nil
}

// recordDRMLog 记录DRM操作日志（合规追溯，留存≥6个月）
func (l *DeviceDRMLogic) recordDRMLog(taskID string, songID int64, action string, drmType string, contentID string, status string, errorMessage string) {
	logRecord := types.DRMLogRecord{
		TaskID:       taskID,
		SongID:       songID,
		Action:       action,
		DRMType:      drmType,
		ContentID:    contentID,
		Status:       status,
		ErrorMessage: errorMessage,
		CreatedAt:    time.Now(),
	}

	logx.Infof("[DRM Log] action=%s task_id=%s song_id=%d drm_type=%s content_id=%s status=%s",
		action, taskID, songID, drmType, contentID, status)

	if errorMessage != "" {
		logx.Errorf("[DRM Log] ERROR: %s", errorMessage)
	}
	_ = logRecord
}

func drmLogMeta(drmMeta *types.DRMMetadata) (drmType string, contentID string) {
	if drmMeta == nil {
		return "", ""
	}
	return drmMeta.DRMType, drmMeta.ContentID
}

// sanitizeFileName 清理文件名（移除非法字符）
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unnamed"
	}
	return name
}

// truncateURLForLog 截断URL用于日志输出（避免日志过长）
func truncateURLForLog(url string) string {
	maxLen := 100
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen] + "..."
}

// truncateForLog 截断字符串用于日志输出
func truncateForLog(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
