package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// UpdateUserInfoHandler 修改用户信息（支持头像上传）
func UpdateUserInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateUserInfoReq

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		req.Username = r.FormValue("username")
		req.Nickname = r.FormValue("nickname")
		req.Avatar = r.FormValue("avatar")
		req.Birthday = r.FormValue("birthday")
		req.RealName = r.FormValue("real_name")
		req.Constellation = r.FormValue("constellation")
		req.Signature = r.FormValue("signature")
		req.Bio = r.FormValue("bio")
		req.Hobbies = r.FormValue("hobbies")
		req.Location = r.FormValue("location")
		req.Language = r.FormValue("language")
		req.Timezone = r.FormValue("timezone")

		if genderStr := r.FormValue("gender"); genderStr != "" {
			if gender, err := strconv.Atoi(genderStr); err == nil {
				req.Gender = gender
			}
		}

		if ageStr := r.FormValue("age"); ageStr != "" {
			if age, err := strconv.Atoi(ageStr); err == nil {
				req.Age = age
			}
		}

		if birthdayVisStr := r.FormValue("birthday_visibility"); birthdayVisStr != "" {
			if birthdayVis, err := strconv.Atoi(birthdayVisStr); err == nil {
				req.BirthdayVisibility = birthdayVis
			}
		}

		if genderVisStr := r.FormValue("gender_visibility"); genderVisStr != "" {
			if genderVis, err := strconv.Atoi(genderVisStr); err == nil {
				req.GenderVisibility = genderVis
			}
		}

		if profileCompleteStr := r.FormValue("profile_complete"); profileCompleteStr != "" {
			if profileComplete, err := strconv.Atoi(profileCompleteStr); err == nil {
				req.ProfileComplete = profileComplete
			}
		}

		if profileCompleteScoreStr := r.FormValue("profile_complete_score"); profileCompleteScoreStr != "" {
			if profileCompleteScore, err := strconv.Atoi(profileCompleteScoreStr); err == nil {
				req.ProfileCompleteScore = profileCompleteScore
			}
		}

		l := logic.NewUpdateUserInfoLogic(r.Context(), svcCtx)
		resp, err := l.UpdateUserInfoWithFile(&req, r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
