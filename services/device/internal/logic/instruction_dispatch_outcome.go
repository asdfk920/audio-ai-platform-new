package logic

// MapInstructionDispatchOutcome 将 commandsvc.CreateImmediateInstructionResult.Status 转成对客户端的文案。
// deliveredVerb 为成功时的主语，例如「播放」「暂停」「音量加」「播放列表」。
// cachedMessage 通常为离线场景的完整提示句；传空则用默认离线提示。
func MapInstructionDispatchOutcome(resultStatus, deliveredVerb, cachedMessage string) (status string, message string) {
	if cachedMessage == "" {
		cachedMessage = "设备离线，指令已缓存，设备上线后将自动执行"
	}
	switch resultStatus {
	case "dispatched":
		return "delivered", deliveredVerb + "指令已通过 WebSocket 下发至设备"
	case "queued":
		return "queued", "云端显示设备在线但未经 WebSocket 送达，" + deliveredVerb +
			"指令在待下发队列，请保持设备 WebSocket 长连接或由后台稍后自动投递"
	default:
		return "cached", cachedMessage
	}
}
