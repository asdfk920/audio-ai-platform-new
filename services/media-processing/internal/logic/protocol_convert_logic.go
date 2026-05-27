package logic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/drmpass"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

const (
	DefaultConnectTimeout = 10 * time.Second
	DefaultReadTimeout    = 30 * time.Minute
	DefaultWriteTimeout   = 10 * time.Second
	HLSTimeout            = 10 * time.Second
	maxM3U8Depth          = 5
)

// readCloserPair 包装已预读前缀的 Reader 与底层 closer（如 WAV 去头后的 bufio.Reader）
type readCloserPair struct {
	io.Reader
	io.Closer
}

// ProtocolConvertLogic 协议转换业务逻辑
type ProtocolConvertLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProtocolConvertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProtocolConvertLogic {
	return &ProtocolConvertLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Convert 协议转换入口方法
// 根据请求参数选择合适的转换策略并执行
func (l *ProtocolConvertLogic) Convert(req *types.ProtocolConvertReq) (*types.ProtocolConvertResp, error) {
	sessionID := uuid.New().String()[:8]

	logx.Infof("====================================")
	logx.Infof("[Protocol Convert] 开始协议转换...")
	logx.Infof("   Session ID: %s", sessionID)
	logx.Infof("   Source URL: %s", req.SourceURL)
	logx.Infof("   Source Format: %s", req.SourceFormat)
	logx.Infof("   Output Protocol: %s", req.OutputProtocol)
	logx.Infof("   Chunk Size: %d bytes", req.ChunkSize)
	logx.Infof("====================================")

	if req.SourceURL == "" {
		return nil, fmt.Errorf("源地址不能为空")
	}

	outputProtocol := strings.ToLower(strings.TrimSpace(req.OutputProtocol))
	if outputProtocol == "" {
		outputProtocol = types.OutputProtocolHTTPChunked
	}
	if outputProtocol == "websocket" {
		return nil, fmt.Errorf("已不再支持 websocket 输出，请使用 HTTP 分块（output_protocol=http_chunked 或可省略默认）")
	}
	if outputProtocol != types.OutputProtocolHTTPChunked {
		return nil, fmt.Errorf("仅支持 HTTP 分块输出 (http_chunked)，简写 chunked / http")
	}

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		if req.Embedded {
			chunkSize = types.EmbeddedDefaultChunkSize
		} else {
			chunkSize = types.DefaultChunkSize
		}
	}

	sourceFormat := l.detectSourceFormat(req.SourceURL, req.SourceFormat)
	if req.PCMRawStrip && sourceFormat != types.SourceFormatWAV {
		return nil, fmt.Errorf("pcm_raw=1 仅适用于 wav 源（当前检测为 %s）", sourceFormat)
	}
	logx.Infof("[Protocol Convert] 检测到源格式: %s", sourceFormat)

	resp := &types.ProtocolConvertResp{
		SessionID: sessionID,
		SourceURL: req.SourceURL,
		Protocol:  outputProtocol,
		Status:    "streaming",
	}

	return resp, nil
}

func (l *ProtocolConvertLogic) passthroughDRM(w http.ResponseWriter, hdr http.Header) {
	if hdr == nil || l.svcCtx == nil || l.svcCtx.DRMPass == nil {
		return
	}
	rt := l.svcCtx.DRMPass
	drmpass.ApplyPassthrough(hdr, w, rt.Signer, rt.Issuer, rt.AttestEnabled)
}

// StreamViaHTTPChunked 通过 HTTP 分块传输推送音频流（MP3 / HLS→TS / WAV 或 WAV 裸 PCM）
func (l *ProtocolConvertLogic) StreamViaHTTPChunked(sourceURL string, sourceFormat string, chunkSize int, pcmRaw bool, w http.ResponseWriter, r *http.Request) error {
	sessionID := uuid.New().String()[:8]
	startTime := time.Now()
	totalBytes := int64(0)

	logx.Infof("====================================")
	logx.Infof("[HTTP Chunked] 开始HTTP分块传输...")
	logx.Infof("   Session ID: %s", sessionID)
	logx.Infof("   Source URL: %s", truncateURL(sourceURL))
	logx.Infof("   Source Format: %s", sourceFormat)
	logx.Infof("   PCM Raw: %v", pcmRaw)
	logx.Infof("   Chunk Size: %d bytes", chunkSize)
	logx.Infof("====================================")

	defer func() {
		duration := time.Since(startTime).Milliseconds()
		logx.Infof("[HTTP Chunked] ✅ 传输完成:")
		logx.Infof("   Session ID: %s", sessionID)
		logx.Infof("   Total Bytes: %.2f MB", float64(totalBytes)/1024/1024)
		logx.Infof("   Duration: %d ms", duration)
	}()

	var streamErr error
	switch sourceFormat {
	case types.SourceFormatMP3:
		resp, err := l.sourceHTTPGet(sourceURL)
		if err != nil {
			logx.Errorf("[HTTP Chunked] 请求源失败: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("源返回错误状态码: HTTP %d", resp.StatusCode)
			logx.Errorf("[HTTP Chunked] %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return err
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		l.passthroughDRM(w, resp.Header)
		l.setHTTPStreamCommonHeaders(w, sessionID)
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		streamErr = l.streamReaderToHTTPChunked(resp.Body, chunkSize, w, &totalBytes, sessionID)

	case types.SourceFormatHLS:
		w.Header().Set("Content-Type", "video/MP2T")
		l.setHTTPStreamCommonHeaders(w, sessionID)
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		streamErr = l.streamHLSToHTTPChunked(sourceURL, sessionID, chunkSize, w, &totalBytes)

	case types.SourceFormatWAV:
		if pcmRaw {
			rc, wavFmt, err := l.openWAVPCMReader(sourceURL)
			if err != nil {
				logx.Errorf("[HTTP Chunked] WAV 裸流准备失败: %v", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return err
			}
			defer rc.Close()
			w.Header().Set("Content-Type", "application/octet-stream")
			if wavFmt != nil {
				w.Header().Set("X-Embedded-Audio-Fmt", embeddedAudioFmtHint(wavFmt))
			}
			l.setHTTPStreamCommonHeaders(w, sessionID)
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			streamErr = l.streamReaderToHTTPChunked(rc, chunkSize, w, &totalBytes, sessionID)
		} else {
			resp, err := l.sourceHTTPGet(sourceURL)
			if err != nil {
				logx.Errorf("[HTTP Chunked] 请求 WAV 源失败: %v", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				err := fmt.Errorf("源返回错误状态码: HTTP %d", resp.StatusCode)
				logx.Errorf("[HTTP Chunked] %v", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return err
			}
			w.Header().Set("Content-Type", "audio/wav")
			l.passthroughDRM(w, resp.Header)
			l.setHTTPStreamCommonHeaders(w, sessionID)
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			streamErr = l.streamReaderToHTTPChunked(resp.Body, chunkSize, w, &totalBytes, sessionID)
		}

	default:
		http.Error(w, fmt.Sprintf("不支持的源格式: %s", sourceFormat), http.StatusBadRequest)
		return fmt.Errorf("不支持的源格式: %s", sourceFormat)
	}

	return streamErr
}

func (l *ProtocolConvertLogic) setHTTPStreamCommonHeaders(w http.ResponseWriter, sessionID string) {
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Session-ID", sessionID)
	w.Header().Set("Connection", "keep-alive")
}

func (l *ProtocolConvertLogic) sourceHTTPGet(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: DefaultReadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(l.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (audio-ai-platform media-processing)")
	return client.Do(req)
}

func (l *ProtocolConvertLogic) openWAVPCMReader(url string) (io.ReadCloser, *WAVFmt, error) {
	resp, err := l.sourceHTTPGet(url)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("源返回错误状态码: HTTP %d", resp.StatusCode)
	}
	br := bufio.NewReader(resp.Body)
	wavFmt, err := ConsumeWAVHeaderToData(br)
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}
	return &readCloserPair{Reader: br, Closer: resp.Body}, wavFmt, nil
}

func (l *ProtocolConvertLogic) streamReaderToHTTPChunked(r io.Reader, chunkSize int, w http.ResponseWriter, totalBytes *int64, sessionID string) error {
	buffer := make([]byte, chunkSize)
	flusher, _ := w.(http.Flusher)
	for {
		select {
		case <-l.ctx.Done():
			logx.Infof("[HTTP Chunked] 上下文取消，停止传输 session=%s", sessionID)
			return l.ctx.Err()
		default:
		}

		n, readErr := r.Read(buffer)
		if n > 0 {
			_, writeErr := w.Write(buffer[:n])
			if writeErr != nil {
				logx.Errorf("[HTTP Chunked] 写入失败: %v", writeErr)
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
			*totalBytes += int64(n)
		}

		if readErr == io.EOF {
			logx.Infof("[HTTP Chunked] 流读取完毕，总字节数: %d", *totalBytes)
			break
		}
		if readErr != nil {
			logx.Errorf("[HTTP Chunked] 读取异常: %v", readErr)
			return readErr
		}
	}
	return nil
}

// streamHLSToHTTPChunked HLS流转换为HTTP分块传输
func (l *ProtocolConvertLogic) streamHLSToHTTPChunked(m3u8URL string, sessionID string, chunkSize int, w http.ResponseWriter, totalBytes *int64) error {
	logx.Infof("[HTTP Chunked] 开始解析HLS流: %s", truncateURL(m3u8URL))

	tsURLs, err := l.parseM3U8(m3u8URL)
	if err != nil {
		logx.Errorf("[HTTP Chunked] 解析m3u8失败: %v", err)
		return err
	}

	logx.Infof("[HTTP Chunked] 解析完成，共 %d 个TS分片", len(tsURLs))

	client := &http.Client{
		Timeout: HLSTimeout,
	}

	flusher, _ := w.(http.Flusher)

	for i, tsURL := range tsURLs {
		select {
		case <-l.ctx.Done():
			logx.Infof("[HTTP Chunked] 客户端断开连接，停止传输")
			return l.ctx.Err()
		default:
		}

		logx.Debugf("[HTTP Chunked] 正在下载第 %d/%d 个TS分片: %s", i+1, len(tsURLs), tsURL)

		tsReq, err := http.NewRequestWithContext(l.ctx, http.MethodGet, tsURL, nil)
		if err != nil {
			logx.Errorf("[HTTP Chunked] 创建TS请求失败: %v", err)
			continue
		}

		tsResp, err := client.Do(tsReq)
		if err != nil {
			logx.Errorf("[HTTP Chunked] 下载TS失败: %v", err)
			continue
		}

		buffer := make([]byte, chunkSize)
		for {
			select {
			case <-l.ctx.Done():
				tsResp.Body.Close()
				logx.Infof("[HTTP Chunked] 上下文取消，停止传输 TS")
				return l.ctx.Err()
			default:
			}

			n, readErr := tsResp.Body.Read(buffer)
			if n > 0 {
				_, writeErr := w.Write(buffer[:n])
				if writeErr != nil {
					tsResp.Body.Close()
					logx.Errorf("[HTTP Chunked] 写入TS数据失败: %v", writeErr)
					return writeErr
				}

				if flusher != nil {
					flusher.Flush()
				}

				*totalBytes += int64(n)
			}

			if readErr == io.EOF {
				break
			}

			if readErr != nil {
				logx.Errorf("[HTTP Chunked] 读取TS数据异常: %v", readErr)
				break
			}
		}

		tsResp.Body.Close()
		logx.Debugf("[HTTP Chunked] 第 %d/%d 个TS分片传输完成", i+1, len(tsURLs))
	}

	logx.Infof("[HTTP Chunked] ✅ 所有TS分片传输完成，总字节数: %d", *totalBytes)
	return nil
}

// parseM3U8 解析 m3u8：支持 MASTER（#EXT-X-STREAM-INF 后接子播放列表）与媒体列表中的 TS（或分段 URI）
func (l *ProtocolConvertLogic) parseM3U8(m3u8URL string) ([]string, error) {
	return l.parseM3U8Depth(m3u8URL, 0)
}

func resolveM3U8Ref(baseURL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	idx := strings.LastIndex(baseURL, "/")
	if idx < 0 {
		return ref
	}
	return baseURL[:idx+1] + ref
}

func (l *ProtocolConvertLogic) parseM3U8Depth(m3u8URL string, depth int) ([]string, error) {
	if depth > maxM3U8Depth {
		return nil, fmt.Errorf("HLS m3u8 嵌套解析超过上限")
	}

	client := &http.Client{
		Timeout: HLSTimeout,
	}

	req, err := http.NewRequestWithContext(l.ctx, http.MethodGet, m3u8URL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建m3u8请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求m3u8文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取m3u8文件失败: HTTP %d", resp.StatusCode)
	}

	bodyReader := bufio.NewReader(resp.Body)
	var segments []string
	var masterVariants []string
	expectingMasterURI := false

	for {
		line, err := bodyReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("读取m3u8内容失败: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			expectingMasterURI = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			expectingMasterURI = false
			continue
		}

		u := resolveM3U8Ref(m3u8URL, line)
		if u == "" {
			continue
		}

		if expectingMasterURI {
			masterVariants = append(masterVariants, u)
			expectingMasterURI = false
			continue
		}

		segments = append(segments, u)
	}

	if len(masterVariants) > 0 {
		return l.parseM3U8Depth(masterVariants[0], depth+1)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("未找到任何分段地址（MASTER 变体或 TS）")
	}

	return segments, nil
}

// detectSourceFormat 自动检测或使用指定的源格式
func (l *ProtocolConvertLogic) detectSourceFormat(url string, specifiedFormat string) string {
	if specifiedFormat != "" {
		format := strings.ToLower(specifiedFormat)
		if format == types.SourceFormatMP3 || format == types.SourceFormatHLS || format == types.SourceFormatWAV {
			return format
		}
	}

	lowerURL := strings.ToLower(url)
	if strings.HasSuffix(lowerURL, ".wav") || strings.Contains(lowerURL, ".wav?") {
		return types.SourceFormatWAV
	}

	if strings.HasSuffix(lowerURL, ".mp3") || strings.Contains(lowerURL, ".mp3?") {
		return types.SourceFormatMP3
	}

	if strings.HasSuffix(lowerURL, ".m3u8") || strings.Contains(lowerURL, ".m3u8?") {
		return types.SourceFormatHLS
	}

	return types.SourceFormatMP3
}
