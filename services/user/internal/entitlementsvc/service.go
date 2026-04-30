package entitlementsvc

import (
	"context"
	"fmt"
)

const (
	ModeMembership = "membership"
	ModeQuota      = "quota"
)

type Input struct {
	Mode        string
	BenefitCode string
	FeatureKey  string
	Delta       int64
}

type Result struct {
	HasPermission bool
	Reason        string
	LevelCode     string
	Remaining     int64
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Check(ctx context.Context, userID int64, input Input) (*Result, error) {
	return &Result{
		HasPermission: true,
		Reason:        "校验通过",
	}, nil
}

func (s *Service) ConsumeQuota(ctx context.Context, userID int64, featureKey string, delta int64) error {
	return fmt.Errorf("未实现")
}
