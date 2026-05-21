package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// spotifyFavoriteListHandler 获取 Spotify 收藏歌曲列表处理器
// @Summary      获取 Spotify 收藏歌曲列表
// @Description  获取用户 Spotify 账号的收藏歌曲列表，需要登录并已绑定 Spotify
// @Tags         Spotify
// @Accept       json
// @Produce      json
// @Param        limit   query     int  false  "每页数量" default(20)
// @Param        offset  query     int  false  "偏移量" default(0)
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      403  {object}  httpresp.Standard  "未绑定 Spotify"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /spotify/favorites [get]
// @Security     BearerAuth
func spotifyFavoriteListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 GET`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

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

		logicInstance := logic.NewSpotifyFavoriteLogic(r.Context(), svcCtx)
		resp, err := logicInstance.GetFavoriteTracks(bearerCtx.UserID, limit, offset)
		if err != nil {
			logx.Errorf("获取 Spotify 收藏歌曲失败：userID=%d, limit=%d, offset=%d, error=%v",
				bearerCtx.UserID, limit, offset, err)

			if err.Error() == "未找到 Spotify 绑定信息" {
				httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.MsgForbidden, `请先绑定 Spotify 账号`), nil)
				return
			}

			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		logx.Infof("获取 Spotify 收藏歌曲成功：userID=%d, total=%d, limit=%d, offset=%d",
			bearerCtx.UserID, resp.Total, resp.Limit, resp.Offset)

		httpresp.WriteSuccess(w, resp)
	}
}
