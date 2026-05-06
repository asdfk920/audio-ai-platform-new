package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// spotifyFavoriteListHandler 获取 Spotify 收藏歌曲列表处理器
// @Summary      获取 Spotify 收藏歌曲列表
// @Description  获取用户 Spotify 账号的收藏歌曲列表，需要登录并已绑定 Spotify
// @Tags         Spotify
// @Accept       json
// @Produce      json
// @Param        limit   query     int  false  "每页数量" default(20)
// @Param        offset  query     int  false  "偏移量" default(0)
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      403  {object}  map[string]interface{}  "未绑定 Spotify"
// @Failure      500  {object}  map[string]interface{}  "服务器错误"
// @Router       /spotify/favorites [get]
// @Security     BearerAuth
func spotifyFavoriteListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
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

		// 解析分页参数
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := 20
		offset := 0

		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		if offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		// 调用 Logic 层获取收藏列表
		logicInstance := logic.NewSpotifyFavoriteLogic(r.Context(), svcCtx)
		resp, err := logicInstance.GetFavoriteTracks(bearerCtx.UserID, limit, offset)
		if err != nil {
			logx.Errorf("获取 Spotify 收藏歌曲失败：userID=%d, limit=%d, offset=%d, error=%v",
				bearerCtx.UserID, limit, offset, err)

			// 检查是否未绑定 Spotify
			if err.Error() == "未找到 Spotify 绑定信息" {
				httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": "请先绑定 Spotify 账号",
					"data":    nil,
				})
				return
			}

			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "获取收藏歌曲失败：" + err.Error(),
				"data":    nil,
			})
			return
		}

		logx.Infof("获取 Spotify 收藏歌曲成功：userID=%d, total=%d, limit=%d, offset=%d",
			bearerCtx.UserID, resp.Total, resp.Limit, resp.Offset)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    resp,
		})
	}
}
