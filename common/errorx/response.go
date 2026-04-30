package errorx

// Response 统一 HTTP JSON 信封（成功 / 失败均可使用）。
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Success 成功响应，data 为业务载荷（如 { "expire_seconds": 180 }）。
func Success(data interface{}) *Response {
	return &Response{
		Code: CodeSuccess,
		Msg:  "success",
		Data: data,
	}
}

// SuccessData 成功响应，自定义 msg（如「绑定成功」），并携带 data。
func SuccessData(msg string, data interface{}) *Response {
	if msg == "" {
		msg = codeMsg[CodeSuccess]
	}
	if msg == "" {
		msg = "success"
	}
	return &Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	}
}

// SuccessMsg 成功响应，仅返回提示消息（不含 data 字段）。
func SuccessMsg(msg string) *Response {
	if msg == "" {
		msg = codeMsg[CodeSuccess]
	}
	if msg == "" {
		msg = "success"
	}
	return &Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: nil,
	}
}

// Error 失败响应，无 data 字段（omitempty）。
func Error(code int, msg string) *Response {
	if msg == "" {
		msg = codeMsg[code]
	}
	if msg == "" {
		msg = "未知错误"
	}
	return &Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
}

// ErrorWithData 失败响应且携带 data（如校验明细）。
func ErrorWithData(code int, msg string, data interface{}) *Response {
	if msg == "" {
		msg = codeMsg[code]
	}
	if msg == "" {
		msg = "未知错误"
	}
	return &Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
