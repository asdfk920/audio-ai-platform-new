package shadow

import "testing"

func TestValidateDeviceReported(t *testing.T) {
	if err := ValidateDeviceReported(map[string]interface{}{
		"battery": 85, "volume": 60, "online": true, "temperature": 25.5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDeviceReported(map[string]interface{}{"desired": 1}); err == nil {
		t.Fatal("expected unknown field error")
	}
	if err := ValidateDeviceReported(map[string]interface{}{
		"battery": 101, "volume": 0, "online": true,
	}); err == nil {
		t.Fatal("expected battery range error")
	}
}
