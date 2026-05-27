// Code scaffolded by goctl. Safe to edit.

package config

import (
	apicors "github.com/jacklau/audio-ai-platform/common/cors"
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
	// Storage 对象存储：local | s3 | oss（阿里云）
	Storage struct {
		Driver       string `json:",default=local"` // local | s3 | oss
		Region       string `json:",optional"`
		Endpoint     string `json:",optional"` // MinIO / 自定义 S3 endpoint
		AccessKey    string `json:",optional"`
		SecretKey    string `json:",optional"`
		Bucket       string `json:",optional"`
		UsePathStyle bool   `json:",optional"`
		// CdnBaseUrl 对外访问前缀（不要尾斜杠），写入 DB 的 cover_url / audio_url
		CdnBaseUrl string `json:",optional"`
		// SyncFromSysConfig 为 true 时，启动先从 Database.DataSource 对应库的 public.sys_config
		// 读取 upload_storage_*（与 go-admin 上传配置一致）；优先级：环境变量 > sys_config > YAML。
		SyncFromSysConfig bool `json:",optional"`
	}
	// Local 本地目录：Driver=local 时为主存储；OSS/S3 时可为私有格式上传的本地镜像备份根目录。
	Local struct {
		Root string `json:",default=./data/content-objects"`
	}
	// Upload 校验上限（可配置）
	Upload struct {
		AudioMaxMB                 int64 `json:",default=100"`
		CoverMaxMB                 int64 `json:",default=10"`
		PrivateFormatMaxMB         int64 `json:",default=128"` // 私有格式整包；需与 RestConf.MaxBytes 协调
		PrivateFormatLocalMirror   bool  `json:",default=true"` // 主存储为 OSS/S3 时是否在 Local.Root 下再写一份副本
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
	// Spotify OAuth 配置
	Spotify struct {
		ClientID        string `json:",optional"`
		ClientSecret    string `json:",optional"`
		CallbackBaseURL string `json:",optional"`
		MockMode        bool   `json:",default=false"` // 开发测试模式，使用模拟数据
	}
	// QQ音乐 OAuth 配置
	QQMusic struct {
		AppID           string `json:",optional"`
		AppSecret       string `json:",optional"`
		CallbackBaseURL string `json:",optional"`
		MockMode        bool   `json:",default=false"` // 开发测试模式，使用模拟数据
	}
	// CORS 浏览器跨域；省略 AllowOrigins 时等价允许 *。
	CORS apicors.Config `json:",optional"`
	// Device 设备微服务 HTTP：内容侧转发 POST /api/device/cmd/download_song。须与承载 /ws/device 的实例同一进程或共用 WS Redis relay，否则会误报「已下发」远端收不到。
	Device struct {
		BaseURL string `json:",optional"` // 不含尾斜杠，如 http://14.103.202.69:8002；空默认 http://127.0.0.1:8002
	} `json:",optional"`
}
