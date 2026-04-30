package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/captcha"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	jwtv5 "github.com/golang-jwt/jwt/v5"

	"go-admin/app/admin/models"
	"go-admin/app/admin/service"
)

// AdminLoginHandler /api/admin/login：sys_admin 表校验 + JWT（claims 含 type/admin_id/role，见 PayloadFunc）
func AdminLoginHandler(mw *jwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := api.GetRequestLogger(c)
		db, err := pkg.GetOrm(c)
		if err != nil {
			log.Errorf("get db error, %s", err.Error())
			mw.Unauthorized(c, http.StatusInternalServerError, "数据库连接获取失败")
			return
		}
		var status = "2"
		var msg = "登录成功"
		var username = ""
		defer func() {
			LoginLogToDB(c, status, msg, username)
		}()

		var loginVals Login
		if err = c.ShouldBind(&loginVals); err != nil {
			username = loginVals.Username
			msg = "数据解析失败"
			status = "1"
			mw.Unauthorized(c, 400, mw.HTTPStatusMessageFunc(jwt.ErrMissingLoginValues, c))
			return
		}
		if config.ApplicationConfig.Mode != "dev" {
			if !captcha.Verify(loginVals.UUID, loginVals.Code, true) {
				username = loginVals.Username
				msg = "验证码错误"
				status = "1"
				mw.Unauthorized(c, 400, mw.HTTPStatusMessageFunc(jwt.ErrInvalidVerificationode, c))
				return
			}
		}

		sysUser, role, e := AuthenticateSysAdmin(db, loginVals.Username, loginVals.Password)
		if e != nil {
			msg = "登录失败"
			status = "1"
			log.Warnf("%s admin login failed: %v", loginVals.Username, e)
			mw.Unauthorized(c, 400, mw.HTTPStatusMessageFunc(jwt.ErrFailedAuthentication, c))
			return
		}
		username = loginVals.Username

		// 迁移 082：在认证通过后再额外校验 IP 白名单 / 登录时间窗。
		// must_change_password 不在此处拒绝，由响应透传给前端触发强制改密流程。
		var secAdm models.SysAdmin
		if err := db.Where("id = ?", sysUser.UserId).First(&secAdm).Error; err == nil {
			svc := service.SysAdmin{}
			if err := svc.EnforceLoginRestrictions(&secAdm, c.ClientIP(), time.Now()); err != nil {
				msg = err.Error()
				status = "1"
				switch {
				case errors.Is(err, service.ErrSysAdminIPRejected):
					mw.Unauthorized(c, http.StatusForbidden, "当前 IP 不在允许登录范围")
				case errors.Is(err, service.ErrSysAdminTimeRejected):
					mw.Unauthorized(c, http.StatusForbidden, "当前时间不在允许登录时段")
				default:
					mw.Unauthorized(c, http.StatusForbidden, err.Error())
				}
				return
			}
		}

		// 透出「需强制改密」给前端（通过响应头 X-Must-Change-Password: 1），不阻断登录。
		if secAdm.Id > 0 && secAdm.MustChangePassword {
			c.Header("X-Must-Change-Password", "1")
			c.Header("Access-Control-Expose-Headers", "X-Must-Change-Password")
		}

		data := map[string]interface{}{
			"user":       sysUser,
			"role":       role,
			"_tokenKind": "admin",
		}
		token := jwtv5.New(jwtv5.GetSigningMethod(mw.SigningAlgorithm))
		claims := token.Claims.(jwtv5.MapClaims)
		if mw.PayloadFunc != nil {
			for key, value := range mw.PayloadFunc(data) {
				claims[key] = value
			}
		}
		expire := mw.TimeFunc().Add(mw.Timeout)
		claims["exp"] = expire.Unix()
		claims["orig_iat"] = mw.TimeFunc().Unix()
		tokenString, err := mwSignedString(mw, token)
		if err != nil {
			mw.Unauthorized(c, http.StatusOK, mw.HTTPStatusMessageFunc(jwt.ErrFailedTokenCreation, c))
			return
		}
		mw.AntdLoginResponse(c, http.StatusOK, tokenString, expire)
	}
}

// mwSignedString 与 GinJWTMiddleware 内部签名一致（包内未导出 signedString）
func mwSignedString(mw *jwt.GinJWTMiddleware, token *jwtv5.Token) (string, error) {
	return token.SignedString(mw.Key)
}
