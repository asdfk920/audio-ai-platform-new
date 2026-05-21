package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDeviceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDeviceLogic {
	return &UpdateDeviceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateDevice 更新设备辅助信息（备注名、位置、分组、场景）
//
// 完整流程：
//  1. 从JWT Token获取当前用户ID
//  2. 校验请求参数合法性
//  3. 验证设备绑定关系和用户权限
//  4. 更新设备的辅助信息到数据库
//  5. 返回更新后的设备信息
//
// 参数 req *types.UpdateDeviceReq: 设备更新请求参数
// 返回 *types.UpdateDeviceResp: 设备更新响应数据
// 返回 error: 错误信息（如果失败）
func (l *UpdateDeviceLogic) UpdateDevice(req *types.UpdateDeviceReq) (*types.UpdateDeviceResp, error) {

	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		l.Logger.Errorf("UpdateDevice: 获取用户ID失败, context value=%+v, type=%T", l.ctx.Value("userId"), l.ctx.Value("userId"))
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	l.Logger.Infof("UpdateDevice: 开始处理请求, userId=%d, sn=%s", userId, req.Sn)

	if err := l.validateRequestParams(req); err != nil {
		return nil, err
	}

	bindRow, err := l.svcCtx.DeviceBind.FindActiveBindBySN(l.ctx, req.Sn)
	if err != nil {
		l.Logger.Errorf("UpdateDevice: 查询设备绑定关系失败, userId=%d, sn=%s, err=%v", userId, req.Sn, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}

	if bindRow == nil || bindRow.UserID != userId {
		l.Logger.Errorf("UpdateDevice: 用户无权限操作该设备或设备未绑定, userId=%d, sn=%s, bindRowNil=%v, bindUserId=%d",
			userId, req.Sn, bindRow == nil, func() int64 {
				if bindRow != nil {
					return bindRow.UserID
				}
				return 0
			}())

		if bindRow == nil {
			return nil, errorx.NewCodeError(errorx.CodeDeviceNoPermission, "设备未绑定，请先绑定该设备")
		}

		return nil, errorx.NewCodeError(errorx.CodeDeviceNoPermission, "无权限操作该设备")
	}

	normalizedAlias := normalizeString(req.Alias, 50)
	normalizedLocation := normalizeString(req.Location, 100)
	normalizedGroupName := normalizeString(req.GroupName, 100)
	normalizedScene := normalizeString(req.Scene, 200)

	l.Logger.Infof("UpdateDevice: 准备更新设备信息, userId=%d, sn=%s, alias=%s, location=%s, group=%s, scene=%s",
		userId, req.Sn, normalizedAlias, normalizedLocation, normalizedGroupName, normalizedScene)

	affected, err := l.svcCtx.DeviceBind.UpdateDeviceAuxiliaryInfo(
		l.ctx, userId, req.Sn,
		normalizedAlias, normalizedLocation, normalizedGroupName, normalizedScene,
	)

	if err != nil {
		l.Logger.Errorf("UpdateDevice: 更新设备信息失败, userId=%d, sn=%s, error=%v", userId, req.Sn, err)

		errMsg := err.Error()
		if containsIgnoreCase(errMsg, "column") || containsIgnoreCase(errMsg, "does not exist") {
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "数据库表缺少必要字段，请执行数据库迁移")
		}
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "更新失败："+errMsg)
	}

	if affected == 0 {
		l.Logger.Errorf("UpdateDevice: 未更新任何记录, userId=%d, sn=%s", userId, req.Sn)
		return nil, errorx.NewCodeError(errorx.CodeInternalError, "更新失败：未找到匹配的绑定记录")
	}

	l.Logger.Infof("UpdateDevice: 设备信息更新成功, userId=%d, sn=%s, alias=%s, location=%s",
		userId, req.Sn, normalizedAlias, normalizedLocation)

	resp := &types.UpdateDeviceResp{
		Sn:        req.Sn,
		Alias:     normalizedAlias,
		Location:  normalizedLocation,
		GroupName: normalizedGroupName,
		Scene:     normalizedScene,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	return resp, nil
}

// validateRequestParams 校验请求参数合法性
func (l *UpdateDeviceLogic) validateRequestParams(req *types.UpdateDeviceReq) error {

	req.Sn = strings.TrimSpace(req.Sn)
	if req.Sn == "" {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}

	if len(req.Sn) > 64 {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号长度不能超过64字符")
	}

	hasUpdateField := false

	if req.Alias != "" {
		req.Alias = strings.TrimSpace(req.Alias)
		if len([]rune(req.Alias)) > 50 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "设备备注名长度不能超过50字符")
		}
		hasUpdateField = true
	}

	if req.Location != "" {
		req.Location = strings.TrimSpace(req.Location)
		if len([]rune(req.Location)) > 100 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "设备位置长度不能超过100字符")
		}
		hasUpdateField = true
	}

	if req.GroupName != "" {
		req.GroupName = strings.TrimSpace(req.GroupName)
		if len([]rune(req.GroupName)) > 100 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "设备分组长度不能超过100字符")
		}
		hasUpdateField = true
	}

	if req.Scene != "" {
		req.Scene = strings.TrimSpace(req.Scene)
		if len([]rune(req.Scene)) > 200 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "常用场景长度不能超过200字符")
		}
		hasUpdateField = true
	}

	if !hasUpdateField {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "至少需要提供一个要更新的字段")
	}

	return nil
}

// normalizeString 规范化字符串：去除首尾空白，截断至最大长度
func normalizeString(s string, maxLen int) string {

	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	runes := []rune(s)
	if len(runes) > maxLen {
		s = string(runes[:maxLen])
	}

	return s
}

// containsIgnoreCase 忽略大小写的字符串包含检查
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
