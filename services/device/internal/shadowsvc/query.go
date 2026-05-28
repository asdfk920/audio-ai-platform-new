package shadowsvc

import (
	"context"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// GetShadowViewBySN 按 SN 查询设备影子（Redis v1 Hash + PostgreSQL，与批量更新同源）
func (s *Service) GetShadowViewBySN(ctx context.Context, sn string) (*View, error) {
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if sn == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "device_sn 不能为空")
	}
	if s.svcCtx == nil || s.svcCtx.DeviceRepo == nil {
		return nil, errorx.NewDefaultError(errorx.CodeInternalError)
	}

	dev, err := s.svcCtx.DeviceRepo.FindBySn(ctx, sn)
	if err != nil {
		return nil, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if dev == nil {
		return nil, errorx.NewCodeError(errorx.CodeNotFound, "设备不存在")
	}

	online := dev.OnlineStatus == 1
	return s.buildView(ctx, dev.ID, sn, online)
}

// GetShadowViewBySNFromDB 仅从 PostgreSQL device_shadow 表查询（不读 Redis）
func (s *Service) GetShadowViewBySNFromDB(ctx context.Context, sn string) (*View, error) {
	return s.GetShadowViewBySNDirect(ctx, sn)
}

// onlineFromReported 从 reported JSON 解析在线状态（仅 DB 查询路径使用）
func onlineFromReported(reported map[string]interface{}) bool {
	if reported == nil {
		return false
	}
	v, ok := reported["online"]
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "1" || strings.EqualFold(s, "true") || strings.EqualFold(s, "online")
	}
	return false
}
