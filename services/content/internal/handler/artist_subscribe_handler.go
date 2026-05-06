package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// artistSubscribeHandler 订阅艺术家处理器
// @Summary      订阅艺术家
// @Description  用户订阅指定的艺术家，需要登录
// @Tags         艺术家管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ArtistSubscribeReq  true  "订阅艺术家请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Failure      404  {object}  map[string]interface{}  "艺术家不存在"
// @Failure      500  {object}  map[string]interface{}  "服务器错误"
// @Router       /artists/subscribe [post]
// @Security     BearerAuth
func artistSubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.ArtistSubscribeReq
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
					"message": "JSON 格式错误",
					"data":    nil,
				})
				return
			}
		} else {
			r.ParseForm()
			artistIDStr := r.FormValue("artist_id")
			if artistIDStr == "" {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "艺术家 ID 不能为空",
					"data":    nil,
				})
				return
			}

			artistID, err := strconv.ParseInt(artistIDStr, 10, 64)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "艺术家 ID 格式错误",
					"data":    nil,
				})
				return
			}
			req.ArtistID = artistID
		}

		// 验证必填参数
		if req.ArtistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "艺术家 ID 必须大于 0",
				"data":    nil,
			})
			return
		}

		// 调用订阅逻辑
		logicInstance := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := logicInstance.Subscribe(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("订阅艺术家失败：userID=%d, artistID=%d, error=%v",
				bearerCtx.UserID, req.ArtistID, err)

			// 根据错误类型返回不同的状态码
			if strings.Contains(err.Error(), "不存在") {
				httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
					"code":    404,
					"message": err.Error(),
					"data":    nil,
				})
				return
			}

			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "订阅失败：" + err.Error(),
				"data":    nil,
			})
			return
		}

		logx.Infof("订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
			bearerCtx.UserID, req.ArtistID, resp.ArtistName)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": resp.Message,
			"data":    resp,
		})
	}
}

// artistUnsubscribeHandler 取消订阅艺术家处理器
// @Summary      取消订阅艺术家
// @Description  用户取消订阅指定的艺术家，需要登录
// @Tags         艺术家管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.ArtistUnsubscribeReq  true  "取消订阅艺术家请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Failure      404  {object}  map[string]interface{}  "艺术家不存在"
// @Failure      500  {object}  map[string]interface{}  "服务器错误"
// @Router       /artists/unsubscribe [post]
// @Security     BearerAuth
func artistUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.ArtistUnsubscribeReq
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
					"message": "JSON 格式错误",
					"data":    nil,
				})
				return
			}
		} else {
			r.ParseForm()
			artistIDStr := r.FormValue("artist_id")
			if artistIDStr == "" {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "艺术家 ID 不能为空",
					"data":    nil,
				})
				return
			}

			artistID, err := strconv.ParseInt(artistIDStr, 10, 64)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "艺术家 ID 格式错误",
					"data":    nil,
				})
				return
			}
			req.ArtistID = artistID
		}

		// 验证必填参数
		if req.ArtistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "艺术家 ID 必须大于 0",
				"data":    nil,
			})
			return
		}

		// 调用取消订阅逻辑
		logicInstance := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := logicInstance.Unsubscribe(&req, bearerCtx.UserID)
		if err != nil {
			logx.Errorf("取消订阅艺术家失败：userID=%d, artistID=%d, error=%v",
				bearerCtx.UserID, req.ArtistID, err)

			// 根据错误类型返回不同的状态码
			if strings.Contains(err.Error(), "不存在") {
				httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
					"code":    404,
					"message": err.Error(),
					"data":    nil,
				})
				return
			}

			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "取消订阅失败：" + err.Error(),
				"data":    nil,
			})
			return
		}

		logx.Infof("取消订阅艺术家成功：userID=%d, artistID=%d, artistName=%s",
			bearerCtx.UserID, req.ArtistID, resp.ArtistName)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": resp.Message,
			"data":    resp,
		})
	}
}
