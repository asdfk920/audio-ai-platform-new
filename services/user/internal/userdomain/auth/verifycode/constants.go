package verifycode

// 联系方式通道（与 API、repo、DAO 约定一致，避免魔法字符串散落）。
const (
	ChannelEmail  = "email"
	ChannelMobile = "mobile"
)

// 发码业务场景（与 types.SendVerifyCodeReq.Scene 一致）。
const (
	SceneRegister = "register"
	SceneReset    = "reset"
	SceneBind     = "bind"
	SceneLogin    = "login" // 验证码登录/静默注册发码：不校验是否已注册
)
