package validate

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var registerMu sync.Mutex

// RegisterValidation 向全局 SharedEngine 注册自定义校验标签（Func 与 go-playground 一致）。
// 须在服务启动阶段、并发 Struct/Var 之前完成注册。
func RegisterValidation(tag string, fn validator.Func) error {
	registerMu.Lock()
	defer registerMu.Unlock()
	return SharedEngine().RegisterValidation(tag, fn)
}

// RegisterValidationCtx 注册带 context 的自定义校验（validator.FuncCtx）。
func RegisterValidationCtx(tag string, fn validator.FuncCtx) error {
	registerMu.Lock()
	defer registerMu.Unlock()
	return SharedEngine().RegisterValidationCtx(tag, fn)
}
