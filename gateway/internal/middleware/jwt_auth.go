package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/audio-ai-platform/gateway/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 声明结构
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth JWT 认证中间件
type JWTAuth struct {
	config config.JWTConfig
}

// NewJWTAuth 创建 JWT 认证中间件
func NewJWTAuth(cfg config.JWTConfig) *JWTAuth {
	return &JWTAuth{
		config: cfg,
	}
}

// Handler JWT 认证处理函数
func (j *JWTAuth) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		
		// 检查是否在白名单中
		if j.isSkipPath(path) {
			c.Next()
			return
		}
		
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证令牌",
			})
			c.Abort()
			return
		}
		
		// 解析 Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的认证格式，请使用 Bearer token",
			})
			c.Abort()
			return
		}
		
		// 验证 JWT token
		claims, err := j.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "认证令牌无效",
				"detail": err.Error(),
			})
			c.Abort()
			return
		}
		
		// 将用户信息设置到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		
		c.Next()
	}
}

// isSkipPath 检查路径是否在白名单中
func (j *JWTAuth) isSkipPath(path string) bool {
	for _, skipPath := range j.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// validateToken 验证 JWT token
func (j *JWTAuth) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.Secret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, jwt.ErrSignatureInvalid
}

// GenerateToken 生成 JWT token（用于测试）
func (j *JWTAuth) GenerateToken(userID int64, username, role string) (string, error) {
	expireTime := time.Now().Add(time.Duration(j.config.ExpireHours) * time.Hour)
	
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "audio-ai-platform-gateway",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.Secret))
}