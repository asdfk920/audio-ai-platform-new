package util

import (
	"fmt"
	"testing"
)

func TestNormalizeSN_ArbitraryText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AUSP2605000001", "AUSP2605000001"},
		{"ausp2605000001", "ausp2605000001"},
		{"MyDevice-123", "MyDevice-123"},
		{"  spaces  ", "spaces"},
		{"中文设备名", "中文设备名"},
		{"device@#$", "device@#$"},
		{"UPPERCASE", "UPPERCASE"},
		{"lowercase", "lowercase"},
		{"MixedCase-123_abc", "MixedCase-123_abc"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeSN(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSN(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateSNFormat_ArbitraryText(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"AUSP2605000001", true},
		{"ausp2605000001", true},
		{"MyDevice-123", true},
		{"中文设备名", true},
		{"x", true},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidateSNFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateSNFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSN_ArbitraryText(t *testing.T) {
	tests := []struct {
		input       string
		expectValid bool
	}{
		{"AUSP2605000001", true},
		{"MyCustomDevice", true},
		{"device-123_abc", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseSN(tt.input)
			valid := result["valid"].(bool)
			if valid != tt.expectValid {
				t.Errorf("ParseSN(%q).valid = %v, want %v", tt.input, valid, tt.expectValid)
			}
			if valid {
				sn := result["sn"].(string)
				if sn != tt.input {
					t.Errorf("ParseSN(%q).sn = %q, want %q", tt.input, sn, tt.input)
				}
			}
		})
	}
}

func TestFormatSNDisplay_ArbitraryText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AUSP2605000001", "AUSP2605000001"},
		{"  My Device  ", "My Device"},
		{"MixedCase-123", "MixedCase-123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatSNDisplay(tt.input)
			if result != tt.expected {
				t.Errorf("FormatSNDisplay(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func ExampleNormalizeSN() {
	fmt.Println(NormalizeSN("  MyDevice-123  "))
	// Output: MyDevice-123
}

func ExampleValidateSNFormat() {
	fmt.Println(ValidateSNFormat("Any text is valid"))
	// Output: true
}
