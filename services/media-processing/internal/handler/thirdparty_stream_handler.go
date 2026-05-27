package handler

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

// ThirdPartyStreamUrlHandler 获取第三方平台音频流地址（网易云音乐公共API）
// GET /api/v1/thirdparty/stream/url?id=1816904098&level=standard
func ThirdPartyStreamUrlHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
			return
		}

		var req types.ThirdPartyStreamUrlReq

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "歌曲ID不能为空", http.StatusBadRequest)
			return
		}

		req.ID, _ = strconv.ParseInt(idStr, 10, 64)
		req.Level = r.URL.Query().Get("level")

		logx.Infof("[ThirdParty Stream URL Handler] 收到请求: id=%d, level=%s",
			req.ID, req.Level)

		l := logic.NewThirdPartyStreamUrlLogic(r.Context(), svcCtx)
		resp, err := l.GetStreamUrl(&req)

		if err != nil {
			logx.Errorf("[ThirdParty Stream URL Handler] 处理失败: error=%v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logx.Infof("[ThirdParty Stream URL Handler] ✅ 处理成功: success=%v, message=%s",
			resp.Success, resp.Message)

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// ThirdPartyStreamProxyHandler 代理转发第三方音频流（支持断点续播）
// GET /api/v1/thirdparty/stream/play?url=https://music.163.com/...
func ThirdPartyStreamProxyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
			return
		}

		audioURL := r.URL.Query().Get("url")
		if audioURL == "" {
			http.Error(w, "音频URL不能为空", http.StatusBadRequest)
			return
		}

		logx.Infof("[ThirdParty Stream Proxy Handler] 收到代理请求: url=%s...",
			truncateURL(audioURL))

		l := logic.NewThirdPartyStreamUrlLogic(r.Context(), svcCtx)
		err := l.ProxyThirdPartyStream(audioURL, w, r)

		if err != nil {
			logx.Errorf("[ThirdParty Stream Proxy Handler] 代理转发失败: error=%v", err)
		} else {
			logx.Infof("[ThirdParty Stream Proxy Handler] ✅ 代理转发完成")
		}
	}
}

func truncateURL(url string, maxLen ...int) string {
	length := 80
	if len(maxLen) > 0 && maxLen[0] > 0 {
		length = maxLen[0]
	}

	if len(url) <= length {
		return url
	}

	return url[:length] + "..."
}
