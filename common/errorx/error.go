package errorx

import "fmt"

// 错误码定义
const (
	// 成功
	CodeSuccess = 0

	// 用户相关错误 1xxx
	CodeUserNotFound                  = 1001
	CodePasswordError                 = 1002
	CodeUserExists                    = 1003
	CodeTokenInvalid                  = 1004
	CodeTokenExpired                  = 1005
	CodeVerifyCodeLimit               = 1006 // 验证码发送过于频繁，请3分钟后重试
	CodeVerifyCodeInvalid             = 1007 // 验证码错误或已过期
	CodeInvalidParam                  = 1008 // 参数无效（需邮箱或手机二选一）
	CodeInvalidEmail                  = 1009 // 邮箱格式不正确
	CodeInvalidMobile                 = 1010 // 手机号格式不正确
	CodeRegisterRateLimit             = 1011 // 注册过于频繁（如按 IP 限流）
	CodeRegisterBusy                  = 1012 // 同一账号注册处理中（分布式锁）
	CodeLoginBusy                     = 1013 // 同一账号登录签发令牌处理中（分布式锁）
	CodeIPDenied                      = 1014 // IP 不在白名单或命中黑名单
	CodeAccountLocked                 = 1015 // 登录失败过多或风控锁定
	CodeResetPasswordRateLimit        = 1016 // 重置密码接口请求过于频繁（如按 IP 限流）
	CodePasswordReuse                 = 1017 // 新密码不能与旧密码相同
	CodeWeakPassword                  = 1018 // 密码过于简单或命中弱口令规则
	CodeInvalidNickname               = 1019 // 昵称不合法
	CodeRegisterDuplicateSubmit       = 1020 // 注册请求重复提交过快（防重放）
	CodeUpdateProfileRateLimit        = 1021 // 资料修改过于频繁（按用户限流）
	CodeInvalidProfileAvatar          = 1022 // 头像链接不合法
	CodeRefreshTokenSuperseded        = 1023 // refresh 非当前会话（已在其它端登录或已轮换）
	CodeRefreshRateLimit              = 1024 // 刷新 Token 接口请求过于频繁
	CodeUserAccountDisabled           = 1025 // 账号已禁用或状态异常，无法刷新
	CodeRebindContactRateLimit        = 1026 // 换绑接口请求过于频繁（按用户或 IP 限流）
	CodeRebindContactCooldown         = 1027 // 换绑冷却中，请稍后再试
	CodeRebindContactConflict         = 1028 // 换绑条件不满足（并发占用或绑定已变更等，不暴露细节）
	CodeRealNameAlreadyVerified       = 1029 // 已完成实名，无需重复提交
	CodeRealNameInProgress            = 1030 // 实名审核进行中，请勿重复提交
	CodeRealNameSubmitBusy            = 1031 // 提交过于频繁（互斥锁）
	CodeRealNameInvalidPayload        = 1032 // 实名参数不合法
	CodeRealNameProviderError         = 1033 // 核验通道异常，请稍后重试
	CodeRealNameAdminUnauthorized     = 1034 // 管理端鉴权失败
	CodeRealNameRecordNotFound        = 1035 // 实名记录不存在
	CodeRealNameInvalidReviewState    = 1036 // 当前记录不可审核
	CodeAccountCancelled              = 1037 // 账号已注销（逻辑销户）
	CodeCancellationCooling           = 1038 // 注销冷静期内，禁止登录或继续使用
	CodeCancellationAlreadyPending    = 1039 // 已有进行中的注销申请（冷静期中）
	CodeCancellationNotInCooling      = 1040 // 当前不在注销冷静期，无法撤销
	CodeRealNameRebindLocked          = 1041 // 实名换绑身份核验失败过多，已临时锁定
	CodeCancellationWithdrawRateLimit = 1042 // 撤销注销申请过于频繁
	CodeNicknameTaken                 = 1043 // 昵称已被占用
	CodeFamilyNotFound                = 1044 // 家庭不存在
	CodeFamilyAlreadyExists           = 1045 // 当前用户已拥有家庭
	CodeFamilyInviteInvalid           = 1046 // 家庭邀请码无效
	CodeFamilyMemberExists            = 1047 // 用户已是家庭成员
	CodeFamilyNoPermission            = 1048 // 无权限操作家庭
	CodeDeviceShareInvalid            = 1049 // 设备共享邀请无效
	CodeDeviceShareExists             = 1050 // 设备共享关系已存在
	CodeDeviceShareExpired            = 1051 // 设备共享已过期
	CodeDeviceShareNoPermission       = 1052 // 无权限操作设备共享
	CodeDeviceShareForbiddenReshare   = 1053 // 当前角色不允许继续分享

	// 设备相关错误 2xxx
	CodeDeviceNotFound = 2001
	CodeDeviceExists   = 2002
	CodeDeviceOffline  = 2003
	// 设备首次注册 / Provisioning
	CodeDeviceSnInvalid            = 2004 // SN 格式或规则不通过
	CodeDeviceProductInvalid       = 2005 // ProductKey 未在平台登记
	CodeDeviceSnBlacklisted        = 2006 // SN 命中黑名单
	CodeDeviceAlreadyRegistered    = 2007 // SN 已注册（重复注册）
	CodeDeviceBoundByOther         = 2008 // 设备已被其他用户绑定
	CodeDeviceUnbindForbidden      = 2009 // 无权限解绑或设备未绑定（兼容）
	CodeDeviceNotBound             = 2012 // 无活跃绑定，无法解绑
	CodeDeviceUnbindNotOwner       = 2013 // 非绑定者解绑
	CodeDeviceRegisterRateLimit    = 2014 // 设备注册过于频繁（IP 限流）
	CodeDeviceDisabled             = 2015 // 设备已禁用，无法查询状态
	CodeDeviceInactive             = 2016 // 设备未激活，无法查询状态
	CodeDeviceScrapped             = 2017 // 设备已报废，无法查询状态
	CodeDeviceStatusQueryRateLimit = 2018 // 设备状态查询过于频繁
	CodeMemberPackageNotFound      = 2020 // 套餐不存在
	CodeMemberPackageDisabled      = 2021 // 套餐已下架
	CodeMemberOrderNotFound        = 2022 // 订单不存在
	CodeMemberOrderNotPending      = 2023 // 订单非待支付
	CodeMemberInsufficientBalance  = 2024 // 余额不足
	CodeMemberPayCallbackInvalid   = 2025 // 支付回调验签失败或参数非法
	CodeDeviceCommandNotFound      = 2026 // 指令不存在
	CodeDeviceNoPermission         = 2027 // 无权限操作设备
	CodeDeviceSecretInvalid        = 2028 // 设备密钥不正确
	CodeDeviceAuthLocked           = 2029 // 设备认证失败次数过多，已锁定
	CodeDeviceSignatureInvalid     = 2030 // 设备签名校验失败
	CodeDeviceTimestampInvalid     = 2031 // 设备时间戳不合法或超出允许窗口
	CodeMemberNoSubscription       = 2032 // 无会员档案或不可退订
	CodeMemberPermanentNoUnsubscribe = 2033 // 永久会员请通过客服处理退订
	CodeMemberUnsubscribeNotPending = 2034 // 当前未处于待到期退订状态
	CodeMemberExpiredNoUnsubscribe  = 2035 // 会员已过期，无需退订

	// 内容相关错误 3xxx
	CodeContentNotFound = 3001
	CodeUploadFailed    = 3002
	CodeProcessFailed   = 3003

	// 推流相关错误 4xxx（media-processing 服务）
	CodeInvalidSourceType = 40002 // 无效的推流来源类型
	CodeAlreadyPushing    = 40003 // 已在推流中
	CodeResourceNotFound  = 40401 // 资源不存在
	CodeIPBanned          = 40303 // IP 被封禁
	CodeRateLimited       = 42901 // 请求频率超限
	CodeVipNotEnough      = 40302 // VIP 等级不足

	// 系统错误 9xxx
	CodeDatabaseError = 9001
	CodeRedisError    = 9002
	CodeParamError    = 9003
	CodeSystemError   = 9004
	CodeDBError       = 9005
	CodeNotFound      = 404
	CodeNoPermission  = 403
)

// 错误消息映射
var codeMsg = map[int]string{
	CodeSuccess:                       "success",
	CodeUserNotFound:                  "用户不存在",
	CodePasswordError:                 "密码错误",
	CodeUserExists:                    "用户已存在",
	CodeTokenInvalid:                  "Token 无效",
	CodeTokenExpired:                  "Token 已过期",
	CodeVerifyCodeLimit:               "验证码发送过于频繁，请3分钟后再试",
	CodeVerifyCodeInvalid:             "验证码错误或已过期",
	CodeInvalidParam:                  "请使用邮箱或手机号其中一种方式",
	CodeInvalidEmail:                  "邮箱格式不正确",
	CodeInvalidMobile:                 "手机号格式不正确",
	CodeRegisterRateLimit:             "注册过于频繁，请稍后再试",
	CodeRegisterBusy:                  "注册处理中，请稍后再试",
	CodeLoginBusy:                     "登录处理中，请稍后再试",
	CodeIPDenied:                      "当前网络环境不允许登录",
	CodeAccountLocked:                 "账号已锁定，请稍后再试",
	CodeResetPasswordRateLimit:        "重置密码过于频繁，请稍后再试",
	CodePasswordReuse:                 "新密码不能与旧密码相同",
	CodeWeakPassword:                  "密码强度不足，请避免弱口令或连续字符",
	CodeInvalidNickname:               "昵称不合法或包含不允许的字符",
	CodeNicknameTaken:                 "该昵称已被占用",
	CodeFamilyNotFound:                "家庭不存在",
	CodeFamilyAlreadyExists:           "当前账号已创建家庭",
	CodeFamilyInviteInvalid:           "家庭邀请码无效或已失效",
	CodeFamilyMemberExists:            "该用户已是家庭成员",
	CodeFamilyNoPermission:            "无权限操作当前家庭",
	CodeDeviceShareInvalid:            "设备共享邀请码无效或已失效",
	CodeDeviceShareExists:             "设备共享关系已存在",
	CodeDeviceShareExpired:            "设备共享已过期",
	CodeDeviceShareNoPermission:       "无权限操作该设备共享",
	CodeDeviceShareForbiddenReshare:   "当前角色不允许继续分享设备",
	CodeRegisterDuplicateSubmit:       "操作过于频繁，请稍后再试",
	CodeUpdateProfileRateLimit:        "资料修改过于频繁，请稍后再试",
	CodeInvalidProfileAvatar:          "头像链接不合法",
	CodeRefreshTokenSuperseded:        "登录状态已更新，请重新登录",
	CodeRefreshRateLimit:              "刷新过于频繁，请稍后再试",
	CodeUserAccountDisabled:           "账号不可用，请重新登录或联系客服",
	CodeRebindContactRateLimit:        "换绑请求过于频繁，请稍后再试",
	CodeRebindContactCooldown:         "换绑过于频繁，请稍后再试",
	CodeRebindContactConflict:         "暂时无法完成换绑，请稍后重试",
	CodeRealNameAlreadyVerified:       "您已完成实名认证",
	CodeRealNameInProgress:            "实名认证处理中，请耐心等待",
	CodeRealNameSubmitBusy:            "操作过于频繁，请稍后再试",
	CodeRealNameInvalidPayload:        "提交的实名信息不合法，请检查后重试",
	CodeRealNameProviderError:         "系统繁忙，请稍后重试",
	CodeRealNameAdminUnauthorized:     "无权限执行该操作",
	CodeRealNameRecordNotFound:        "实名记录不存在",
	CodeRealNameInvalidReviewState:    "当前状态不可审核",
	CodeAccountCancelled:              "账号已注销",
	CodeCancellationCooling:           "账号注销冷静期内，暂无法使用该账号",
	CodeCancellationAlreadyPending:    "已提交注销申请，请等待冷静期结束或先撤销申请",
	CodeCancellationNotInCooling:      "当前没有可撤销的注销申请",
	CodeRealNameRebindLocked:          "身份核验尝试次数过多，请稍后再试",
	CodeCancellationWithdrawRateLimit: "撤销注销申请过于频繁，请稍后再试",
	CodeDeviceNotFound:                "设备不存在",
	CodeDeviceExists:                  "设备已绑定",
	CodeDeviceOffline:                 "设备离线",
	CodeDeviceSnInvalid:               "设备序列号不合法",
	CodeDeviceProductInvalid:          "产品型号未注册或无效",
	CodeDeviceSnBlacklisted:           "设备序列号不允许注册",
	CodeDeviceAlreadyRegistered:       "设备已注册，请勿重复注册",
	CodeDeviceBoundByOther:            "该设备已被其他用户绑定",
	CodeDeviceUnbindForbidden:         "无权限解绑该设备",
	CodeDeviceNotBound:                "该设备未绑定",
	CodeDeviceUnbindNotOwner:          "无权限解绑他人设备",
	CodeDeviceRegisterRateLimit:       "设备注册过于频繁，请稍后再试",
	CodeDeviceDisabled:                "设备已禁用，暂无法查询状态",
	CodeDeviceInactive:                "设备未激活，暂无法查询状态",
	CodeDeviceScrapped:                "设备已报废，暂无法查询状态",
	CodeDeviceStatusQueryRateLimit:    "设备状态查询过于频繁，请稍后再试",
	CodeMemberPackageNotFound:         "套餐不存在",
	CodeMemberPackageDisabled:         "套餐已下架",
	CodeMemberOrderNotFound:           "订单不存在",
	CodeMemberOrderNotPending:         "订单状态不可支付",
	CodeMemberInsufficientBalance:     "账户余额不足",
	CodeMemberPayCallbackInvalid:      "支付回调校验失败",
	CodeDeviceCommandNotFound:         "指令不存在",
	CodeDeviceNoPermission:            "无权限操作该设备",
	CodeDeviceSecretInvalid:           "设备密钥错误",
	CodeDeviceAuthLocked:              "设备认证失败次数过多，已锁定",
	CodeDeviceSignatureInvalid:        "设备签名校验失败",
	CodeDeviceTimestampInvalid:        "设备时间戳不合法或已过期",
	CodeMemberNoSubscription:          "暂无有效会员订阅",
	CodeMemberPermanentNoUnsubscribe:    "永久会员暂不支持自助退订，请联系客服",
	CodeMemberUnsubscribeNotPending:     "当前无需撤销或状态已变更",
	CodeMemberExpiredNoUnsubscribe:      "会员已过期",
	CodeContentNotFound:               "内容不存在",
	CodeUploadFailed:                  "上传失败",
	CodeProcessFailed:                 "处理失败",
	CodeInvalidSourceType:             "无效的推流来源类型",
	CodeAlreadyPushing:                "该资源已在推流中",
	CodeResourceNotFound:              "资源不存在",
	CodeIPBanned:                      "当前 IP 已被封禁",
	CodeRateLimited:                   "请求过于频繁，请稍后再试",
	CodeVipNotEnough:                  "VIP 等级不足",
	CodeDatabaseError:                 "数据库错误",
	CodeRedisError:                    "Redis 错误",
	CodeParamError:                    "参数错误",
	CodeSystemError:                   "系统错误",
	CodeDBError:                       "数据库操作失败",
	CodeNotFound:                      "资源不存在",
	CodeNoPermission:                  "无权限访问",
}

type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func NewCodeError(code int, msg string) error {
	if msg == "" {
		msg = codeMsg[code]
	}
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

func NewDefaultError(code int) error {
	return &CodeError{
		Code: code,
		Msg:  codeMsg[code],
	}
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

func (e *CodeError) GetCode() int {
	return e.Code
}

func (e *CodeError) GetMsg() string {
	return e.Msg
}

// CodeOf 从错误中提取错误码
func CodeOf(err error) int {
	if err == nil {
		return 0
	}
	// 尝试从 CodeError 中获取错误码
	if ce, ok := err.(*CodeError); ok {
		return ce.GetCode()
	}
	// 其他错误返回系统错误码
	return CodeSystemError
}

// IsCode 判断错误是否为指定错误码
func IsCode(err error, code int) bool {
	return CodeOf(err) == code
}
