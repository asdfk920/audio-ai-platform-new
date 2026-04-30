package errorx

import "net/http"

// HTTPStatusForCode 将业务错误码映射为 HTTP 状态码，便于网关与监控区分语义。
// 未列出的 1xxx/2xxx/3xxx 默认 400；9xxx 中数据库/Redis 为 503，其余 500。
func HTTPStatusForCode(code int) int {
	switch code {
	case CodeTokenInvalid, CodeTokenExpired, CodePasswordError, CodeRefreshTokenSuperseded,
		CodeDeviceSecretInvalid, CodeDeviceSignatureInvalid, CodeDeviceTimestampInvalid:
		return http.StatusUnauthorized
	case CodeIPDenied, CodeAccountLocked, CodeUserAccountDisabled, CodeRealNameAdminUnauthorized, CodeCancellationCooling, CodeRealNameRebindLocked, CodeDeviceSnBlacklisted, CodeDeviceUnbindForbidden, CodeDeviceUnbindNotOwner,
		CodeDeviceAuthLocked,
		CodeDeviceDisabled, CodeDeviceInactive, CodeDeviceScrapped:
		return http.StatusForbidden
	case CodeUserNotFound, CodeDeviceNotFound, CodeContentNotFound, CodeRealNameRecordNotFound, CodeAccountCancelled, CodeMemberOrderNotFound, CodeMemberPackageNotFound:
		return http.StatusNotFound
	case CodeUserExists, CodeNicknameTaken, CodeDeviceExists, CodeRebindContactConflict, CodeCancellationAlreadyPending, CodeDeviceAlreadyRegistered, CodeDeviceBoundByOther,
		CodeFamilyAlreadyExists, CodeFamilyMemberExists, CodeDeviceShareExists:
		return http.StatusConflict
	case CodeVerifyCodeLimit, CodeRegisterRateLimit, CodeRegisterBusy, CodeLoginBusy, CodeResetPasswordRateLimit,
		CodeRegisterDuplicateSubmit, CodeUpdateProfileRateLimit, CodeRefreshRateLimit,
		CodeRebindContactRateLimit, CodeRebindContactCooldown, CodeRealNameSubmitBusy,
		CodeCancellationWithdrawRateLimit, CodeDeviceRegisterRateLimit, CodeDeviceStatusQueryRateLimit,
		CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeVerifyCodeInvalid, CodeInvalidParam, CodeInvalidEmail, CodeInvalidMobile,
		CodePasswordReuse, CodeWeakPassword, CodeInvalidNickname, CodeInvalidProfileAvatar, CodeDeviceOffline, CodeUploadFailed, CodeParamError,
		CodeRealNameInvalidPayload, CodeRealNameAlreadyVerified, CodeRealNameInProgress, CodeRealNameInvalidReviewState,
		CodeCancellationNotInCooling, CodeDeviceSnInvalid, CodeDeviceProductInvalid, CodeDeviceNotBound,
		CodeMemberPackageDisabled, CodeMemberOrderNotPending, CodeMemberInsufficientBalance, CodeMemberPayCallbackInvalid,
		CodeFamilyInviteInvalid, CodeDeviceShareInvalid, CodeDeviceShareExpired,
		CodeMemberNoSubscription, CodeMemberPermanentNoUnsubscribe, CodeMemberUnsubscribeNotPending, CodeMemberExpiredNoUnsubscribe:
		return http.StatusBadRequest
	case CodeFamilyNoPermission, CodeDeviceShareNoPermission, CodeDeviceShareForbiddenReshare:
		return http.StatusForbidden
	case CodeFamilyNotFound:
		return http.StatusNotFound
	case CodeProcessFailed, CodeSystemError:
		return http.StatusInternalServerError
	case CodeDatabaseError, CodeRedisError, CodeRealNameProviderError:
		return http.StatusServiceUnavailable
	default:
		if code >= 1000 && code < 4000 {
			return http.StatusBadRequest
		}
		if code >= 9000 {
			return http.StatusInternalServerError
		}
		return http.StatusBadRequest
	}
}
