package memberpay

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// NewOrderNo 生成唯一订单号（M + 毫秒时间 + 随机 8 位十六进制）。
func NewOrderNo() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("M%d%08X", time.Now().UnixMilli(), binary.BigEndian.Uint32(b[:]))
}
