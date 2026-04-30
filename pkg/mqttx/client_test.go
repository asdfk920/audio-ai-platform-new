package mqttx

import "testing"

func TestNewClientNoBroker(t *testing.T) {
	c, err := NewClient(Config{})
	if err != nil || c != nil {
		t.Fatalf("want nil client, got c=%v err=%v", c, err)
	}
	c.Disconnect()
	if err := c.Publish("t", 0, false, []byte("x")); err != nil {
		t.Fatal(err)
	}
}
