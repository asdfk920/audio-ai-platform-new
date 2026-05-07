package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// SN 格式：厂商码 (3 位) + 产品线 (2 位) + 年月 (4 位) + 流水号 (5 位) + 校验位 (1 位)
	// 示例：AUD-SP-2605-00001-X
	snPattern = regexp.MustCompile(`^[A-Z0-9]{3}-[A-Z0-9]{2}-\d{4}-\d{5}-[A-Z0-9]$`)
	
	// 厂商码映射（3 位字母数字组合）
	vendorCodes = map[string]string{
		"AUD": "Audio Tech",
		"SND": "Sound Pro",
		"SPK": "Speaker Co",
		"HPH": "Headphone Inc",
		"MIC": "Mic Master",
	}
	
	// 产品线代码（2 位字母数字组合）
	productLines = map[string]string{
		"SP": "Speaker",
		"HP": "Headphone",
		"SB": "SoundBar",
		"MI": "Mini",
		"PR": "Pro",
		"X1": "X1 Series",
		"X2": "X2 Series",
	}
)

// GenerateSN 生成符合规范的设备序列号
// 格式：厂商码 (3 位) + 产品线 (2 位) + 年月 (4 位) + 流水号 (5 位) + 校验位 (1 位)
// 示例：AUD-SP-2605-00001-X
func GenerateSN(vendorCode, productLine string) (string, error) {
	// 验证厂商码
	if len(vendorCode) != 3 {
		return "", fmt.Errorf("厂商码必须为 3 位，当前：%s", vendorCode)
	}
	
	// 验证产品线
	if len(productLine) != 2 {
		return "", fmt.Errorf("产品线必须为 2 位，当前：%s", productLine)
	}
	
	// 生成年月（YYMM 格式）
	now := time.Now()
	year := now.Year() % 100
	month := int(now.Month())
	yearMonth := fmt.Sprintf("%02d%02d", year, month)
	
	// 生成流水号（00001-99999）
	serialNum, err := rand.Int(rand.Reader, big.NewInt(99999))
	if err != nil {
		return "", fmt.Errorf("生成流水号失败：%w", err)
	}
	serialNum = serialNum.Add(serialNum, big.NewInt(1))
	serialStr := fmt.Sprintf("%05d", serialNum.Int64())
	
	// 拼接前缀（不含校验位）
	prefix := fmt.Sprintf("%s-%s-%s-%s",
		strings.ToUpper(vendorCode),
		strings.ToUpper(productLine),
		yearMonth,
		serialStr,
	)
	
	// 生成校验位
	checkDigit := generateCheckDigit(prefix)
	
	sn := prefix + "-" + string(checkDigit)
	return sn, nil
}

// generateCheckDigit 根据前缀生成校验位（第 15 位）
// 算法：Luhn 算法变体，将字母转换为数字后计算
func generateCheckDigit(prefix string) byte {
	sum := 0
	for i, ch := range prefix {
		var val int
		if ch >= '0' && ch <= '9' {
			val = int(ch - '0')
		} else if ch >= 'A' && ch <= 'Z' {
			val = int(ch - 'A' + 10)
		} else {
			val = i // 其他字符（如横杠）使用位置索引
		}
		sum += val
	}
	
	// 校验位范围：0-9, A-Z（36 进制）
	checkVal := (36 - (sum % 36)) % 36
	if checkVal < 10 {
		return byte('0' + checkVal)
	}
	return byte('A' + checkVal - 10)
}

// ValidateSN 验证序列号格式是否合法
// 返回：是否合法、厂商码、产品线、生产年月、流水号、校验位、错误信息
func ValidateSN(sn string) (bool, string, string, string, string, string, error) {
	sn = strings.TrimSpace(sn)
	
	// 基本长度检查（15 位）
	if len(sn) != 17 { // 包含 4 个横杠
		return false, "", "", "", "", "", fmt.Errorf("序列号长度应为 17 位（含横杠），当前：%d", len(sn))
	}
	
	// 格式检查
	if !snPattern.MatchString(sn) {
		return false, "", "", "", "", "", fmt.Errorf("序列号格式不正确，应为 XXX-XX-YYYY-NNNNN-X 格式")
	}
	
	// 分解各部分
	parts := strings.Split(sn, "-")
	if len(parts) != 5 {
		return false, "", "", "", "", "", fmt.Errorf("序列号分段错误")
	}
	
	vendorCode := parts[0]      // 厂商码 (3 位)
	productLine := parts[1]     // 产品线 (2 位)
	yearMonth := parts[2]       // 年月 (4 位)
	serialNum := parts[3]       // 流水号 (5 位)
	checkDigit := parts[4]      // 校验位 (1 位)
	
	// 验证校验位
	prefix := fmt.Sprintf("%s-%s-%s-%s", vendorCode, productLine, yearMonth, serialNum)
	expectedCheckDigit := generateCheckDigit(prefix)
	if expectedCheckDigit != checkDigit[0] {
		return false, "", "", "", "", "", fmt.Errorf("校验位错误，应为：%c", expectedCheckDigit)
	}
	
	// 验证年月是否合理
	year, _ := strconv.Atoi(yearMonth[:2])
	month, _ := strconv.Atoi(yearMonth[2:])
	if month < 1 || month > 12 {
		return false, "", "", "", "", "", fmt.Errorf("月份无效：%d", month)
	}
	
	// 验证流水号
	serialInt, err := strconv.Atoi(serialNum)
	if err != nil || serialInt < 1 || serialInt > 99999 {
		return false, "", "", "", "", "", fmt.Errorf("流水号无效：%s", serialNum)
	}
	
	return true, vendorCode, productLine, yearMonth, serialNum, checkDigit, nil
}

// GetVendorName 根据厂商码获取厂商名称
func GetVendorName(vendorCode string) string {
	if name, ok := vendorCodes[strings.ToUpper(vendorCode)]; ok {
		return name
	}
	return "未知厂商"
}

// GetProductLineName 根据产品线代码获取产品线名称
func GetProductLineName(productLine string) string {
	if name, ok := productLines[strings.ToUpper(productLine)]; ok {
		return name
	}
	return "未知产品线"
}

// ParseSN 解析序列号，返回详细信息
func ParseSN(sn string) (map[string]interface{}, error) {
	valid, vendorCode, productLine, yearMonth, serialNum, checkDigit, err := ValidateSN(sn)
	if err != nil {
		return nil, err
	}
	
	if !valid {
		return nil, fmt.Errorf("序列号无效")
	}
	
	result := map[string]interface{}{
		"sn":              sn,
		"vendor_code":     vendorCode,
		"vendor_name":     GetVendorName(vendorCode),
		"product_line":    productLine,
		"product_line_name": GetProductLineName(productLine),
		"year_month":      yearMonth,
		"year":            "20" + yearMonth[:2],
		"month":           yearMonth[2:],
		"serial_number":   serialNum,
		"check_digit":     checkDigit,
		"valid":           true,
	}
	
	return result, nil
}
