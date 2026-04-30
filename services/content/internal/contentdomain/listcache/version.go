package listcache

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RedisKeyContentListVer 列表缓存版本号，变更目录数据时 INCR 以整批失效缓存。
const RedisKeyContentListVer = "content:list:ver"

// BumpContentListCacheVersion 新增/编辑/上下架/删除 / 改权重 / 播放量批量刷新后调用，INCR 版本号使
// content:list:*:ver:*、content:recommend:list:v*、content:recommend:cate:*:v* 等热点 Key 全部失效（读路径使用新 ver）。
func BumpContentListCacheVersion(ctx context.Context, r *redis.Redis) {
	if r == nil {
		return
	}
	_, _ = r.IncrCtx(ctx, RedisKeyContentListVer)
}
