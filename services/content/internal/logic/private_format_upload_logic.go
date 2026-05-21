package logic

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// PrivateFormatUploadResult 私有格式上传：接口仅返回可访问的文件链接。
type PrivateFormatUploadResult struct {
	URL string `json:"url"`
}

type PrivateFormatUploadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPrivateFormatUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrivateFormatUploadLogic {
	return &PrivateFormatUploadLogic{ctx: ctx, svcCtx: svcCtx}
}

// Upload 先写主存储（OSS/S3/local），在云主存储时再镜像 Local.Root；最后归档 content_files。成功只返回 OSS 公链。
func (l *PrivateFormatUploadLogic) Upload(
	fileName, fileSizeStr, fileMd5, format, durationStr string,
	file io.Reader,
) (*PrivateFormatUploadResult, error) {
	if l.svcCtx.Storage == nil {
		return nil, fmt.Errorf("对象存储未初始化，请检查 Storage 配置")
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("fileName 不能为空")
	}
	fileName = path.Base(fileName)
	if fileName == "." || fileName == "/" {
		return nil, fmt.Errorf("fileName 无效")
	}

	format = sanitizeFormatToken(format)
	if format == "" {
		return nil, fmt.Errorf("format 不能为空")
	}

	expectedSize, err := strconv.ParseInt(strings.TrimSpace(fileSizeStr), 10, 64)
	if err != nil || expectedSize <= 0 {
		return nil, fmt.Errorf("fileSize 须为正整数（字节）")
	}

	maxBytes := l.svcCtx.Config.Upload.PrivateFormatMaxMB * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 128 * 1024 * 1024
	}
	if expectedSize > maxBytes {
		return nil, fmt.Errorf("文件超过配置上限 %d MB", l.svcCtx.Config.Upload.PrivateFormatMaxMB)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("文件超过配置上限 %d MB", l.svcCtx.Config.Upload.PrivateFormatMaxMB)
	}
	if int64(len(data)) != expectedSize {
		return nil, fmt.Errorf("实际大小与 fileSize 不一致")
	}

	sum := md5.Sum(data)
	gotMd5 := hex.EncodeToString(sum[:])
	wantMd5 := strings.ToLower(strings.TrimSpace(fileMd5))
	if wantMd5 == "" {
		return nil, fmt.Errorf("fileMd5 不能为空")
	}
	if gotMd5 != wantMd5 {
		return nil, fmt.Errorf("MD5 校验失败")
	}

	var duration float64
	if s := strings.TrimSpace(durationStr); s != "" {
		duration, err = strconv.ParseFloat(s, 64)
		if err != nil || duration < 0 {
			return nil, fmt.Errorf("duration 须为非负数字（秒）")
		}
	}

	objectKey := fmt.Sprintf("private-format/%s/%s/%s", format, uuid.New().String(), fileName)

	driver := strings.ToLower(strings.TrimSpace(l.svcCtx.Config.Storage.Driver))
	logx.Infof("[private-format-upload] Put primary driver=%s object_key=%s bytes=%d format=%s", driver, objectKey, len(data), format)

	r := bytes.NewReader(data)
	if err := l.svcCtx.Storage.Put(l.ctx, objectKey, r, int64(len(data)), "application/octet-stream"); err != nil {
		logx.Errorf("[private-format-upload] primary Put failed object_key=%s: %v", objectKey, err)
		return nil, fmt.Errorf("上传对象存储失败: %w", err)
	}

	pubURL := l.svcCtx.Storage.PublicURL(objectKey)
	logx.Infof("[private-format-upload] primary ok url=%s md5=%s object_key=%s", pubURL, gotMd5, objectKey)

	if l.svcCtx.Config.Upload.PrivateFormatLocalMirror && primaryUsesRemoteBlob(driver) {
		if err := l.writeLocalMirror(objectKey, data); err != nil {
			logx.Errorf("[private-format-upload] local mirror failed object_key=%s: %v", objectKey, err)
		}
	}

	l.saveContentFileBackup(pubURL, objectKey, fileName, int64(len(data)), gotMd5, format, duration)

	return &PrivateFormatUploadResult{URL: pubURL}, nil
}

func primaryUsesRemoteBlob(driver string) bool {
	switch driver {
	case "oss", "aliyun", "aliyunoss", "s3", "minio":
		return true
	default:
		return false
	}
}

func (l *PrivateFormatUploadLogic) writeLocalMirror(objectKey string, data []byte) error {
	root := strings.TrimSpace(l.svcCtx.Config.Local.Root)
	if root == "" {
		root = "./data/content-objects"
	}
	fullPath := filepath.Join(root, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	logx.Infof("[private-format-upload] local mirror ok path=%s bytes=%d", fullPath, len(data))
	return nil
}

// saveContentFileBackup 归档 content_files；失败只打日志，不影响已成功的主存储上传。
func (l *PrivateFormatUploadLogic) saveContentFileBackup(pubURL, objectKey, originalName string, size int64, md5Hex, format string, duration float64) {
	if l.svcCtx.DB == nil {
		logx.Infof("[private-format-upload] DB 为空，跳过 content_files 归档")
		return
	}
	drv := strings.ToLower(strings.TrimSpace(l.svcCtx.Config.Storage.Driver))
	keyHash := "PRIVATE_FORMAT_UPLOAD_NO_ENCRYPT|" + md5Hex

	var dur sql.NullFloat64
	if duration > 0 {
		dur = sql.NullFloat64{Float64: duration, Valid: true}
	}

	tx := l.svcCtx.DB.WithContext(l.ctx)
	var id int64
	err := tx.Raw(`
INSERT INTO content_files (url, key_hash, file_type, original_name, original_size, storage_driver, object_key, format_tag, duration_seconds, file_md5)
VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?)
RETURNING id`,
		pubURL, keyHash, originalName, size, drv, objectKey, format, dur, md5Hex,
	).Scan(&id).Error
	if err == nil {
		logx.Infof("[private-format-upload] content_files archived id=%d object_key=%s", id, objectKey)
		return
	}

	logx.Infof("[private-format-upload] content_files extended insert failed: %v, retry minimal", err)
	errMin := tx.Raw(`
INSERT INTO content_files (url, key_hash, file_type, original_name, original_size)
VALUES (?, ?, 1, ?, ?)
RETURNING id`,
		pubURL, keyHash, originalName, size,
	).Scan(&id).Error
	if errMin != nil {
		logx.Errorf("[private-format-upload] content_files archive failed object_key=%s: %v", objectKey, errMin)
		return
	}
	logx.Infof("[private-format-upload] content_files minimal archived id=%d (考虑执行 migrations/100)", id)
}

func sanitizeFormatToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
