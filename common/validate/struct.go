package validate

// Struct 按 validate 标签校验结构体（含嵌套字段；切片元素需父字段使用 dive 等标签）。
// 返回原始 validator 错误，需自行处理或改用 ValidateStruct。
func Struct(s any) error {
	if s == nil {
		return nil
	}
	return SharedEngine().Struct(s)
}

// ValidateStruct 校验结构体并将首个字段错误映射为 errorx.CodeError（与 httpx 响应风格一致）。
func ValidateStruct(s any) error {
	if s == nil {
		return nil
	}
	if err := SharedEngine().Struct(s); err != nil {
		return MapToCodeError(err)
	}
	return nil
}
