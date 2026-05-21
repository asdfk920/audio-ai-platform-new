package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// artistSubscribeHandler 订阅艺术家处理器
// @Summary      订阅艺术家
// @Description  用户订阅指定的艺术家，需要登录
// @Tags         艺术家管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ArtistSubscribeReq  true  "订阅艺术家请求"
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      400  {object}  httpresp.Standard  "参数错误"
// @Failure      404  {object}  httpresp.Standard  "艺术家不存在"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /artists/subscribe [post]
// @Security     BearerAuth
func artistSubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.ArtistSubscribeReq
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
			artistIDStr := r.FormValue("artist_id")
			if artistIDStr == "" {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 不能为空`), nil)
				return
			}

			artistID, err := strconv.ParseInt(artistIDStr, 10, 64)
			if err != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 格式错误`), nil)
				return
			}
			req.ArtistID = artistID
		}

		if req.ArtistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 必须大于 0`), nil)
			return
		}

		logicInstance := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := logicInstance.Subscribe(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("订阅艺术家失败：userID=%d, artistID=%d, error=%v",
				bearerCtx.UserID, req.ArtistID, err)

			if strings.Contains(err.Error(), "不存在") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}

			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		logx.Infof("订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
			bearerCtx.UserID, req.ArtistID, resp.ArtistName)

		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// artistUnsubscribeHandler 取消订阅艺术家处理器
// @Summary      取消订阅艺术家
// @Description  用户取消订阅指定的艺术家，需要登录
// @Tags         艺术家管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ArtistUnsubscribeReq  true  "取消订阅艺术家请求"
// @Success      200  {object}  httpresp.Standard  "成功"
// @Failure      401  {object}  httpresp.Standard  "未登录"
// @Failure      400  {object}  httpresp.Standard  "参数错误"
// @Failure      404  {object}  httpresp.Standard  "艺术家不存在"
// @Failure      500  {object}  httpresp.Standard  "服务器错误"
// @Router       /artists/unsubscribe [post]
// @Security     BearerAuth
func artistUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.ArtistUnsubscribeReq
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
			artistIDStr := r.FormValue("artist_id")
			if artistIDStr == "" {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 不能为空`), nil)
				return
			}

			artistID, err := strconv.ParseInt(artistIDStr, 10, 64)
			if err != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 格式错误`), nil)
				return
			}
			req.ArtistID = artistID
		}

		if req.ArtistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `艺术家 ID 必须大于 0`), nil)
			return
		}

		logicInstance := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := logicInstance.Unsubscribe(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("取消订阅艺术家失败：userID=%d, artistID=%d, error=%v",
				bearerCtx.UserID, req.ArtistID, err)

			if strings.Contains(err.Error(), "不存在") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
				return
			}

			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		logx.Infof("取消订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
			bearerCtx.UserID, req.ArtistID, resp.ArtistName)

		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}
