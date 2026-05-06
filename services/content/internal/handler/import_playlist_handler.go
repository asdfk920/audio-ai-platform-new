package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// importPlaylistHandler 导入歌单处理器
// @Summary      导入第三方歌单
// @Description  将Spotify或QQ音乐的歌单导入到本地，需要登录
// @Tags         歌单管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ImportPlaylistReq  true  "导入歌单请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Failure      403  {object}  map[string]interface{}  "歌单数量限制"
// @Failure      404  {object}  map[string]interface{}  "第三方歌单不存在"
// @Failure      500  {object}  map[string]interface{}  "服务器错误"
// @Router       /import/playlist [post]
// @Security     BearerAuth
func importPlaylistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req types.ImportPlaylistReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}

			if err := json.Unmarshal(body, &req); err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "JSON格式错误",
					"data":    nil,
				})
				return
			}
		} else {
			r.ParseForm()
			req.Platform = r.FormValue("platform")
			req.PlaylistID = r.FormValue("playlist_id")
			req.PlaylistName = r.FormValue("playlist_name")
		}

		// 验证必填参数
		if req.Platform == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "平台不能为空",
				"data":    nil,
			})
			return
		}

		if req.PlaylistID == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单ID不能为空",
				"data":    nil,
			})
			return
		}

		if req.Platform != "spotify" && req.Platform != "qq-music" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "不支持的平台，仅支持spotify和qq-music",
				"data":    nil,
			})
			return
		}

		// 调用导入逻辑
		logicInstance := logic.NewImportPlaylistLogic(r.Context(), svcCtx)
		resp, err := logicInstance.ImportPlaylist(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("导入歌单失败: userID=%d, platform=%s, playlistID=%s, error=%v",
				bearerCtx.UserID, req.Platform, req.PlaylistID, err)

			// 根据错误类型返回不同的状态码
			if strings.Contains(err.Error(), "歌单数量限制") {
				httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": err.Error(),
					"data":    nil,
				})
				return
			} else if strings.Contains(err.Error(), "未找到") || strings.Contains(err.Error(), "不存在") {
				httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
					"code":    404,
					"message": err.Error(),
					"data":    nil,
				})
				return
			} else if strings.Contains(err.Error(), "已导入") {
				// 歌单已导入，返回成功但提示已存在
				httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
					"code":    200,
					"message": err.Error(),
					"data":    resp,
				})
				return
			}

			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "导入歌单失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		logx.Infof("导入歌单成功: userID=%d, platform=%s, playlistID=%s, localPlaylistID=%d, imported=%d/%d",
			bearerCtx.UserID, req.Platform, req.PlaylistID, resp.PlaylistID, resp.ImportedCount, resp.TotalCount)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": resp.Message,
			"data":    resp,
		})
	}
}
