package util

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	SnTotalLength = 16
	SnMaxLength   = 64 // 与库表 sn VARCHAR(64) 对齐
	VendorCodeLen = 2
	DeviceTypeLen = 2
	BatchNoLen    = 4
	SerialNoLen   = 6
	CheckCodeLen  = 2
)

var vendorCodes = map[string]string{
	"AU": "AudioPlatform",
	"HX": "HuaXi",
}

var deviceTypeCodes = map[string]string{
	"SP": "Speaker",
	"HP": "Headphone",
	"EP": "EarPhone",
	"SB": "SoundBox",
	"AM": "Amplifier",
}

func GetVendorCodeFromModel(model string) string {
	modelUpper := strings.ToUpper(model)
	if strings.Contains(modelUpper, "AUDIO") || strings.Contains(modelUpper, "AUD") {
		return "AU"
	}
	if strings.Contains(modelUpper, "HUAXI") || strings.Contains(modelUpper, "HX") {
		return "HX"
	}
	return "AU"
}

func GetDeviceTypeFromModel(model string) string {
	modelUpper := strings.ToUpper(model)
	if strings.Contains(modelUpper, "SPK") || strings.Contains(modelUpper, "SPEAKER") || strings.Contains(modelUpper, "SP") {
		return "SP"
	}
	if strings.Contains(modelUpper, "HEADPHONE") || strings.Contains(modelUpper, "HP") || strings.Contains(modelUpper, "EAR") {
		return "HP"
	}
	if strings.Contains(modelUpper, "SOUNDBOX") || strings.Contains(modelUpper, "BOX") {
		return "SB"
	}
	if strings.Contains(modelUpper, "AMP") || strings.Contains(modelUpper, "AMPLIFIER") {
		return "AM"
	}
	if strings.Contains(modelUpper, "EP") {
		return "EP"
	}
	return "SP"
}

func GenerateBatchNo() string {
	now := time.Now()
	return fmt.Sprintf("%02d%02d", now.Year()%100, int(now.Month()))
}

func GetNextSerialNumber(ctx context.Context, db *sql.DB, prefix string) (int64, error) {
	var maxSerial sql.NullInt64
	query := `
		SELECT CAST(SUBSTRING(sn FROM 9 FOR 6) AS BIGINT) as max_serial
		FROM device
		WHERE sn LIKE $1 || '%'
		ORDER BY max_serial DESC
		LIMIT 1
	`
	err := db.QueryRowContext(ctx, query, prefix).Scan(&maxSerial)

	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("查询流水号失败: %v", err)
	}

	if !maxSerial.Valid || maxSerial.Int64 <= 0 {
		return 1, nil
	}

	return maxSerial.Int64 + 1, nil
}

func CalculateCheckCode(snWithoutCheck string) string {
	hash := sha256.Sum256([]byte(snWithoutCheck))
	hashStr := fmt.Sprintf("%x", hash)
	checkSum := 0
	for i := 0; i < len(hashStr); i++ {
		checkSum += int(hashStr[i])
	}
	checkChar1 := 'A' + (checkSum % 26)
	checkChar2 := '0' + ((checkSum / 26) % 10)
	return fmt.Sprintf("%c%c", checkChar1, checkChar2)
}

func GenerateSN(ctx context.Context, db *sql.DB, model string) (string, error) {
	vendorCode := GetVendorCodeFromModel(model)
	deviceType := GetDeviceTypeFromModel(model)
	batchNo := GenerateBatchNo()
	prefix := vendorCode + deviceType + batchNo

	serialNum, err := GetNextSerialNumber(ctx, db, prefix)
	if err != nil {
		return "", fmt.Errorf("获取流水号失败: %v", err)
	}

	serialStr := fmt.Sprintf("%06d", serialNum)
	snWithoutCheck := prefix + serialStr
	checkCode := CalculateCheckCode(snWithoutCheck)

	fullSn := prefix + serialStr + checkCode
	fullSn = strings.ToUpper(fullSn)

	if len(fullSn) != SnTotalLength {
		return "", fmt.Errorf("生成的SN长度错误: 期望%d位, 实际%d位 (%s)", SnTotalLength, len(fullSn), fullSn)
	}

	return fullSn, nil
}

// NormalizeSN 规范化设备序列号（去空格）
func NormalizeSN(sn string) string {
	return strings.TrimSpace(sn)
}

// ValidateDeviceSN 校验设备序列号：非空、长度上限
func ValidateDeviceSN(sn string) error {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}
	if len(sn) > SnMaxLength {
		return fmt.Errorf("设备序列号过长: 最多%d位", SnMaxLength)
	}
	return nil
}

func ValidateSNFormat(sn string) bool {
	return ValidateDeviceSN(sn) == nil
}

func ParseSN(sn string) map[string]interface{} {
	sn = strings.TrimSpace(sn)
	result := make(map[string]interface{})
	if len(sn) > 0 {
		result["sn"] = sn
		result["valid"] = true
	} else {
		result["valid"] = false
		result["error"] = "SN不能为空"
	}
	return result
}

func FormatSNDisplay(sn string) string {
	return strings.TrimSpace(sn)
}
