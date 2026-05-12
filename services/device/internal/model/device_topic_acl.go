package model

import "time"

const (
	TopicActionSubscribe int16 = 0 // 订阅操作 (subscribe)
	TopicActionPublish   int16 = 1 // 发布操作 (publish)
	TopicActionBoth      int16 = 2 // 两者都允许 (both)

	PermissionDeny  int16 = 0 // 拒绝（黑名单）
	PermissionAllow int16 = 1 // 允许（白名单）
)

type DeviceTopicACL struct {
	ID           int64     `db:"id"`
	Sn           string    `db:"sn"`
	TopicPattern string    `db:"topic_pattern"`
	Action       int16     `db:"action"`
	Permission   int16     `db:"permission"`
	Description  string    `db:"description"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (a *DeviceTopicACL) IsAllow() bool {
	return a.Permission == PermissionAllow
}

func (a *DeviceTopicACL) ActionName() string {
	switch a.Action {
	case TopicActionSubscribe:
		return "subscribe"
	case TopicActionPublish:
		return "publish"
	default:
		return "both"
	}
}