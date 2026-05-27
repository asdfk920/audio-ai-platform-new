package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

// PlaySongWithDRMLogic 播放/下载歌曲（DRM透传版本）
// 核心原则：不解密、不破解、不转码，只做原样透传
type PlaySongWithDRMLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPlaySongWithDRMLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaySongWithDRMLogic {
	return &PlaySongWithDRMLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PlaySong 获取加密音频流+DRM信息（原样透传）
// 完整流程：参数校验 → 用户鉴权 → 请求第三方API → 原样返回
func (l *PlaySongWithDRMLogic) PlaySong(req *types.PlaySongWithDRMReq) (*types.PlaySongWithDRMResp, error) {
	logx.Infof("====================================")
	logx.Infof("[DRM Play Song] 开始处理播放/下载请求")
	logx.Infof("   Song ID: %d", req.SongID)
	logx.Infof("   Quality: %s", req.Quality)
	logx.Infof("====================================")

	if req.SongID <= 0 {
		return nil, fmt.Errorf("歌曲ID不能为空")
	}

	quality := strings.ToLower(strings.TrimSpace(req.Quality))
	if quality == "" {
		quality = "hq"
	}

	if l.svcCtx.Config.ThirdPartyMusic.EnableMock {
		return l.mockPlaySong(req.SongID, quality, strings.TrimSpace(req.Token)), nil
	}

	thirdPartyResp, err := l.fetchThirdPartySongStream(req.SongID, quality)
	if err != nil {
		logx.Errorf("[DRM Play Song] 请求第三方失败: %v", err)
		return nil, fmt.Errorf("获取歌曲信息失败: %v", err)
	}

	resp := &types.PlaySongWithDRMResp{
		Success:     true,
		Message:     "获取成功（DRM保护，原样透传）",
		SongID:      req.SongID,
		SongName:    thirdPartyResp.SongName,
		StreamURL:   thirdPartyResp.StreamURL,
		DRMInfo:     thirdPartyResp.DRMInfo,
		DRMToken:    thirdPartyResp.DRMToken,
		Copyright:   thirdPartyResp.Copyright,
		Encrypted:   true,
		EncryptType: thirdPartyResp.EncryptType,
		Quality:     quality,
	}

	logx.Infof("====================================")
	logx.Infof("[DRM Play Song] ✅ 原样透传成功")
	logx.Infof("   Song ID: %d", resp.SongID)
	logx.Infof("   Song Name: %s", resp.SongName)
	logx.Infof("   Stream URL: %s", truncateURL(resp.StreamURL))
	logx.Infof("   DRM Type: %s", resp.DRMInfo.DRMType)
	logx.Infof("   Copyright: %s", resp.Copyright)
	logx.Infof("   Encrypted: %v (合规)", resp.Encrypted)
	logx.Infof("====================================")

	return resp, nil
}

func (l *PlaySongWithDRMLogic) mockPlaySong(songID int64, quality string, token string) *types.PlaySongWithDRMResp {
	streamURL := fmt.Sprintf("http://mock-cdn.local/encrypted/%d.m3u8", songID)
	resp := &types.PlaySongWithDRMResp{
		Success:   true,
		Message:   "获取成功（Mock DRM保护，原样透传）",
		SongID:    songID,
		SongName:  fmt.Sprintf("Mock Song %d", songID),
		StreamURL: streamURL,
		DRMInfo: types.DRMMetadata{
			DRMType:     "mock-drm",
			ContentID:   fmt.Sprintf("mock-song-%d", songID),
			KeyID:       fmt.Sprintf("mock-key-%d", songID),
			PSSH:        "mock-pssh",
			LicenseURL:  "http://mock-license.local/widevine",
			AuthToken:   token,
			Copyright:   "Mock-ThirdParty-(c)2026",
			UsagePolicy: "mock-test-only",
			Signature:   "mock-signature",
		},
		DRMToken:    token,
		Copyright:   "Mock-ThirdParty-(c)2026",
		Encrypted:   true,
		EncryptType: "AES-128-DRM",
		Quality:     quality,
	}
	logx.Infof("[DRM Play Song] Mock模式启用，返回模拟加密流: song_id=%d url=%s", songID, truncateURL(streamURL))
	return resp
}

// fetchThirdPartySongStream 请求第三方音乐API，获取加密流+DRM信息
// 核心原则：只获取，不解析、不修改、不存储明文
func (l *PlaySongWithDRMLogic) fetchThirdPartySongStream(songID int64, quality string) (*thirdPartySongResponse, error) {
	logx.Infof("[DRM Play Song] 请求第三方音乐API...")
	logx.Infof("   Song ID: %d", songID)
	logx.Infof("   Quality: %s", quality)

	baseURL := strings.TrimRight(strings.TrimSpace(l.svcCtx.Config.ThirdPartyMusic.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("未配置 ThirdPartyMusic.BaseURL")
	}

	thirdPartyURL := fmt.Sprintf("%s/api/v1/song/%d/stream?quality=%s",
		baseURL,
		songID,
		quality,
	)

	timeout := 10 * time.Second
	if l.svcCtx.Config.ThirdPartyMusic.TimeoutSec > 0 {
		timeout = time.Duration(l.svcCtx.Config.ThirdPartyMusic.TimeoutSec) * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(l.ctx, http.MethodGet, thirdPartyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	apiKey := strings.TrimSpace(l.svcCtx.Config.ThirdPartyMusic.AppSecret)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AudioAI-Platform/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求第三方失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("第三方返回错误 HTTP %d: %s", resp.StatusCode, string(body))
	}

	var thirdResp thirdPartySongResponse
	if err := json.NewDecoder(resp.Body).Decode(&thirdResp); err != nil {
		return nil, fmt.Errorf("解析第三方响应失败: %w", err)
	}

	if thirdResp.StreamURL == "" {
		return nil, fmt.Errorf("第三方未返回音频流地址")
	}

	logx.Infof("[DRM Play Song] ✅ 获取到第三方数据:")
	logx.Infof("   Stream URL: %s", truncateURL(thirdResp.StreamURL))
	logx.Infof("   DRM Type: %s", thirdResp.DRMInfo.DRMType)
	logx.Infof("   Copyright: %s", thirdResp.Copyright)
	logx.Infof("   Song Name: %s", thirdResp.SongName)

	return &thirdResp, nil
}

// thirdPartySongResponse 第三方音乐API响应结构（实际使用时替换为真实API）。
type thirdPartySongResponse struct {
	StreamURL   string            `json:"stream_url"`   // 加密音频流地址
	DRMInfo     types.DRMMetadata `json:"drm_info"`     // DRM信息
	DRMToken    string            `json:"drm_token"`    // DRM/版权令牌
	Copyright   string            `json:"copyright"`    // 版权声明
	EncryptType string            `json:"encrypt_type"` // 加密类型
	SongName    string            `json:"song_name"`    // 歌曲名
}
