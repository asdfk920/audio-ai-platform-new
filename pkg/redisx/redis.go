package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

type Config struct {
	Addr     string
	Password string
	DB       int
}

// Init 初始化 Redis 客户端
func Init(cfg Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	return err
}

// Set 设置键值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func Get(ctx context.Context, key string) (string, error) {
	return Client.Get(ctx, key).Result()
}

// Del 删除键
func Del(ctx context.Context, keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return Client.Exists(ctx, keys...).Result()
}

// Incr 自增并返回新值
func Incr(ctx context.Context, key string) (int64, error) {
	return Client.Incr(ctx, key).Result()
}

// ExistsIncrPipeline 在一次网络往返内执行 EXISTS(blockKey) 与 INCR(cntKey)，用于发码频控等场景。
func ExistsIncrPipeline(ctx context.Context, blockKey, cntKey string) (blocked int64, count int64, err error) {
	if Client == nil {
		return 0, 0, redis.ErrClosed
	}
	pipe := Client.Pipeline()
	ex := pipe.Exists(ctx, blockKey)
	inc := pipe.Incr(ctx, cntKey)
	if _, err = pipe.Exec(ctx); err != nil {
		return 0, 0, err
	}
	blocked, err = ex.Result()
	if err != nil {
		return 0, 0, err
	}
	count, err = inc.Result()
	return blocked, count, err
}

// TTL 获取键的剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return Client.TTL(ctx, key).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

// HSet 设置哈希字段
func HSet(ctx context.Context, key string, values ...interface{}) error {
	return Client.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段
func HGet(ctx context.Context, key, field string) (string, error) {
	return Client.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return Client.HGetAll(ctx, key).Result()
}

// SetNX 不存在时设置键（带过期），返回是否设置成功。
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return Client.SetNX(ctx, key, value, expiration).Result()
}

// Eval 执行 Lua 脚本（KEYS / ARGV 与 Redis 约定一致）。
func Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return Client.Eval(ctx, script, keys, args...).Result()
}

// ZAdd 有序集合添加成员
func ZAdd(ctx context.Context, key string, score float64, member string) error {
	return Client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRem 有序集合删除成员
func ZRem(ctx context.Context, key string, members ...interface{}) error {
	return Client.ZRem(ctx, key, members...).Err()
}

// ZRange 按 score 升序取成员
func ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return Client.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 按 score 降序取成员
func ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return Client.ZRevRange(ctx, key, start, stop).Result()
}

// Close 关闭连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
