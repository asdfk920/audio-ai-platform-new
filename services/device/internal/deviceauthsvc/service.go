// Package deviceauthsvc 设备认证服务包
// 提供设备云端认证功能，包括 Token 生成、验证、签名校验等
package deviceauthsvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"golang.org/x/crypto/bcrypt"
)

// contextKey 上下文键类型定义
type contextKey string

const bearerTokenContextKey contextKey = "device-bearer-token"
const clientIPContextKey contextKey = "device-client-ip"

// Service 设备认证服务结构体
// 提供设备认证的核心方法，包括 Token 签发、验证、设备身份校验等
type Service struct {
	svcCtx *svc.ServiceContext
}

// Principal 设备认证主体信息结构体
// 包含认证通过后的设备基本信息，用于后续业务逻辑
type Principal struct {
	DeviceID        int64
	DeviceSN        string
	ProductKey      string
	Mac             string
	FirmwareVersion string
	IP              string
	Status          int16
}

// TokenClaims JWT Token 声明结构体
// 用于生成和解析设备认证 Token，包含设备 ID、SN 等信息
type TokenClaims struct {
	DeviceID int64  `json:"device_id"`
	SN       string `json:"sn"`
	jwt.RegisteredClaims
}

// New 创建设备认证服务实例
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *Service: 设备认证服务实例
func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

// WithBearerToken 将 Bearer Token 添加到请求上下文
// 参数 ctx context.Context: 原始上下文
// 参数 token string: Bearer Token
// 返回 context.Context: 添加了 Token 的上下文
func WithBearerToken(ctx context.Context, token string) context.Context {
	token = strings.TrimSpace(token)
	if ctx == nil || token == "" {
		return ctx
	}
	return context.WithValue(ctx, bearerTokenContextKey, token)
}

// BearerTokenFromContext 从上下文中提取 Bearer Token
// 参数 ctx context.Context: 请求上下文
// 返回 string: Bearer Token，如果不存在则返回空字符串
func BearerTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(bearerTokenContextKey).(string)
	return strings.TrimSpace(v)
}

// WithClientIP 将客户端 IP 添加到请求上下文
// 参数 ctx context.Context: 原始上下文
// 参数 clientIP string: 客户端 IP 地址
// 返回 context.Context: 添加了 IP 的上下文
func WithClientIP(ctx context.Context, clientIP string) context.Context {
	clientIP = strings.TrimSpace(clientIP)
	if ctx == nil || clientIP == "" {
		return ctx
	}
	return context.WithValue(ctx, clientIPContextKey, clientIP)
}

// ClientIPFromContext 从上下文中提取客户端 IP
// 参数 ctx context.Context: 请求上下文
// 返回 string: 客户端 IP 地址，如果不存在则返回空字符串
func ClientIPFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(clientIPContextKey).(string)
	return strings.TrimSpace(v)
}

func ExtractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return header
}

func (s *Service) AuthenticateRequest(ctx context.Context, sn, secret, clientIP string) (*Principal, error) {
	if strings.TrimSpace(clientIP) == "" {
		clientIP = ClientIPFromContext(ctx)
	}
	if token := BearerTokenFromContext(ctx); token != "" {
		return s.VerifyDeviceToken(ctx, token)
	}
	return s.AuthenticateBySecret(ctx, sn, secret, clientIP, true, "http_secret")
}

func (s *Service) AuthenticateBySecret(ctx context.Context, sn, secret, clientIP string, requireActive bool, authType string) (*Principal, error) {
	row, err := s.getDevice(ctx, sn)
	if err != nil {
		s.recordFailure(ctx, normalizeSN(sn), authType, clientIP, "device_not_found", errorx.CodeDeviceNotFound)
		return nil, err
	}
	if err := s.ensureNotLocked(ctx, row.SN); err != nil {
		s.recordAudit(ctx, row.SN, authType, clientIP, "locked", errorx.CodeDeviceAuthLocked, map[string]interface{}{})
		return nil, err
	}
	if requireActive {
		if err := repo.ErrIfNotQueryable(row.Status); err != nil {
			s.recordFailure(ctx, row.SN, authType, clientIP, "device_status_invalid", errorx.CodeOf(err))
			return nil, err
		}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.DeviceSecret), []byte(strings.TrimSpace(secret))); err != nil {
		s.recordFailure(ctx, row.SN, authType, clientIP, "secret_mismatch", errorx.CodeDeviceSecretInvalid)
		return nil, errorx.NewDefaultError(errorx.CodeDeviceSecretInvalid)
	}
	s.clearFailures(ctx, row.SN)
	s.recordAudit(ctx, row.SN, authType, clientIP, "success", 0, map[string]interface{}{})
	return principalFromRow(row), nil
}

func (s *Service) VerifyBootstrapSignature(ctx context.Context, sn, secret string, timestamp int64, nonce, signature, clientIP, authType string, payload map[string]string, allowInactive bool) (*Principal, error) {
	row, err := s.getDevice(ctx, sn)
	if err != nil {
		s.recordFailure(ctx, normalizeSN(sn), authType, clientIP, "device_not_found", errorx.CodeDeviceNotFound)
		return nil, err
	}
	if err := s.ensureNotLocked(ctx, row.SN); err != nil {
		s.recordAudit(ctx, row.SN, authType, clientIP, "locked", errorx.CodeDeviceAuthLocked, map[string]interface{}{})
		return nil, err
	}
	if !allowInactive {
		if err := repo.ErrIfNotQueryable(row.Status); err != nil {
			s.recordFailure(ctx, row.SN, authType, clientIP, "device_status_invalid", errorx.CodeOf(err))
			return nil, err
		}
	} else if row.Status == 2 || row.Status == 4 {
		statusErr := repo.ErrIfNotQueryable(row.Status)
		s.recordFailure(ctx, row.SN, authType, clientIP, "device_status_invalid", errorx.CodeOf(statusErr))
		return nil, statusErr
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.DeviceSecret), []byte(strings.TrimSpace(secret))); err != nil {
		s.recordFailure(ctx, row.SN, authType, clientIP, "secret_mismatch", errorx.CodeDeviceSecretInvalid)
		return nil, errorx.NewDefaultError(errorx.CodeDeviceSecretInvalid)
	}
	if err := s.validateTimestamp(timestamp); err != nil {
		s.recordFailure(ctx, row.SN, authType, clientIP, "timestamp_invalid", errorx.CodeDeviceTimestampInvalid)
		return nil, err
	}
	if !verifySignature(secret, payload, nonce, timestamp, signature) {
		s.recordFailure(ctx, row.SN, authType, clientIP, "signature_invalid", errorx.CodeDeviceSignatureInvalid)
		return nil, errorx.NewDefaultError(errorx.CodeDeviceSignatureInvalid)
	}
	s.clearFailures(ctx, row.SN)
	s.recordAudit(ctx, row.SN, authType, clientIP, "success", 0, map[string]interface{}{
		"timestamp": timestamp,
	})
	return principalFromRow(row), nil
}

func (s *Service) IssueDeviceToken(principal *Principal) (string, int64, error) {
	if principal == nil || principal.DeviceID <= 0 || strings.TrimSpace(principal.DeviceSN) == "" {
		return "", 0, errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	now := time.Now()
	expireSeconds := s.tokenExpireSeconds()
	claims := TokenClaims{
		DeviceID: principal.DeviceID,
		SN:       principal.DeviceSN,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   principal.DeviceSN,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireSeconds) * time.Second)),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.tokenSecret()))
	if err != nil {
		return "", 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return signed, expireSeconds, nil
}

func (s *Service) VerifyDeviceToken(ctx context.Context, tokenString string) (*Principal, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, errorx.NewDefaultError(errorx.CodeTokenInvalid)
	}
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.tokenSecret()), nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, errorx.NewDefaultError(errorx.CodeTokenInvalid)
	}
	row, err := s.getDevice(ctx, claims.SN)
	if err != nil {
		return nil, err
	}
	if row.ID != claims.DeviceID {
		return nil, errorx.NewDefaultError(errorx.CodeTokenInvalid)
	}
	if err := repo.ErrIfNotQueryable(row.Status); err != nil {
		return nil, err
	}
	return principalFromRow(row), nil
}

func (s *Service) TouchAuthSuccess(ctx context.Context, principal *Principal, clientIP, firmwareVersion string) {
	if principal == nil || principal.DeviceID <= 0 || s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return
	}
	now := time.Now()
	ip := strings.TrimSpace(clientIP)
	fw := strings.TrimSpace(firmwareVersion)
	if ip == "" {
		ip = principal.IP
	}
	if fw == "" {
		fw = principal.FirmwareVersion
	}
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
UPDATE public.device
SET online_status = 1,
    ip = CASE WHEN COALESCE($1, '') = '' THEN ip ELSE $1 END,
    firmware_version = CASE WHEN COALESCE($2, '') = '' THEN firmware_version ELSE $2 END,
    last_auth_at = $3,
    last_active_at = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $4`, ip, fw, now, principal.DeviceID)
}

func (s *Service) tokenSecret() string {
	if s == nil || s.svcCtx == nil {
		return "device-secret"
	}
	if v := strings.TrimSpace(s.svcCtx.Config.DeviceAuth.TokenSecret); v != "" {
		return v
	}
	if v := strings.TrimSpace(s.svcCtx.Config.Auth.AccessSecret); v != "" {
		return v + "-device"
	}
	return "device-secret"
}

func (s *Service) tokenExpireSeconds() int64 {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceAuth.TokenExpireSeconds <= 0 {
		return 86400
	}
	return s.svcCtx.Config.DeviceAuth.TokenExpireSeconds
}

func (s *Service) timestampToleranceSeconds() int64 {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceAuth.TimestampToleranceSecond <= 0 {
		return 300
	}
	return s.svcCtx.Config.DeviceAuth.TimestampToleranceSecond
}

func (s *Service) failureThreshold() int {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceAuth.FailureThreshold <= 0 {
		return 5
	}
	return s.svcCtx.Config.DeviceAuth.FailureThreshold
}

func (s *Service) lockSeconds() int64 {
	if s == nil || s.svcCtx == nil || s.svcCtx.Config.DeviceAuth.LockSeconds <= 0 {
		return 900
	}
	return s.svcCtx.Config.DeviceAuth.LockSeconds
}

func (s *Service) validateTimestamp(ts int64) error {
	if ts <= 0 {
		return errorx.NewDefaultError(errorx.CodeDeviceTimestampInvalid)
	}
	var target time.Time
	if ts > 1_000_000_000_000 {
		target = time.UnixMilli(ts)
	} else {
		target = time.Unix(ts, 0)
	}
	if target.Before(time.Now().Add(-time.Duration(s.timestampToleranceSeconds())*time.Second)) ||
		target.After(time.Now().Add(time.Duration(s.timestampToleranceSeconds())*time.Second)) {
		return errorx.NewDefaultError(errorx.CodeDeviceTimestampInvalid)
	}
	return nil
}

func (s *Service) getDevice(ctx context.Context, sn string) (*repo.DeviceAuthRow, error) {
	row, err := repo.GetDeviceForMQTTAuth(ctx, s.svcCtx.DB, sn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewDefaultError(errorx.CodeDeviceNotFound)
		}
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return row, nil
}

func (s *Service) ensureNotLocked(ctx context.Context, sn string) error {
	var count int
	err := s.svcCtx.DB.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM public.device_auth_failures
WHERE sn = $1
  AND created_at >= $2`, normalizeSN(sn), time.Now().Add(-time.Duration(s.lockSeconds())*time.Second)).Scan(&count)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if count >= s.failureThreshold() {
		return errorx.NewDefaultError(errorx.CodeDeviceAuthLocked)
	}
	return nil
}

func (s *Service) clearFailures(ctx context.Context, sn string) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return
	}
	_, _ = s.svcCtx.DB.ExecContext(ctx, `DELETE FROM public.device_auth_failures WHERE sn = $1`, normalizeSN(sn))
}

func (s *Service) recordFailure(ctx context.Context, sn, authType, clientIP, reason string, code int) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return
	}
	sn = normalizeSN(sn)
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_auth_failures (sn, reason, client_ip)
VALUES ($1, $2, $3)`, sn, truncate(reason, 50), truncate(clientIP, 64))
	s.recordAudit(ctx, sn, authType, clientIP, "failed", code, map[string]interface{}{
		"reason": reason,
	})
}

func (s *Service) recordAudit(ctx context.Context, sn, authType, clientIP, outcome string, code int, detail map[string]interface{}) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return
	}
	if detail == nil {
		detail = map[string]interface{}{}
	}
	payload, _ := json.Marshal(detail)
	_, _ = s.svcCtx.DB.ExecContext(ctx, `
INSERT INTO public.device_auth_audit (sn, auth_type, client_ip, outcome, error_code, detail)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		normalizeSN(sn), truncate(authType, 32), truncate(clientIP, 64), truncate(outcome, 32), code, string(payload))
}

func principalFromRow(row *repo.DeviceAuthRow) *Principal {
	if row == nil {
		return nil
	}
	return &Principal{
		DeviceID:        row.ID,
		DeviceSN:        row.SN,
		ProductKey:      row.ProductKey,
		Mac:             row.Mac,
		FirmwareVersion: row.FirmwareVersion,
		IP:              row.IP,
		Status:          row.Status,
	}
}

func verifySignature(secret string, payload map[string]string, nonce string, timestamp int64, signature string) bool {
	signature = strings.ToLower(strings.TrimSpace(signature))
	if signature == "" {
		return false
	}
	body := canonicalPayload(payload, nonce, timestamp)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(body))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func canonicalPayload(payload map[string]string, nonce string, timestamp int64) string {
	keys := make([]string, 0, len(payload)+2)
	clean := make(map[string]string, len(payload)+2)
	for k, v := range payload {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		clean[k] = strings.TrimSpace(v)
		keys = append(keys, k)
	}
	clean["nonce"] = strings.TrimSpace(nonce)
	clean["timestamp"] = strconv.FormatInt(timestamp, 10)
	keys = append(keys, "nonce", "timestamp")
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"="+clean[key])
	}
	return strings.Join(parts, "&")
}

func normalizeSN(sn string) string {
	return strings.ToUpper(strings.TrimSpace(sn))
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
