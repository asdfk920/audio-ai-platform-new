package logger

import (
	"encoding/json"
	"os"
	"reflect"
	"time"
)

const debugLogPath = "C:\\Users\\Lenovo\\Desktop\\audio-ai-platform\\debug-bd9af0.log"

// AgentNDJSON appends one NDJSON line for the current debug session.
func AgentNDJSON(hypothesisID, location, message string, data map[string]any) {
	if hypothesisID == "" {
		hypothesisID = "?"
	}
	payload := map[string]any{
		"sessionId":    "bd9af0",
		"runId":        "pre-fix",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}

// ErrType returns a short type description without message body (no PII).
func ErrType(err error) string {
	if err == nil {
		return ""
	}
	return reflect.TypeOf(err).String()
}
