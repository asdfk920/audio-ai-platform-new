package config

import (
	"time"
)

// Config 网关配置结构
type Config struct {
	Name string `yaml:"Name"`
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
	
	Log       LogConfig       `yaml:"Log"`
	JWT       JWTConfig      `yaml:"JWT"`
	RateLimit RateLimitConfig `yaml:"RateLimit"`
	Routes    []RouteConfig  `yaml:"Routes"`
	CORS      CORSConfig     `yaml:"CORS"`
	Redis     RedisConfig    `yaml:"Redis"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"Level"`
	Format string `yaml:"Format"`
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	Secret    string   `yaml:"Secret"`
	ExpireHours int    `yaml:"ExpireHours"`
	SkipPaths []string `yaml:"SkipPaths"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	GlobalRPS     int `yaml:"GlobalRPS"`
	IPRPS         int `yaml:"IPRPS"`
	UserRPS       int `yaml:"UserRPS"`
	WindowSeconds int `yaml:"WindowSeconds"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	PathPrefix string        `yaml:"PathPrefix"`
	Target     string        `yaml:"Target"`
	Timeout    time.Duration `yaml:"Timeout"`
	Name       string        `yaml:"Name"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string `yaml:"AllowOrigins"`
	AllowMethods     []string `yaml:"AllowMethods"`
	AllowHeaders     []string `yaml:"AllowHeaders"`
	ExposeHeaders    []string `yaml:"ExposeHeaders"`
	MaxAge           int      `yaml:"MaxAge"`
	AllowCredentials bool     `yaml:"AllowCredentials"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"Addr"`
	Password string `yaml:"Password"`
	DB       int    `yaml:"DB"`
}