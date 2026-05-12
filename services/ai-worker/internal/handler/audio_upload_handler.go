package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// AudioUploadHandler 音频文件上传处理
// @Summary 上传音频文件到 OSS
// @Description 用户上传音频文件到对象存储，返回音频 URL，用于后续音轨分离
// @Tags 音频上传
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "音频文件 (mp3/wav/flac/aac/ogg/m4a)"
// @Success 200 {object} logic.AudioUploadResp
// @Router /api/v1/audio/upload [post]
func AudioUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "方法不允许，请使用 POST 方法")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 500*1024*1024)
		if err := r.ParseMultipartForm(500 * 1024 * 1024); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("解析表单失败：%v", err))
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "请选择要上传的文件")
			return
		}
		defer file.Close()

		ext := getExt(header.Filename)
		if !isAllowedAudioFormat(ext) {
			writeJSONError(w, http.StatusBadRequest, "不支持的音频格式，仅支持 mp3/wav/flac/aac/ogg/m4a")
			return
		}

		if header.Size > 500*1024*1024 {
			writeJSONError(w, http.StatusBadRequest, "文件大小超过限制（最大500MB）")
			return
		}

		userID := auth.GetUserIDFromContext(r.Context())
		if userID == 0 {
			writeJSONError(w, http.StatusUnauthorized, "用户未登录或登录已过期")
			return
		}

		uploadLogic := logic.NewAudioUploadLogic(svcCtx)

		resp, err := uploadLogic.UploadAudio(file, header.Filename, header.Size, userID)
		if err != nil {
			errorMsg := err.Error()
			statusCode := http.StatusInternalServerError
			if errorMsg == "文件为空" || errorMsg == "读取文件内容失败" || errorMsg == "不支持的音频格式" {
				statusCode = http.StatusBadRequest
			}
			writeJSONError(w, statusCode, errorMsg)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"code":    statusCode,
		"success": false,
	})
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
	}
	return ""
}

func isAllowedAudioFormat(ext string) bool {
	allowed := map[string]bool{
		"mp3":  true,
		"wav":  true,
		"flac": true,
		"aac":  true,
		"ogg":  true,
		"m4a":  true,
	}
	return allowed[ext]
}
