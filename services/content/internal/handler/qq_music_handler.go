package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// qqMusicAuthHandler 发起QQ音乐授权
// @Summary      获取QQ音乐授权链接
// @Description  生成OAuth授权链接，支持两种模式：1)默认返回JSON授权链接 2)redirect模式直接跳转
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Param        callback_url  query     string  false  "回调地址（可选）"
// @Param        redirect      query     string  false  "是否直接跳转（true=302重定向，默认false=返回JSON）"
// @Success      200  {object}  httpresp.Standard  "成功返回授权链接"
// @Success      302  {object}  httpresp.Standard  "跳转到QQ音乐授权页面"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Router       /qq-music/auth [get]
// @Security     BearerAuth
func qqMusicAuthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		callbackURL := r.URL.Query().Get("callback_url")
		redirect := r.URL.Query().Get("redirect")

		l := logic.NewQQMusicAuthLogic(r.Context(), svcCtx)
		resp, err := l.Auth(bearerCtx.UserID, callbackURL)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		if redirect == "true" {
			http.Redirect(w, r, resp.AuthURL, http.StatusFound)
		} else {
			httpresp.WriteSuccess(w, map[string]interface{}{
				"auth_url": resp.AuthURL,
			})
		}
	}
}

// qqMusicCallbackHandler QQ音乐回调处理
// @Summary      QQ音乐授权回调
// @Description  接收QQ音乐回调，使用授权码换取AccessToken并绑定账号
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Param        code   query     string  true  "授权码"
// @Param        state  query     string  true  "状态参数"
// @Param        redirect  query  string  false  "是否直接跳转（true=302重定向，默认false=返回JSON）"
// @Success      200  {object}  httpresp.Standard  "绑定成功"
// @Success      302  {object}  httpresp.Standard  "绑定成功，跳转到成功页面"
// @Failure      400  {object}  httpresp.Standard  "参数错误"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /qq-music/callback [get]
// @Security     BearerAuth
func qqMusicCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.QQMusicCallbackReq

		req.Code = r.URL.Query().Get("code")
		req.State = r.URL.Query().Get("state")

		if req.Code == "" || req.State == "" {
			if r.Method == http.MethodPost {
				contentType := r.Header.Get("Content-Type")
				if strings.Contains(contentType, "application/json") {
					json.NewDecoder(r.Body).Decode(&req)
				} else if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
					r.ParseForm()
					req.Code = r.FormValue("code")
					req.State = r.FormValue("state")
				}
			}
		}

		if req.Code == "" || req.State == "" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `缺少必要参数`), nil)
			return
		}

		l := logic.NewQQMusicCallbackLogic(r.Context(), svcCtx)
		err := l.Callback(bearerCtx.UserID, req.Code, req.State)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		redirect := r.URL.Query().Get("redirect")
		if redirect == "true" {
			http.Redirect(w, r, "/binding/success?platform=qq-music", http.StatusFound)
		} else {
			httpresp.WriteSuccessMsg(w, `账号绑定成功`, nil)
		}
	}
}

// qqMusicBindingStatusHandler 查询QQ音乐绑定状态
// @Summary      查询QQ音乐绑定状态
// @Description  查询当前用户的QQ音乐绑定状态和信息
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Router       /qq-music/status [get]
// @Security     BearerAuth
func qqMusicBindingStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewQQMusicBindingStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetStatus(bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}

// qqMusicUnbindHandler 解绑QQ音乐账号
// @Summary      解绑QQ音乐账号
// @Description  解除当前用户的QQ音乐绑定关系
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      404  {object}  httpresp.Standard  "未绑定"
// @Router       /qq-music/unbind [post]
// @Security     BearerAuth
func qqMusicUnbindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewQQMusicUnbindLogic(r.Context(), svcCtx)
		err := l.Unbind(bearerCtx.UserID)
		if err != nil {
			if err.Error() == "未找到绑定的QQ音乐账号" {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `解绑成功`, map[string]interface{}{
			"success": true,
		})
	}
}

// qqMusicPlaylistTracksHandler 获取QQ音乐歌单歌曲
// @Summary      获取QQ音乐歌单歌曲
// @Description  获取指定QQ音乐歌单中的歌曲列表
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Param        playlist_id  query     string  true  "歌单ID"
// @Param        limit        query     int     false  "每页数量（默认20，最大100）"
// @Param        offset       query     int     false  "偏移量（默认0）"
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      400  {object}  httpresp.Standard  "参数错误"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      404  {object}  httpresp.Standard  "未绑定QQ音乐"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /qq-music/playlist/tracks [get]
// @Security     BearerAuth
func qqMusicPlaylistTracksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		playlistID := r.URL.Query().Get("playlist_id")
		if playlistID == "" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `缺少 playlist_id 参数`), nil)
			return
		}

		limit := 20
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}

		offset := 0
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			fmt.Sscanf(offsetStr, "%d", &offset)
		}

		l := logic.NewQQMusicPlaylistLogic(r.Context(), svcCtx)
		resp, err := l.GetPlaylistTracks(bearerCtx.UserID, playlistID, limit, offset)
		if err != nil {
			if strings.Contains(err.Error(), "未找到QQ音乐绑定记录") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `获取歌单歌曲成功`, resp)
	}
}

// qqMusicPlaylistListHandler 获取QQ音乐歌单列表
// @Summary      获取QQ音乐歌单列表
// @Description  获取当前用户的QQ音乐歌单列表
// @Tags         QQ音乐绑定
// @Accept       json
// @Produce      json
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      404  {object}  httpresp.Standard  "未绑定QQ音乐"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /qq-music/playlists [get]
// @Security     BearerAuth
func qqMusicPlaylistListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewQQMusicPlaylistListLogic(r.Context(), svcCtx)
		resp, err := l.GetPlaylists(bearerCtx.UserID)
		if err != nil {
			if err.Error() == "未找到QQ音乐绑定记录" {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `获取歌单列表成功`, resp)
	}
}
