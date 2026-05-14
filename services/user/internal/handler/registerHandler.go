// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 用户注册（需先发验证码；按邮箱/手机判断是否已注册；密码加盐存储；支持头像上传）
func registerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq

		// 解析 multipart/form-data（支持文件上传）
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 提取文本字段
		req.Email = r.FormValue("email")
		req.Mobile = r.FormValue("mobile")
		req.Password = r.FormValue("password")
		req.VerifyCode = r.FormValue("verify_code")
		req.Nickname = r.FormValue("nickname")

		l := logic.NewRegisterLogic(r.Context(), svcCtx)
		resp, err := l.RegisterWithFile(&req, r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
