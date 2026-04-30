package oauthstart

import (
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
)

// WeChatCallbackURL 微信 OAuth2 回调地址（须与开放平台配置一致）。
func WeChatCallbackURL(c *config.Config) string {
	if c == nil {
		return ""
	}
	if u := strings.TrimSpace(c.OAuth.WeChat.RedirectURL); u != "" {
		return u
	}
	host := strings.TrimSpace(c.Host)
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d/api/v1/user/oauth/wechat/callback", host, c.Port)
}

// GoogleCallbackURL Google OAuth 回调地址。
func GoogleCallbackURL(c *config.Config) string {
	if c == nil {
		return ""
	}
	if u := strings.TrimSpace(c.OAuth.Google.RedirectURL); u != "" {
		return u
	}
	host := strings.TrimSpace(c.Host)
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d/api/v1/user/oauth/google/callback", host, c.Port)
}
