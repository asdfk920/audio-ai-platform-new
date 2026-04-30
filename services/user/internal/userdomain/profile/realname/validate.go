package realname

import (
	"encoding/base64"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/validate"
)

var (
	namePattern = regexp.MustCompile(`^[\p{Han}a-zA-Z0-9·\-\s]{2,50}$`)
	usccPattern = regexp.MustCompile(`^[0-9A-Z]{18}$`)
	maxB64Photo = 6 << 20 // 约 4.5MB 原始二进制上限的 base64 体量防护
	maxB64Face  = 6 << 20
)

// ValidateSubmit 校验姓名、证件号，以及证件影像 / 人脸（个人身份证须人像面+国徽面；企业可选）。
// legacyIdPhotoB64 兼容旧字段 id_photo_base64，视为「人像面」。
// requireFacePersonal 为 true 时个人须上传人脸 Base64。
func ValidateSubmit(certType int16, realName, idNumber string, legacyIdPhotoB64, idCardFrontB64, idCardBackB64, faceB64 string, requireFacePersonal bool) error {
	name := strings.TrimSpace(realName)
	id := strings.TrimSpace(idNumber)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 50 {
		return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
	}
	if !namePattern.MatchString(name) {
		return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
	}
	switch certType {
	case CertTypePersonal:
		id = strings.ToUpper(id)
		if err := validate.CheckIDCard(id); err != nil {
			return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
		}
	case CertTypeEnterprise:
		id = strings.ToUpper(id)
		if !usccPattern.MatchString(id) {
			return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
		}
	default:
		return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
	}

	front := strings.TrimSpace(idCardFrontB64)
	if front == "" {
		front = strings.TrimSpace(legacyIdPhotoB64)
	}
	back := strings.TrimSpace(idCardBackB64)
	face := strings.TrimSpace(faceB64)

	switch certType {
	case CertTypePersonal:
		if front == "" || back == "" {
			return errorx.NewCodeError(errorx.CodeRealNameInvalidPayload, "请上传身份证人像面与国徽面照片")
		}
		if err := checkBase64Image(front, maxB64Photo); err != nil {
			return err
		}
		if err := checkBase64Image(back, maxB64Photo); err != nil {
			return err
		}
		if requireFacePersonal {
			if face == "" {
				return errorx.NewCodeError(errorx.CodeRealNameInvalidPayload, "请完成人脸核验并上传人脸照片")
			}
			if err := checkBase64Image(face, maxB64Face); err != nil {
				return err
			}
		} else if face != "" {
			if err := checkBase64Image(face, maxB64Face); err != nil {
				return err
			}
		}
	case CertTypeEnterprise:
		if front != "" {
			if err := checkBase64Image(front, maxB64Photo); err != nil {
				return err
			}
		}
		if back != "" {
			if err := checkBase64Image(back, maxB64Photo); err != nil {
				return err
			}
		}
		if face != "" {
			if err := checkBase64Image(face, maxB64Face); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBase64Image(s string, maxLen int) error {
	if len(s) > maxLen {
		return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
	}
	if _, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s)); err != nil {
		return errorx.NewDefaultError(errorx.CodeRealNameInvalidPayload)
	}
	return nil
}
