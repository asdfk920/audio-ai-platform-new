package familysvc

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

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
	c := ctx.(context.Context)
	tx, err := s.svcCtx.DB.BeginTx(c, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	family, member, err := dao.FindCurrentFamilyByUser(c, tx, userID)
	if err != nil {
		return nil, err
	}

	if family != nil && member != nil && member.Role == dao.FamilyRoleOwner {
		return &FamilyView{
			ID:          family.ID,
			Name:        family.Name,
			OwnerUserID: family.OwnerUserID,
		}, nil
	}

	if family == nil {
		newFamily, err := dao.InsertFamily(c, tx, userID, name)
		if err != nil {
			return nil, err
		}
		if _, err := dao.InsertFamilyMember(c, tx, newFamily.ID, userID, userID, dao.FamilyRoleOwner); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &FamilyView{
			ID:          newFamily.ID,
			Name:        name,
			OwnerUserID: userID,
		}, nil
	}

	if member == nil || member.Role != dao.FamilyRoleOwner {
		if _, err := dao.InsertFamilyMember(c, tx, family.ID, userID, userID, dao.FamilyRoleOwner); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &FamilyView{
			ID:          family.ID,
			Name:        family.Name,
			OwnerUserID: family.OwnerUserID,
		}, nil
	}

	return &FamilyView{
		ID:          family.ID,
		Name:        family.Name,
		OwnerUserID: family.OwnerUserID,
	}, nil
}
