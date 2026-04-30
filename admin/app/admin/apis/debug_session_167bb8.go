package apis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// #region agent log
func debugSessionLog167bb8(hypothesisId, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "167bb8",
		"runId":        "pre-fix",
		"hypothesisId": hypothesisId,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	b = append(b, '\n')
	wd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(wd, "debug-167bb8.log"),
		filepath.Join(wd, "..", "debug-167bb8.log"),
		filepath.Join(wd, "..", "..", "debug-167bb8.log"),
		filepath.Join(wd, "..", "..", "..", "debug-167bb8.log"),
	}
	for _, p := range candidates {
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = f.Write(b)
			_ = f.Close()
			return
		}
	}
}

// #endregion
