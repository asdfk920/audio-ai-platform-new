package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceVolumeHandler Device volume adjustment unified interface handler
// POST /api/device/cmd/volume
// Purpose: User sends volume adjustment instruction via App (requires JWT authentication)
//
// Flow:
//  1. Receive POST request, parse Authorization header to get user Token
//  2. Parse request body JSON data, get sn, target_volume, action parameters
//  3. Validate Token and parameter format
//  4. Verify user permission (query user_device_bind table)
//  5. Query device current volume value (Redis + MySQL)
//  6. Calculate target volume
//  7. Check device online status (Redis + MySQL)
//  8. Online: deliver immediately via WebSocket; Offline: cache instruction
//  9. Record command log and return response
func deviceVolumeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "Only POST method supported",
				"data": nil,
			})
			return
		}

		var req types.DeviceVolumeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "Request body format error",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceVolumeLogic(r.Context(), svcCtx)
		resp, err := l.DeviceVolume(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "Operation successful",
			"data": resp,
		})
	})
}

// DeviceVolumeHandler Device volume adjustment unified interface
// @Summary      Device Volume Adjustment
// @Description  User sends volume adjustment instruction to device via App
// @Tags         Device Control
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceVolumeReq  true  "Device volume adjustment request"
// @Success      200  {object}  types.DeviceVolumeResp  "Success"
// @Failure      400  {object}  errorx.Response  "Parameter error"
// @Failure      401  {object}  errorx.Response  "Not logged in"
// @Failure      500  {object}  errorx.Response  "Server error"
// @Router       /api/device/cmd/volume [post]
// @Security     BearerAuth
func DeviceVolumeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceVolumeHandler(svcCtx)
}