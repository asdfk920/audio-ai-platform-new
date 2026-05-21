package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// importPlaylistHandler 导入歌单处理器
// @Summary      导入第三方歌单
// @Description  将Spotify或QQ音乐的歌单导入到本地，需要登录
// @Tags         歌单管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ImportPlaylistReq  true  "导入歌单请求"
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      400  {object}  httpresp.Standard  "参数错误"
// @Failure      403  {object}  httpresp.Standard  "歌单数量限制"
// @Failure      404  {object}  httpresp.Standard  "第三方歌单不存在"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /import/playlist [post]
// @Security     BearerAuth
func importPlaylistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.ImportPlaylistReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `读取请求体失败`), nil)
				return
			}

			if err := json.Unmarshal(body, &req); err != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `JSON 格式错误`), nil)
				return
			}
		} else {
			r.ParseForm()
			req.Platform = r.FormValue("platform")
			req.PlaylistID = r.FormValue("playlist_id")
			req.PlaylistName = r.FormValue("playlist_name")
		}

		if req.Platform == "" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `平台不能为空`), nil)
			return
		}

		if req.PlaylistID == "" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `歌单 ID 不能为空`), nil)
			return
		}

		if req.Platform != "spotify" && req.Platform != "qq-music" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `不支持的平台，仅支持 spotify 和 qq-music`), nil)
			return
		}

		logicInstance := logic.NewImportPlaylistLogic(r.Context(), svcCtx)
		resp, err := logicInstance.ImportPlaylist(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("导入歌单失败: userID=%d, platform=%s, playlistID=%s, error=%v",
				bearerCtx.UserID, req.Platform, req.PlaylistID, err)

			if strings.Contains(err.Error(), "歌单数量限制") {
				httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.MsgForbidden, err.Error()), nil)
				return
			}
			if strings.Contains(err.Error(), "未找到") || strings.Contains(err.Error(), "不存在") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}
			if strings.Contains(err.Error(), "已导入") {
				httpresp.WriteSuccessMsg(w, err.Error(), resp)
				return
			}

			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		logx.Infof("导入歌单成功: userID=%d, platform=%s, playlistID=%s, localPlaylistID=%d, imported=%d/%d",
			bearerCtx.UserID, req.Platform, req.PlaylistID, resp.PlaylistID, resp.ImportedCount, resp.TotalCount)

		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}
