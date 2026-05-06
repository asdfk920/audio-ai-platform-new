package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @createDownloadRecord 创建下载记录
// @Summary      创建下载记录
// @Description  用户点击下载时创建下载记录
// @Tags         下载服务
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/download/create [post]
// @Security     BearerAuth
// createDownloadRecordHandler 创建下载记录处理器
// POST /api/v1/user/download/create
// 用途：用户点击下载时创建下载记录，返回下载地址和断点位置
func createDownloadRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		userID := ctxuser.ParseUserID(r.Context())
		if userID == 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "未登录",
				"data": nil,
			})
			return
		}

		var req types.CreateDownloadRecordReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误",
				"data": nil,
			})
			return
		}

		l := logic.NewCreateDownloadRecordLogic(r.Context(), svcCtx)
		resp, err := l.CreateDownloadRecord(userID, &req)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code": 500,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "成功",
			"data": resp,
		})
	}
}
