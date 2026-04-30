package dao

// SQL 片段：用户主表公开资料列。
// 注意：该列顺序必须与 scanUserProfileArgs(u) 返回的 Scan 参数顺序完全一致。
const sqlUserProfileColumns = `id, email, mobile, nickname, avatar, status, real_name_status, real_name_time, real_name_type, cancellation_cooling_until, account_cancelled_at, birthday, gender, constellation, age, signature, bio, birthday_visibility, gender_visibility, profile_complete, profile_complete_score, hobbies, location`

// scanUserProfileArgs 与 sqlUserProfileColumns 顺序一致，供 QueryRow/Row.Scan 使用。
func scanUserProfileArgs(u *User) []any {
	return []any{
		&u.Id, &u.Email, &u.Mobile, &u.Nickname, &u.Avatar, &u.Status,
		&u.RealNameStatus, &u.RealNameAt, &u.RealNameCertType,
		&u.CancellationCoolingUntil, &u.AccountCancelledAt,
		&u.Birthday, &u.Gender,
		&u.Constellation, &u.Age, &u.Signature, &u.Bio,
		&u.BirthdayVisibility, &u.GenderVisibility,
		&u.ProfileComplete, &u.ProfileCompleteScore,
		&u.Hobbies, &u.Location,
	}
}
