// Package validate 基于 github.com/go-playground/validator/v10，统一封装项目内校验能力，供 HTTP（httpx）、业务逻辑等调用。
//
// # 能力概览（与 struct 标签一致）
//
//   - 非空：required；标量封装见 RequireNonBlank。
//   - 长度/范围：len、min、max、gte、lte；标量封装见 CheckLen、CheckMinMax。
//   - 格式：内置 email、url、uuid；IPv4/IPv6 见 CheckIP、CheckIPv4；项目自定义 mobile、password、idcard。
//   - 可配置最小密码长度：CheckPasswordMin（与 config.Register.EffectiveMinPasswordLen 配合）。
//   - 枚举：oneof；标量封装见 CheckOneOf。
//   - 嵌套：子结构体字段照常写 validate 标签即可；切片/数组元素用 dive，例如 validate:"required,dive,email"。
//   - 自定义标签：RegisterValidation（须在进程内并发校验前注册完成）。
//
// # 用法
//
//   - HTTP 请求体：在 struct 上写 `validate:"..."`，main 中 httpx.SetValidator(validate.NewHTTPValidator())。
//   - 任意结构体：ValidateStruct(s) 得到与 HTTP 一致映射的 CodeError。
//   - 单值：CheckEmail、CheckMobile、CheckURL、VarTag 等。
//   - 发码/验证码联系方式：ValidateContactChannel、ValidateContactTarget（channel 为 email|mobile）。
package validate
