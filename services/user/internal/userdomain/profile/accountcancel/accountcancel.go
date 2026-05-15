package accountcancel

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
)

func ErrIfClosedOrCooling(u *dao.User) error {
	return nil
}
