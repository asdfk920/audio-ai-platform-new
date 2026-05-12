package logic

import (
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/lib/pq"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

type AudioUploadReq struct {
	Filename string
	FileSize int64
	UserID   int64
}

type AudioUploadResp struct {
	Success  bool   `json:"success"`
	ID       int64  `json:"id"` // 新增：数据库记录ID
	URL      string `json:"url"`
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Duration string `json:"duration,omitempty"`
	Message  string `json:"message"`
}

type AudioUploadLogic struct {
	logx.Logger
	svcCtx *svc.ServiceContext
	db     *sql.DB
}

func NewAudioUploadLogic(svcCtx *svc.ServiceContext) *AudioUploadLogic {
	return &AudioUploadLogic{
		Logger: logx.WithContext(nil),
		svcCtx: svcCtx,
		db:     svcCtx.DB,
	}
}

func (l *AudioUploadLogic) UploadAudio(file io.Reader, filename string, size int64, userID int64) (*AudioUploadResp, error) {
	if file == nil {
		return nil, fmt.Errorf("文件为空")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".mp3"
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}

	objectKey := l.generateObjectKey(userID, filename, ext)

	l.Infof("开始上传音频：user_id=%d, filename=%s, size=%d bytes", userID, filename, size)

	var url string
	var err error

	switch l.svcCtx.Config.Storage.Driver {
	case "s3":
		url, err = l.uploadToS3(file, objectKey)
	case "oss":
		url, err = l.uploadToOSS(file, objectKey)
	default:
		url, err = l.uploadToLocal(file, objectKey)
	}

	if err != nil {
		l.Errorf("上传失败：%v", err)
		return nil, fmt.Errorf("上传失败：%w", err)
	}

	recordID, err := l.saveToDatabase(userID, objectKey, url, filename, ext, size)
	if err != nil {
		l.Errorf("保存到数据库失败：%v", err)
		// 数据库保存失败不影响返回（文件已上传成功）
		recordID = 0
	}

	resp := &AudioUploadResp{
		Success:  true,
		ID:       recordID,
		URL:      url,
		Key:      objectKey,
		Filename: filename,
		Size:     size,
		Message:  "音频上传成功",
	}

	l.Infof("音频上传成功：id=%d, url=%s, key=%s", recordID, url, objectKey)

	return resp, nil
}

func (l *AudioUploadLogic) saveToDatabase(userID int64, fileKey, fileURL, filename, extension string, fileSize int64) (int64, error) {
	if l.db == nil {
		l.Errorf("数据库连接为空，跳过保存")
		return 0, fmt.Errorf("数据库未初始化")
	}

	mimeType := getMimeType(extension)

	query := `
		INSERT INTO audio_uploads (
			user_id, file_key, file_url, original_filename,
			file_extension, file_size, mime_type,
			storage_driver, storage_bucket, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'uploaded')
		RETURNING id
	`

	var recordID int64
	err := l.db.QueryRow(query,
		userID,
		fileKey,
		fileURL,
		filename,
		extension,
		fileSize,
		mimeType,
		l.svcCtx.Config.Storage.Driver,
		l.svcCtx.Config.Storage.Bucket,
	).Scan(&recordID)

	if err != nil {
		return 0, fmt.Errorf("插入音频上传记录失败：%w", err)
	}

	l.Infof("已保存到数据库：id=%d, user_id=%d", recordID, userID)

	return recordID, nil
}

func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		"mp3":  "audio/mpeg",
		"wav":  "audio/wav",
		"flac": "audio/flac",
		"aac":  "audio/aac",
		"ogg":  "audio/ogg",
		"m4a":  "audio/mp4",
		"wma":  "audio/x-ms-wma",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func (l *AudioUploadLogic) generateObjectKey(userID int64, filename string, ext string) string {
	timestamp := time.Now().Format("20060102")
	fileUUID := uuid.New().String()[:8]
	safeName := sanitizeFilename(filename)

	return fmt.Sprintf("audio/%d/%s/%s_%s.%s",
		userID,
		timestamp,
		fileUUID,
		safeName,
		ext,
	)
}

func sanitizeFilename(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "..", "")

	const maxLen = 50
	if len(name) > maxLen {
		name = name[:maxLen]
	}

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		name = strings.ReplaceAll(name, string(r), "_")
	}

	return name
}

func (l *AudioUploadLogic) uploadToS3(file io.Reader, objectKey string) (string, error) {
	cfg := l.svcCtx.Config.Storage

	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return "", fmt.Errorf("S3 配置不完整，请检查 Endpoint、AccessKey、SecretKey、Bucket")
	}

	l.Infof("上传到 S3：bucket=%s, key=%s", cfg.Bucket, objectKey)

	session, err := newAWSSession(cfg.Region, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
	if err != nil {
		return "", fmt.Errorf("创建 S3 会话失败：%w", err)
	}

	uploader := newS3Uploader(session, cfg.Bucket, cfg.Region)

	url, err := uploader.Upload(file, objectKey)
	if err != nil {
		return "", fmt.Errorf("S3 上传失败：%w", err)
	}

	return url, nil
}

func (l *AudioUploadLogic) uploadToOSS(file io.Reader, objectKey string) (string, error) {
	cfg := l.svcCtx.Config.Storage

	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return "", fmt.Errorf("OSS 配置不完整，请检查 Endpoint、AccessKey、SecretKey、Bucket")
	}

	l.Infof("上传到阿里云 OSS：bucket=%s, key=%s", cfg.Bucket, objectKey)

	client, err := newOSSClient(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
	if err != nil {
		return "", fmt.Errorf("创建 OSS 客户端失败：%w", err)
	}

	bucket := client.Bucket(cfg.Bucket)

	err = bucket.PutObject(objectKey, file)
	if err != nil {
		return "", fmt.Errorf("OSS 上传失败：%w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", cfg.Bucket, cfg.Endpoint, objectKey)

	return url, nil
}

func (l *AudioUploadLogic) uploadToLocal(file io.Reader, objectKey string) (string, error) {
	rootDir := l.svcCtx.Config.Storage.Local.Root
	if rootDir == "" {
		rootDir = "./data/ai-worker-objects"
	}

	fullPath := filepath.Join(rootDir, objectKey)

	dir := filepath.Dir(fullPath)
	if err := osMkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败：%w", err)
	}

	dstFile, err := osCreate(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败：%w", err)
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, file)
	if err != nil {
		return "", fmt.Errorf("写入文件失败：%w", err)
	}

	l.Infof("本地存储成功：path=%s, size=%d bytes", fullPath, written)

	url := "/api/v1/storage/" + objectKey
	return url, nil
}
