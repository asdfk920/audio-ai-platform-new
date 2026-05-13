package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

func WillMessageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logx.Errorf("WillMessageHandler 发生panic: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
					Code:    500,
					Message: "系统内部错误，请联系管理员",
				})
			}
		}()

		var req types.WillMessageReq

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logx.Error("解析请求JSON失败: ", err)
			w.WriteHeader(http.StatusBadRequest)
			httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
				Code:    400,
				Message: "JSON格式错误: " + err.Error(),
			})
			return
		}

		if req.SN == "" {
			logx.Error("缺少必填参数: sn")
			w.WriteHeader(http.StatusBadRequest)
			httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
				Code:    400,
				Message: "缺少必填参数: sn",
			})
			return
		}

		if req.Status == "" {
			logx.Error("缺少必填参数: status")
			w.WriteHeader(http.StatusBadRequest)
			httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
				Code:    400,
				Message: "缺少必填参数: status",
			})
			return
		}

		l := logic.NewWillMessageLogic(r.Context(), svcCtx)

		err := l.ProcessWillMessage(&logic.WillMessageReq{
			SN:     req.SN,
			Status: req.Status,
			Time:   req.Time,
			Reason: req.Reason,
		})

		if err != nil {
			logx.Error("处理WILL消息失败: sn=", req.SN, ", error=", err)
			w.WriteHeader(http.StatusInternalServerError)
			httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
				Code:    500,
				Message: err.Error(),
			})
			return
		}

		httpx.OkJsonCtx(r.Context(), w, types.WillMessageResp{
			Code:    200,
			Message: "success",
		})
	}
}
