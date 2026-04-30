// Code scaffolded by goctl. Safe to edit.

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
	// CacheRedis 与 go-zero 示例一致，取第一项作为节点客户端
	CacheRedis []redis.RedisConf `json:",optional"`
	// Database 业务库（PostgreSQL），与迁移 045 public.content 等表一致
	Database struct {
		DataSource string `json:",optional"`
	}
	// Storage 对象存储：s3/minio 或 local（开发落盘）
	Storage struct {
		Driver       string `json:",default=local"` // local | s3
		Region       string `json:",optional"`
		Endpoint     string `json:",optional"` // MinIO / 自定义 S3 endpoint
		AccessKey    string `json:",optional"`
		SecretKey    string `json:",optional"`
		Bucket       string `json:",optional"`
		UsePathStyle bool   `json:",optional"`
		// CdnBaseUrl 对外访问前缀（不要尾斜杠），写入 DB 的 cover_url / audio_url
		CdnBaseUrl string `json:",optional"`
	}
	// Local 本地存储根目录（Driver=local 时使用）
	Local struct {
		Root string `json:",default=./data/content-objects"`
	}
	// Upload 校验上限（可配置）
	Upload struct {
		AudioMaxMB int64 `json:",default=100"`
		CoverMaxMB int64 `json:",default=10"`
	}
	// List 内容列表：热点缓存 TTL 在逻辑层限制为 300–600 秒
	List struct {
		CacheEnabled     bool  `json:",default=true"`
		CacheTTLSeconds  int   `json:",default=600"`
		HotPlayThreshold int64 `json:",default=1000"` // 预留（排序热度等）
	}
	// Recommend 首页推荐池：Redis JSON + 返回前按会员等级裁剪（池子 20～50 条）
	Recommend struct {
		CacheEnabled    bool `json:",default=true"`
		CacheTTLSeconds int  `json:",default=600"` // 5～15 分钟，逻辑层 clamp 300–900
		PoolSize        int  `json:",default=50"`
	}
	// ContentAuth 统一鉴权：会员内容降级试听秒数
	ContentAuth struct {
		PreviewSeconds int `json:",default=60"`
	}
	// Internal 运维/管理回调（非空才启用对应路由）
	Internal struct {
		// ListCacheBumpSecret 非空时允许通过 Header X-Internal-Secret 调用列表缓存失效
		ListCacheBumpSecret string `json:",optional"`
	}
}
