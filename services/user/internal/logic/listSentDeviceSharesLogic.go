package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListSentDeviceSharesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSentDeviceSharesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSentDeviceSharesLogic {
	return &ListSentDeviceSharesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSentDeviceSharesLogic) ListSentDeviceShares() (resp *types.DeviceShareListResp, err error) {
	userId := l.getUserIdFromCtx()
	if userId == 0 {
		l.Logger.Errorf("ListSentDeviceShares: 获取用户ID失败")
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "请重新登录")
	}

	rows, err := l.svcCtx.DB.QueryContext(l.ctx, `
		SELECT s.id, s.device_sn, s.target_account, s.status, s.created_at, s.end_at
		FROM public.user_device_share s
		WHERE s.owner_user_id = $1
		ORDER BY s.created_at DESC
	`, userId)
	if err != nil {
		l.Logger.Errorf("ListSentDeviceShares: 查询共享列表失败, userId=%d, err=%v", userId, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}
	defer rows.Close()

	var list []types.DeviceShareItem
	for rows.Next() {
		var item types.DeviceShareItem
		var createdAt time.Time
		var endAt sql.NullTime
		var st dao.DeviceShareStatus
		if err := rows.Scan(&item.ShareId, &item.Sn, &item.ShareTo, &st, &createdAt, &endAt); err != nil {
			l.Logger.Errorf("ListSentDeviceShares: 扫描行数据失败, err=%v", err)
			continue
		}
		item.Status = dao.DeviceShareStatusToLegacyInt(st)
		item.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		item.EndAt = formatNullTime(endAt)
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		l.Logger.Errorf("ListSentDeviceShares: 遍历结果失败, userId=%d, err=%v", userId, err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询失败")
	}

	return &types.DeviceShareListResp{List: list}, nil
}

func (l *ListSentDeviceSharesLogic) getUserIdFromCtx() int64 {
	v := l.ctx.Value("userId")
	if v == nil {
		return 0
	}
	switch id := v.(type) {
	case json.Number:
		n, err := id.Int64()
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(id)
	case int64:
		return id
	case int:
		return int64(id)
	case string:
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
