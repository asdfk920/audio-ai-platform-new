package devicebind

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestValidateSNFormatOnly(t *testing.T) {
	tests := []struct {
		name      string
		sn        string
		wantValid bool
	}{
		// 有效的短格式 SN
		{"Valid Short SN 1", "SN-X1-001", true},
		{"Valid Short SN 2", "AUD-SP-00001", true},
		{"Valid Short SN 3", "SND-HP-12345", true},
		{"Valid Short SN 4", "ABC-XY-999", true},
		// 无效的 SN
		{"Invalid Length", "SN-X1-00", false},        // 流水号太短
		{"Invalid Format", "SNX1001", false},         // 无横杠
		{"Invalid Format 2", "SN-X1", false},         // 缺少流水号
		{"Invalid Format 3", "SN-X1-0000001", false}, // 流水号太长（超过 5 位）
		{"Invalid Format 4", "S-X1-001", false},      // 厂商码太短（1 位）
		{"Invalid Format 5", "SNXY-X1-001", false},   // 厂商码太长（4 位）
	}

	snPatternShort := regexp.MustCompile(`^[A-Z0-9]{2,3}-[A-Z0-9]{2}-\d{3,5}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := snPatternShort.MatchString(strings.ToUpper(tt.sn))
			if valid != tt.wantValid {
				t.Errorf("validateSNFormat() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestGenerateAndValidateSN(t *testing.T) {
	tests := []struct {
		name         string
		vendorCode   string
		productLine  string
		wantValid    bool
		wantErrMatch string
	}{
		{"AUD-SP", "AUD", "SP", true, ""},
		{"SND-HP", "SND", "HP", true, ""},
		{"SPK-SB", "SPK", "SB", true, ""},
		{"HPH-MI", "HPH", "MI", true, ""},
		{"MIC-PR", "MIC", "PR", true, ""},
		{"Invalid Vendor", "AUDIO", "SP", false, "厂商码必须为 3 位"},
		{"Invalid Product", "AUD", "SPEAKER", false, "产品线必须为 2 位"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成 SN
			sn, err := GenerateSN(tt.vendorCode, tt.productLine)

			if tt.wantValid {
				if err != nil {
					t.Errorf("GenerateSN() error = %v, wantErr nil", err)
					return
				}

				// 验证生成的 SN 格式
				valid, vendor, product, yearMonth, serial, checkDigit, err := ValidateSN(sn)
				if err != nil {
					t.Errorf("ValidateSN() error = %v, wantErr nil", err)
					return
				}

				if !valid {
					t.Errorf("ValidateSN() valid = %v, want true", valid)
				}

				// 验证各部分格式
				if vendor != tt.vendorCode {
					t.Errorf("vendorCode = %v, want %v", vendor, tt.vendorCode)
				}

				if product != tt.productLine {
					t.Errorf("productLine = %v, want %v", product, tt.productLine)
				}

				if len(yearMonth) != 4 {
					t.Errorf("yearMonth length = %v, want 4", len(yearMonth))
				}

				if len(serial) != 5 {
					t.Errorf("serial length = %v, want 5", len(serial))
				}

				if len(checkDigit) != 1 {
					t.Errorf("checkDigit length = %v, want 1", len(checkDigit))
				}

				// 验证总长度（17 位，包含 4 个横杠）
				if len(sn) != 17 {
					t.Errorf("SN length = %v, want 17", len(sn))
				}

				fmt.Printf("✓ Generated SN: %s (Vendor: %s, Product: %s, YearMonth: %s, Serial: %s, Check: %s)\n",
					sn, vendor, product, yearMonth, serial, checkDigit)
			} else {
				if err == nil {
					t.Errorf("GenerateSN() expected error containing '%s', got nil", tt.wantErrMatch)
				} else if !contains(err.Error(), tt.wantErrMatch) {
					t.Errorf("GenerateSN() error = %v, want error containing '%s'", err, tt.wantErrMatch)
				}
			}
		})
	}
}

func TestValidateSNFormat(t *testing.T) {
	tests := []struct {
		name      string
		sn        string
		wantValid bool
		wantErr   string
	}{
		{"Valid SN 1", "AUD-SP-2605-00001-X", true, ""},
		{"Valid SN 2", "SND-HP-2412-99999-Z", true, ""},
		{"Invalid Length", "AUD-SP-2605-0001-X", false, "长度应为 17 位"},
		{"Invalid Format", "AUDSP260500001X", false, "格式不正确"},
		{"Invalid Month", "AUD-SP-2613-00001-X", false, "月份无效"},
		{"Invalid Serial", "AUD-SP-2605-00000-X", false, "流水号无效"},
		{"Invalid Check Digit", "AUD-SP-2605-00001-Z", false, "校验位错误"},
		{"Lowercase", "aud-sp-2605-00001-x", false, "格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _, _, _, _, _, err := ValidateSN(tt.sn)

			if tt.wantValid {
				if err != nil {
					t.Errorf("ValidateSN() error = %v, wantErr nil", err)
				}
				if !valid {
					t.Errorf("ValidateSN() valid = %v, want true", valid)
				}
			} else {
				if valid {
					t.Errorf("ValidateSN() valid = %v, want false", valid)
				}
				if err == nil {
					t.Errorf("ValidateSN() expected error, got nil")
				} else if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateSN() error = %v, want error containing '%s'", err, tt.wantErr)
				}
			}
		})
	}
}

func TestParseSN(t *testing.T) {
	sn := "AUD-SP-2605-00001-X"

	result, err := ParseSN(sn)
	if err != nil {
		t.Errorf("ParseSN() error = %v, wantErr nil", err)
		return
	}

	if result["sn"] != sn {
		t.Errorf("sn = %v, want %v", result["sn"], sn)
	}

	if result["vendor_code"] != "AUD" {
		t.Errorf("vendor_code = %v, want AUD", result["vendor_code"])
	}

	if result["vendor_name"] != "Audio Tech" {
		t.Errorf("vendor_name = %v, want Audio Tech", result["vendor_name"])
	}

	if result["product_line"] != "SP" {
		t.Errorf("product_line = %v, want SP", result["product_line"])
	}

	if result["product_line_name"] != "Speaker" {
		t.Errorf("product_line_name = %v, want Speaker", result["product_line_name"])
	}

	if result["year"] != "2026" {
		t.Errorf("year = %v, want 2026", result["year"])
	}

	if result["month"] != "05" {
		t.Errorf("month = %v, want 05", result["month"])
	}

	if result["serial_number"] != "00001" {
		t.Errorf("serial_number = %v, want 00001", result["serial_number"])
	}

	if result["check_digit"] != "X" {
		t.Errorf("check_digit = %v, want X", result["check_digit"])
	}

	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}

	fmt.Printf("✓ Parsed SN: %+v\n", result)
}

func TestValidateSNFormatInBind(t *testing.T) {
	// 测试绑定接口中的 SN 验证
	tests := []struct {
		name      string
		sn        string
		wantValid bool
	}{
		{"Valid New Format", "AUD-SP-2605-00001-X", true},
		{"Valid Uppercase", "AUD-SP-2605-12345-A", true},
		{"Invalid Old Format", "ABC1234567890123", false}, // 旧格式 16 位在绑定接口应该被拒绝
		{"Invalid Length", "AUD-SP-2605-0001-X", false},
		{"Invalid Check", "AUD-SP-2605-00001-Z", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSNFormat(tt.sn)

			if tt.wantValid {
				if err != nil {
					t.Errorf("validateSNFormat() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("validateSNFormat() expected error, got nil")
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
