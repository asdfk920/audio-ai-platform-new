package familysvc

import "github.com/jacklau/audio-ai-platform/services/user/internal/svc"

type Service struct {
	svcCtx *svc.ServiceContext
}

type FamilyView struct {
	ID          int64
	Name        string
	OwnerUserID int64
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) EnsureFamilyForOwner(ctx interface{}, userID int64, name string) (*FamilyView, error) {
	return nil, nil
}
