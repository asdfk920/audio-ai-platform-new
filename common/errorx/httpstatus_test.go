package errorx

import (
	"net/http"
	"testing"
)

func TestHTTPStatusForCode(t *testing.T) {
	tests := []struct {
		code int
		want int
	}{
		{CodeTokenInvalid, http.StatusUnauthorized},
		{CodeTokenExpired, http.StatusUnauthorized},
		{CodePasswordError, http.StatusUnauthorized},
		{CodeUserNotFound, http.StatusNotFound},
		{CodeUserExists, http.StatusConflict},
		{CodeNicknameTaken, http.StatusConflict},
		{CodeVerifyCodeLimit, http.StatusTooManyRequests},
		{CodeResetPasswordRateLimit, http.StatusTooManyRequests},
		{CodeInvalidParam, http.StatusBadRequest},
		{CodePasswordReuse, http.StatusBadRequest},
		{CodeWeakPassword, http.StatusBadRequest},
		{CodeInvalidNickname, http.StatusBadRequest},
		{CodeRegisterDuplicateSubmit, http.StatusTooManyRequests},
		{CodeUpdateProfileRateLimit, http.StatusTooManyRequests},
		{CodeInvalidProfileAvatar, http.StatusBadRequest},
		{CodeIPDenied, http.StatusForbidden},
		{CodeDeviceNotBound, http.StatusBadRequest},
		{CodeDeviceRegisterRateLimit, http.StatusTooManyRequests},
		{CodeDeviceUnbindNotOwner, http.StatusForbidden},
		{CodeDatabaseError, http.StatusServiceUnavailable},
		{CodeRedisError, http.StatusServiceUnavailable},
		{CodeSystemError, http.StatusInternalServerError},
		{999, http.StatusBadRequest},
	}
	for _, tt := range tests {
		if got := HTTPStatusForCode(tt.code); got != tt.want {
			t.Errorf("HTTPStatusForCode(%d) = %d, want %d", tt.code, got, tt.want)
		}
	}
}
