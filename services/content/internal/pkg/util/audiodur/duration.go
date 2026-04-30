package audiodur

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/hajimehoshi/go-mp3"
	"github.com/mewkiz/flac"
)

// Seconds 解析音频时长（秒）。mp3/wav/flac 本地解析；aac 等依赖 ffprobe（PATH 中需有 ffmpeg）。
func Seconds(ext string, r io.ReadSeeker) (int, error) {
	e := strings.ToLower(strings.TrimPrefix(ext, "."))
	switch e {
	case "wav":
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return wavSeconds(r)
	case "mp3":
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return mp3Seconds(r)
	case "flac":
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return flacSeconds(r)
	case "aac":
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		if sec, ok := tagTLENSafe(r); ok {
			return sec, nil
		}
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return ffprobeFromSeeker(r)
	default:
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		if sec, ok := tagTLENSafe(r); ok {
			return sec, nil
		}
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return ffprobeFromSeeker(r)
	}
}

func mp3Seconds(r io.Reader) (int, error) {
	d, err := mp3.NewDecoder(r)
	if err != nil {
		return 0, fmt.Errorf("mp3: %w", err)
	}
	pcmBytes := d.Length()
	sr := d.SampleRate()
	if sr <= 0 || pcmBytes <= 0 {
		return 0, fmt.Errorf("mp3: 无法计算时长")
	}
	ch := 2
	sec := int(pcmBytes / int64(2*ch) / int64(sr))
	if sec < 1 {
		return 1, nil
	}
	return sec, nil
}

func flacSeconds(r io.ReadSeeker) (int, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	stream, err := flac.New(r)
	if err != nil {
		return 0, fmt.Errorf("flac: %w", err)
	}
	si := stream.Info
	if si == nil || si.SampleRate == 0 {
		return 0, fmt.Errorf("flac: 缺少采样率")
	}
	if si.NSamples == 0 {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return ffprobeFromSeeker(r)
	}
	sec := int(si.NSamples / uint64(si.SampleRate))
	if sec < 1 {
		return 1, nil
	}
	return sec, nil
}

func tagTLENSafe(r io.ReadSeeker) (int, bool) {
	m, err := tag.ReadFrom(r)
	if err != nil {
		return 0, false
	}
	raw := m.Raw()
	for _, k := range []string{"TLEN", "length", "LENGTH", "duration"} {
		v, ok := raw[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case int:
			if t <= 0 {
				continue
			}
			if t > 1_000_000 {
				return t / 1_000_000, true
			}
			if t > 10_000 {
				return t / 1000, true
			}
			return t, true
		case int64:
			if t <= 0 {
				continue
			}
			if t > 1_000_000 {
				return int(t / 1_000_000), true
			}
			if t > 10_000 {
				return int(t / 1000), true
			}
			return int(t), true
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(t))
			if err != nil || n <= 0 {
				continue
			}
			if n > 1_000_000 {
				return n / 1_000_000, true
			}
			if n > 10_000 {
				return n / 1000, true
			}
			return n, true
		}
	}
	return 0, false
}

func ffprobeFromSeeker(r io.ReadSeeker) (int, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp("", "content-audio-probe-*")
	if err != nil {
		return 0, err
	}
	path := tmp.Name()
	if _, err := tmp.Write(buf); err != nil {
		tmp.Close()
		_ = os.Remove(path)
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return 0, err
	}
	defer os.Remove(path)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("无法解析时长（aac 等需安装 ffmpeg/ffprobe 并加入 PATH）: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("ffprobe 未返回时长")
	}
	secF, err := strconv.ParseFloat(s, 64)
	if err != nil || secF <= 0 {
		return 0, fmt.Errorf("ffprobe 时长无效: %q", s)
	}
	sec := int(secF + 0.5)
	if sec < 1 {
		return 1, nil
	}
	return sec, nil
}

func wavSeconds(r io.Reader) (int, error) {
	var hdr [12]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, err
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return 0, fmt.Errorf("非 WAV 文件")
	}
	var byteRate uint32
	var dataSize uint32
	for i := 0; i < 256; i++ {
		var chunkID [4]byte
		var chunkSize uint32
		if err := binary.Read(r, binary.LittleEndian, &chunkID); err != nil {
			return 0, err
		}
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return 0, err
		}
		id := string(chunkID[:])
		pad := int64(chunkSize)
		if chunkSize%2 == 1 {
			pad++
		}
		switch id {
		case "fmt ":
			buf := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, buf); err != nil {
				return 0, err
			}
			br := bytes.NewReader(buf)
			var audioFormat, numCh uint16
			var sampleRate uint32
			_ = binary.Read(br, binary.LittleEndian, &audioFormat)
			_ = binary.Read(br, binary.LittleEndian, &numCh)
			_ = binary.Read(br, binary.LittleEndian, &sampleRate)
			_ = binary.Read(br, binary.LittleEndian, &byteRate)
		case "data":
			dataSize = chunkSize
			if _, err := io.CopyN(io.Discard, r, pad); err != nil {
				return 0, err
			}
		default:
			if _, err := io.CopyN(io.Discard, r, pad); err != nil {
				return 0, err
			}
		}
		if dataSize > 0 && byteRate > 0 {
			sec := int(dataSize / byteRate)
			if sec < 1 {
				return 1, nil
			}
			return sec, nil
		}
	}
	return 0, fmt.Errorf("WAV 缺少 fmt/data")
}
