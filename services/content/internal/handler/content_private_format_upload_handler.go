package handler

import (
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
)

// contentPrivateFormatUploadHandler 私有格式文件上传（multipart：file + 元数据）
// @Summary      私有格式文件上传 OSS/对象存储
// @Description  字段：file(二进制)、fileName、fileSize、fileMd5、format；可选 duration（秒）。无需登录。
// @Tags         内容
// @Accept       multipart/form-data
// @Produce      json
// @Success      200  {object}  httpresp.Standard  "data 仅含 url"
// @Failure      400  {object}  httpresp.Standard
// @Failure      500  {object}  httpresp.Standard
// @Router       /upload/private-format [post]
func contentPrivateFormatUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 multipart：`+err.Error()), nil)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `缺少表单字段 file`), nil)
			return
		}
		defer file.Close()

		fileName := strings.TrimSpace(r.FormValue("fileName"))
		fileSize := strings.TrimSpace(r.FormValue("fileSize"))
		fileMd5 := strings.TrimSpace(r.FormValue("fileMd5"))
		format := strings.TrimSpace(r.FormValue("format"))
		duration := strings.TrimSpace(r.FormValue("duration"))

		l := logic.NewPrivateFormatUploadLogic(r.Context(), svcCtx)
		out, err := l.Upload(fileName, fileSize, fileMd5, format, duration, file)
		if err != nil {
			status, msg := mapPrivateFormatUploadErr(err.Error())
			httpresp.Write(w, status, msg, nil)
			return
		}

		httpresp.WriteSuccess(w, out)
	}
}

func mapPrivateFormatUploadErr(errMsg string) (int, string) {
	s := strings.TrimSpace(errMsg)
	if s == "" {
		return http.StatusBadRequest, httpresp.MsgBadRequest
	}
	if strings.Contains(s, "上传对象存储失败") || strings.Contains(s, "对象存储未初始化") ||
		strings.Contains(s, "content_files 归档") || strings.Contains(s, "mkdir") ||
		strings.Contains(s, "write：") {
		return http.StatusInternalServerError, httpresp.MsgInternal
	}
	return http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, s)
}
