package kafkax

import (
	"context"
	"testing"
)

func TestNewProducerNoBrokers(t *testing.T) {
	p, err := NewProducer(Config{})
	if err != nil || p != nil {
		t.Fatalf("want nil producer, got p=%v err=%v", p, err)
	}
	if err := p.Publish(context.Background(), "t", nil, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}
