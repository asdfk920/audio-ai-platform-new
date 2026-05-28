package service

import (
	"encoding/json"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 集成测试：需设置 TEST_PG_DSN，例如
// host=14.103.202.69 port=5432 user=admin password=admin123 dbname=audio_platform sslmode=disable TimeZone=Asia/Shanghai
func TestListDeviceShadows_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	svc := PlatformDeviceService{}
	svc.Orm = db

	list, total, err := svc.ListDeviceShadows(1, 20, DeviceShadowListFilter{})
	if err != nil {
		t.Fatalf("ListDeviceShadows: %v", err)
	}
	t.Logf("total=%d len=%d", total, len(list))
	if len(list) == 0 {
		t.Fatal("expected non-empty list")
	}
	if list[0].Sn == "" {
		t.Fatalf("first row missing sn: %+v", list[0])
	}
	// 模拟 PageOK 外层 data 结构，确认 JSON 含 list 数组
	body, err := json.Marshal(struct {
		Code int `json:"code"`
		Data struct {
			Count int                    `json:"count"`
			List  []DeviceShadowListItem `json:"list"`
		} `json:"data"`
	}{Code: 200, Data: struct {
		Count int                    `json:"count"`
		List  []DeviceShadowListItem `json:"list"`
	}{Count: int(total), List: list}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("json sample: %s", string(body[:min(200, len(body))]))
}
