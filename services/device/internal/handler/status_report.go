package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/reg"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/pkg/ip"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

const headerDeviceSecret = "X-Device-Secret"

type statusReportReq struct {
	DeviceID     string      `json:"device_id"`
	ReportType   string      `json:"report_type"`
	BatteryLevel int         `json:"battery_level"`
	StorageUsed  int64       `json:"storage_used"`
	StorageTotal int64       `json:"storage_total"`
	SpeakerCount int         `json:"speaker_count"`
	UWB          uwbObj      `json:"uwb"`
	Acoustic     acousticObj `json:"acoustic"`
	ReportedAt   string      `json:"reported_at"`
}

type uwbObj struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
	Z *float64 `json:"z"`
}

type acousticObj struct {
	Calibrated *int     `json:"calibrated"`
	Offset     *float64 `json:"offset"`
}

func statusReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	auth := deviceauthsvc.New(svcCtx)
	max := 60
	if svcCtx != nil && svcCtx.Config.StatusReportHTTP.RateLimitPerMinute > 0 {
		max = svcCtx.Config.StatusReportHTTP.RateLimitPerMinute
	}
	limiter := reg.NewIPLimiter(time.Minute, max)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusMethodNotAllowed, errorx.Error(errorx.CodeInvalidParam, "仅支持 POST"))
			return
		}
		var req statusReportReq
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		if err := dec.Decode(&req); err != nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体不是合法 JSON"))
			return
		}

		sn := strings.TrimSpace(req.DeviceID)
		if sn == "" {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "device_id 必填"))
			return
		}
		snUpper := strings.ToUpper(sn)

		clientIP := ip.ClientIPTrusted(r, svcCtx.RegisterTrustedNets)
		ctx := r.Context()
		ctx = deviceauthsvc.WithClientIP(ctx, clientIP)
		secret := strings.TrimSpace(r.Header.Get(headerDeviceSecret))
		if tok := deviceauthsvc.ExtractBearerToken(r.Header.Get("Authorization")); tok != "" {
			ctx = deviceauthsvc.WithBearerToken(ctx, tok)
		}

		principal, err := auth.AuthenticateRequest(ctx, sn, secret, clientIP)
		if err != nil {
			writeAuthErr(w, r, err)
			return
		}
		if !strings.EqualFold(strings.TrimSpace(principal.DeviceSN), sn) {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "device_id 与凭证不匹配"))
			return
		}

		if !limiter.Allow(snUpper) {
			httpx.WriteJsonCtx(r.Context(), w, errorx.HTTPStatusForCode(errorx.CodeRateLimited), errorx.Error(errorx.CodeRateLimited, ""))
			return
		}

		if req.BatteryLevel < 0 || req.BatteryLevel > 100 {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "battery_level 须在 0-100"))
			return
		}
		if req.StorageUsed < 0 || req.StorageTotal < 0 {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "storage 不能为负"))
			return
		}
		if req.SpeakerCount < 0 {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "speaker_count 不能为负"))
			return
		}

		reportedAt, err := parseReportedAt(req.ReportedAt)
		if err != nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "reported_at 时间格式无效"))
			return
		}

		reportType := strings.TrimSpace(strings.ToLower(req.ReportType))
		if reportType == "" {
			reportType = "auto"
		}
		if reportType != "auto" && reportType != "manual" && reportType != "sync" {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "report_type 仅支持 auto、manual、sync"))
			return
		}

		acousticCal := int16(0)
		if req.Acoustic.Calibrated != nil {
			v := *req.Acoustic.Calibrated
			if v != 0 && v != 1 {
				httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "acoustic.calibrated 仅支持 0 或 1"))
				return
			}
			acousticCal = int16(v)
		}

		var off *float64
		if req.Acoustic.Offset != nil {
			off = req.Acoustic.Offset
		}

		if svcCtx == nil || svcCtx.DB == nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, errorx.Error(errorx.CodeSystemError, "数据库未就绪"))
			return
		}

		err = repo.InsertDeviceStatusLog(ctx, svcCtx.DB, principal.DeviceID, principal.DeviceSN,
			req.BatteryLevel, req.StorageUsed, req.StorageTotal, req.SpeakerCount,
			req.UWB.X, req.UWB.Y, req.UWB.Z, acousticCal, off, reportedAt, reportType)
		if err != nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, errorx.Error(errorx.CodeDatabaseError, "写入失败"))
			return
		}

		auth.TouchAuthSuccess(ctx, principal, clientIP, "")

		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "状态上报成功",
		})
	}
}

func writeAuthErr(w http.ResponseWriter, r *http.Request, err error) {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		st := errorx.HTTPStatusForCode(ce.GetCode())
		httpx.WriteJsonCtx(r.Context(), w, st, errorx.Error(ce.GetCode(), ce.GetMsg()))
		return
	}
	httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, errorx.Error(errorx.CodeSystemError, ""))
}

func parseReportedAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty")
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, e := time.ParseInLocation(layout, s, time.Local); e == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("parse")
}
