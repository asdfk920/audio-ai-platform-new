package logic

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/zeromicro/go-zero/core/logx"
)

// WAVFmt WAV fmt 块解析结果（仅用于嵌入式响应头提示）
type WAVFmt struct {
	AudioFormat   uint16
	Channels      uint16
	SampleRate    uint32
	BitsPerSample uint16
}

// ConsumeWAVHeaderToData 从 br 读出并丢弃 WAV 容器头，停在 data 正文首字节；PCM 裸流从这里开始读。
func ConsumeWAVHeaderToData(br *bufio.Reader) (*WAVFmt, error) {
	var riffHdr [12]byte
	if _, err := io.ReadFull(br, riffHdr[:]); err != nil {
		return nil, fmt.Errorf("wav: 读取头部: %w", err)
	}
	if string(riffHdr[0:4]) != "RIFF" || string(riffHdr[8:12]) != "WAVE" {
		return nil, fmt.Errorf("wav: 不是 RIFF/WAVE 文件")
	}

	var wavFmt *WAVFmt

	for {
		var chunkID [4]byte
		if _, err := io.ReadFull(br, chunkID[:]); err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("wav: 未找到 data 块")
			}
			return nil, fmt.Errorf("wav: 读 chunk id: %w", err)
		}
		var chunkSize uint32
		if err := binary.Read(br, binary.LittleEndian, &chunkSize); err != nil {
			return nil, fmt.Errorf("wav: 读 chunk 大小: %w", err)
		}

		id := string(chunkID[:])

		switch id {
		case "fmt ":
			buf := make([]byte, chunkSize)
			if _, err := io.ReadFull(br, buf); err != nil {
				return nil, fmt.Errorf("wav: fmt 块: %w", err)
			}
			wavFmt = parseWAVFmt(buf)
			if wavFmt != nil && wavFmt.AudioFormat != 1 {
				logx.Errorf("wav: 警告 非 PCM 音频格式(audio_format=%d)，嵌入式可能无法播放", wavFmt.AudioFormat)
			}
			if chunkSize%2 == 1 {
				_, _ = br.ReadByte()
			}
		case "data":
			if wavFmt == nil {
				logx.Slowf("wav: 未遇到 fmt 即出现 data，将缺少采样率响应头提示")
			}
			return wavFmt, nil
		default:
			if _, err := io.CopyN(io.Discard, br, int64(chunkSize)); err != nil {
				return nil, fmt.Errorf("wav: 跳过 chunk %q: %w", id, err)
			}
			if chunkSize%2 == 1 {
				_, _ = br.ReadByte()
			}
		}
	}
}

func parseWAVFmt(b []byte) *WAVFmt {
	if len(b) < 16 {
		return nil
	}
	return &WAVFmt{
		AudioFormat:   binary.LittleEndian.Uint16(b[0:2]),
		Channels:      binary.LittleEndian.Uint16(b[2:4]),
		SampleRate:    binary.LittleEndian.Uint32(b[4:8]),
		BitsPerSample: binary.LittleEndian.Uint16(b[14:16]),
	}
}

// embeddedAudioFmtHint HTTP 头 X-Embedded-Audio-Fmt 取值（wav pcm_raw）
func embeddedAudioFmtHint(f *WAVFmt) string {
	if f == nil {
		return "pcm_le;bits=16;channels=2;rate=48000"
	}
	return fmt.Sprintf("pcm_le;bits=%d;channels=%d;rate=%d", f.BitsPerSample, f.Channels, f.SampleRate)
}
