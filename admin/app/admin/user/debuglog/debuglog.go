package debuglog

import (
	"os"
	"path/filepath"
)

// AppendSession0a21fa appends one NDJSON line to workspace debug-0a21fa.log (tries cwd parents).
func AppendSession0a21fa(line []byte) {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	candidates := []string{
		filepath.Join(wd, "debug-0a21fa.log"),
		filepath.Join(wd, "..", "debug-0a21fa.log"),
		filepath.Join(wd, "..", "..", "debug-0a21fa.log"),
		filepath.Join(wd, "..", "..", "..", "debug-0a21fa.log"),
	}
	for _, p := range candidates {
		p = filepath.Clean(p)
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		_, _ = f.Write(append(line, '\n'))
		_ = f.Close()
		return
	}
}
