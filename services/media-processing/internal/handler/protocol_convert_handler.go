package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

// ProtocolConvertHandler 协议转换接口（仅 HTTP chunked 流传输）
// GET /api/v1/protocol/convert?source_url=...&output_protocol=http_chunked|chunked|http（可选，默认同上）
// 嵌入式：embedded=1 仅影响默认 chunk_size=4KB
// WAV 裸 PCM：pcm_raw=1（需 source_format=wav 或 URL 指向 .wav）
func ProtocolConvertHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logx.Infof("====================================")
		logx.Infof("[Protocol Convert Handler] 收到协议转换请求")
		logx.Infof("   Method: %s", r.Method)
		logx.Infof("   URL: %s", r.URL.String())
		logx.Infof("====================================")

		var req types.ProtocolConvertReq

		req.SourceURL = r.URL.Query().Get("source_url")
		req.SourceFormat = r.URL.Query().Get("source_format")
		req.OutputProtocol = logic.NormalizeOutputProtocol(r.URL.Query().Get("output_protocol"))
		req.Embedded = logic.ParseTruthyQuery(r.URL.Query().Get("embedded")) || logic.ParseTruthyQuery(r.URL.Query().Get("embed"))
		req.PCMRawStrip = logic.ParseTruthyQuery(r.URL.Query().Get("pcm_raw"))

		if req.Embedded && req.ChunkSize <= 0 {
			req.ChunkSize = types.EmbeddedDefaultChunkSize
		}

		if req.OutputProtocol == "" {
			req.OutputProtocol = types.OutputProtocolHTTPChunked
		}

		chunkSizeStr := r.URL.Query().Get("chunk_size")
		if chunkSizeStr != "" {
			if size, err := strconv.Atoi(chunkSizeStr); err == nil && size > 0 {
				req.ChunkSize = size
			}
		}

		if req.SourceURL == "" {
			http.Error(w, `{"success":false,"message":"源地址不能为空"}`, http.StatusBadRequest)
			return
		}

		l := logic.NewProtocolConvertLogic(r.Context(), svcCtx)

		_, err := l.Convert(&req)
		if err != nil {
			logx.Errorf("[Protocol Convert Handler] 参数校验失败: error=%v", err)
			http.Error(w, `{"success":false,"message":"`+escapeJSON(err.Error())+`"}`, http.StatusBadRequest)
			return
		}

		sourceFormat := detectSourceFormat(req.SourceURL, req.SourceFormat)
		chunkSz := req.ChunkSize
		if chunkSz <= 0 {
			chunkSz = types.DefaultChunkSize
		}

		logx.Infof("[Protocol Convert Handler] HTTP 分块传输")
		err = l.StreamViaHTTPChunked(req.SourceURL, sourceFormat, chunkSz, req.PCMRawStrip, w, r)
		if err != nil {
			logx.Errorf("[Protocol Convert Handler] HTTP分块传输失败: error=%v", err)
		} else {
			logx.Infof("[Protocol Convert Handler] ✅ HTTP分块传输完成")
		}
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `'`)
	return s
}

// detectSourceFormat 辅助函数：检测源格式
func detectSourceFormat(url string, specifiedFormat string) string {
	if specifiedFormat != "" {
		format := strings.ToLower(specifiedFormat)
		if format == types.SourceFormatMP3 || format == types.SourceFormatHLS || format == types.SourceFormatWAV {
			return format
		}
	}

	lowerURL := strings.ToLower(url)
	if strings.HasSuffix(lowerURL, ".mp3") || strings.Contains(lowerURL, ".mp3?") {
		return types.SourceFormatMP3
	}

	if strings.HasSuffix(lowerURL, ".m3u8") || strings.Contains(lowerURL, ".m3u8?") {
		return types.SourceFormatHLS
	}

	if strings.HasSuffix(lowerURL, ".wav") || strings.Contains(lowerURL, ".wav?") {
		return types.SourceFormatWAV
	}

	return types.SourceFormatMP3
}
