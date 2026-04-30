package logic

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

type ContentDownloadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentDownloadLogic {
	return &ContentDownloadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContentDownloadLogic) GetDownloadURL(userID int64, req *types.GetDownloadURLReq) (*types.GetDownloadURLResp, error) {
	if req == nil || req.ContentID <= 0 {
		return nil, fmt.Errorf("内容ID无效")
	}

	var audioURL, coverURL sql.NullString
	var audioID, coverID sql.NullInt64
	query := `SELECT audio_url, cover_url, audio_id, cover_id FROM content WHERE id = $1 AND status = 1`
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, req.ContentID).Scan(&audioURL, &coverURL, &audioID, &coverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("内容不存在或未上架")
		}
		return nil, fmt.Errorf("查询内容失败: %v", err)
	}

	if !audioURL.Valid || audioURL.String == "" {
		return nil, fmt.Errorf("音频文件不存在")
	}

	if !audioID.Valid || audioID.Int64 == 0 {
		return nil, fmt.Errorf("音频文件ID不存在")
	}

	fileName := fmt.Sprintf("content_%d_audio.aasp", req.ContentID)
	if coverID.Valid && coverID.Int64 > 0 {
		fileName = fmt.Sprintf("content_%d_package.aasp", req.ContentID)
	}

	downloadURL := fmt.Sprintf("/api/v1/user/content/download/stream?content_id=%d", req.ContentID)

	return &types.GetDownloadURLResp{
		DownloadURL: downloadURL,
		FileName:    fileName,
		ContentType: "application/octet-stream",
	}, nil
}

func (l *ContentDownloadLogic) DownloadContent(userID int64, contentID int64, w http.ResponseWriter, r *http.Request) error {
	var audioURL, coverURL sql.NullString
	var audioID, coverID sql.NullInt64
	query := `SELECT audio_url, cover_url, audio_id, cover_id FROM content WHERE id = $1 AND status = 1`
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, contentID).Scan(&audioURL, &coverURL, &audioID, &coverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("内容不存在或未上架")
		}
		return fmt.Errorf("查询内容失败: %v", err)
	}

	if !audioURL.Valid || audioURL.String == "" {
		return fmt.Errorf("音频文件不存在")
	}

	deviceURL := audioURL.String
	if !audioID.Valid || audioID.Int64 == 0 {
		return fmt.Errorf("音频文件ID不存在")
	}

	fileID := audioID.Int64
	fileName := fmt.Sprintf("content_%d_audio.aasp", contentID)
	if coverID.Valid && coverID.Int64 > 0 {
		fileName = fmt.Sprintf("content_%d_package.aasp", contentID)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Id", fmt.Sprintf("%d", contentID))
	w.Header().Set("X-Audio-File-Id", fmt.Sprintf("%d", fileID))

	localDir := filepath.Join("./downloads/user", fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %v", err)
	}

	localFilePath := filepath.Join(localDir, fileName)

	resp, err := http.Get(deviceURL)
	if err != nil {
		return fmt.Errorf("请求设备服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("设备服务返回错误: %d", resp.StatusCode)
	}

	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer localFile.Close()

	multiWriter := io.MultiWriter(w, localFile)

	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("传输数据失败: %v", err)
	}

	return nil
}
