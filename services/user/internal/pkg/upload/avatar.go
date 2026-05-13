package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

var (
	defaultAllowedExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	defaultMaxFileSize       = int64(10 * 1024 * 1024) // 10MB
)

// Config 文件上传配置。
type Config struct {
	MaxFileSize       int64
	SavePath          string
	AllowedExtensions []string
}

// UploadResult 上传结果。
type UploadResult struct {
	Filename string // 存储的文件名（不含路径）
	URL      string // 可访问的URL或相对路径
	Path     string // 完整的本地存储路径
	Size     int64  // 文件大小（字节）
}

// SaveAvatar 保存头像文件并返回结果。
func SaveAvatar(file *multipart.FileHeader, cfg *Config) (*UploadResult, error) {
	if file == nil {
		return nil, nil // 没有上传文件，返回nil但不报错
	}

	if cfg == nil {
		cfg = &Config{
			MaxFileSize:       defaultMaxFileSize,
			AllowedExtensions: defaultAllowedExtensions,
			SavePath:          "./uploads",
		}
	}
	if len(cfg.AllowedExtensions) == 0 {
		cfg.AllowedExtensions = defaultAllowedExtensions
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = defaultMaxFileSize
	}
	if cfg.SavePath == "" {
		cfg.SavePath = "./uploads"
	}

	if err := validateFile(file, cfg); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		logx.Errorf("open uploaded file: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "无法读取上传的文件")
	}
	defer src.Close()

	filename := generateFilename(file.Filename)
	subDir := "avatars"
	saveDir := filepath.Join(cfg.SavePath, subDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		logx.Errorf("create upload directory: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "无法创建存储目录")
	}

	savePath := filepath.Join(saveDir, filename)
	dst, err := os.Create(savePath)
	if err != nil {
		logx.Errorf("create file: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "无法保存文件")
	}
	defer dst.Close()

	size, err := io.Copy(dst, src)
	if err != nil {
		logx.Errorf("copy file content: %v", err)
		os.Remove(savePath)
		return nil, errorx.NewCodeError(errorx.CodeSystemError, "文件写入失败")
	}

	urlPath := fmt.Sprintf("/uploads/%s/%s", subDir, filename)

	return &UploadResult{
		Filename: filename,
		URL:      urlPath,
		Path:     savePath,
		Size:     size,
	}, nil
}

func validateFile(file *multipart.FileHeader, cfg *Config) error {
	if file.Size > cfg.MaxFileSize {
		return errorx.NewCodeError(errorx.CodeInvalidParam,
			fmt.Sprintf("文件大小超过限制，最大允许 %dMB", cfg.MaxFileSize/(1024*1024)))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := false
	for _, e := range cfg.AllowedExtensions {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		return errorx.NewCodeError(errorx.CodeInvalidParam,
			fmt.Sprintf("不支持的文件格式，允许的格式：%s", strings.Join(cfg.AllowedExtensions, ", ")))
	}

	contentType := file.Header.Get("Content-Type")
	if !isImageContentType(contentType) {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "只支持图片文件")
	}

	return nil
}

func generateFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	}
	return hex.EncodeToString(b) + ext
}

func isImageContentType(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}
