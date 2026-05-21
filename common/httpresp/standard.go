// Package httpresp 提供平台统一的 HTTP JSON 响应信封：HTTP 状态码与 body.code 一致，msg 与语义对齐。
package httpresp

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Standard 标准 JSON 结构（成功 / 失败通用）。
// code：与 HTTP 状态码相同（如 200、400）；msg：人类可读说明；data：业务数据，失败时常省略。
type Standard struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// 与《HTTP 接口响应》约定一致的核心文案（可按场景在 Write 时传入更具体的 msg）。
const (
	MsgOK           = "请求成功、接口正常连通"
	MsgBadRequest   = "请求参数错误"
	MsgUnauthorized = "未登录 / Token 无效 / 过期"
	MsgForbidden    = "登录成功但无权限"
	MsgConflict     = "资源冲突"
	MsgInternal     = "服务内部异常"
	// 未在表中列出但常用
	MsgNotFound          = "资源不存在"
	MsgMethodNotAllowed  = "HTTP 方法不允许"
	MsgRangeNotSatisfiable = "请求范围不满足"
	MsgServiceUnavailable = "服务暂不可用"
)

// DefaultMsg 返回给定 HTTP 状态默认中文描述；未知状态回退为英文 StatusText。
func DefaultMsg(status int) string {
	switch status {
	case http.StatusOK:
		return MsgOK
	case http.StatusBadRequest:
		return MsgBadRequest
	case http.StatusUnauthorized:
		return MsgUnauthorized
	case http.StatusForbidden:
		return MsgForbidden
	case http.StatusNotFound:
		return MsgNotFound
	case http.StatusConflict:
		return MsgConflict
	case http.StatusInternalServerError:
		return MsgInternal
	case http.StatusMethodNotAllowed:
		return MsgMethodNotAllowed
	case http.StatusRequestedRangeNotSatisfiable:
		return MsgRangeNotSatisfiable
	case http.StatusServiceUnavailable:
		return MsgServiceUnavailable
	default:
		return http.StatusText(status)
	}
}

// Write 写出 JSON：status 为 HTTP 状态码，body.code 与之相同。msg 为空时使用 DefaultMsg(status)。
func Write(w http.ResponseWriter, status int, msg string, data interface{}) {
	if msg == "" {
		msg = DefaultMsg(status)
	}
	httpx.WriteJson(w, status, Standard{Code: status, Msg: msg, Data: data})
}

// WriteSuccess 成功响应（200 + MsgOK）。
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	Write(w, http.StatusOK, MsgOK, data)
}

// WriteSuccessMsg 成功响应并自定义 msg（msg 为空则等价 WriteSuccess）。
func WriteSuccessMsg(w http.ResponseWriter, msg string, data interface{}) {
	if msg == "" {
		msg = MsgOK
	}
	Write(w, http.StatusOK, msg, data)
}

// WithDetail 在标准后缀上拼接具体原因（如「请求参数错误：缺少 file」）。
func WithDetail(standardMsg, detail string) string {
	if detail == "" {
		return standardMsg
	}
	return standardMsg + "：" + detail
}
