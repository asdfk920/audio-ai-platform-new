package shadowv2

type ShadowUpdateCallback func(deviceSN string, data interface{})

type ShadowNotifier struct {
	callback ShadowUpdateCallback
}

func NewShadowNotifier(callback ShadowUpdateCallback) *ShadowNotifier {
	return &ShadowNotifier{
		callback: callback,
	}
}

func (n *ShadowNotifier) Notify(deviceSN string, data interface{}) {
	if n.callback != nil && deviceSN != "" && data != nil {
		n.callback(deviceSN, data)
	}
}
