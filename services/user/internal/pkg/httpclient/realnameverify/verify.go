package realnameverify

import (
	"context"

	"github.com/google/uuid"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
)

// Outcome 三方核验结果语义。
type Outcome int

const (
	OutcomePass Outcome = iota
	OutcomeFail
	OutcomeError
)

// Result 核验返回（不含敏感证件明文）。
type Result struct {
	Outcome    Outcome
	FlowNo     string
	RawJSON    string
	FailReason string
}

// Input 送检参数。
type Input struct {
	RealName string
	IDNumber string
	CertType int16
}

// Verifier 实名核验抽象（可替换为 HTTP 调用）。
type Verifier interface {
	Verify(ctx context.Context, in Input) (Result, error)
}

// NewFromConfig 按配置构造核验器；当前仅实现 mock。
func NewFromConfig(cfg config.Config) Verifier {
	if cfg.RealName.EffectiveProvider() == "http" {
		return &noopVerifier{outcome: OutcomeError}
	}
	return &mockVerifier{outcome: cfg.RealName.EffectiveMockOutcome()}
}

type mockVerifier struct {
	outcome string
}

func (m *mockVerifier) Verify(_ context.Context, _ Input) (Result, error) {
	switch m.outcome {
	case "fail":
		return Result{
			Outcome:    OutcomeFail,
			FlowNo:     "mock-" + uuid.New().String(),
			RawJSON:    `{"mock":true,"pass":false}`,
			FailReason: "身份信息核验未通过",
		}, nil
	case "error":
		return Result{Outcome: OutcomeError, RawJSON: `{"mock":true,"error":true}`}, nil
	default:
		return Result{
			Outcome: OutcomePass,
			FlowNo:  "mock-" + uuid.New().String(),
			RawJSON: `{"mock":true,"pass":true}`,
		}, nil
	}
}

type noopVerifier struct {
	outcome Outcome
}

func (n *noopVerifier) Verify(_ context.Context, _ Input) (Result, error) {
	return Result{Outcome: n.outcome, RawJSON: `{"provider":"http","stub":true}`}, nil
}
