package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/model"
)

// ThirdPartyAccountRepo 第三方账号数据访问层
type ThirdPartyAccountRepo struct {
	db *sql.DB
}

// NewThirdPartyAccountRepo 创建第三方账号 Repository
func NewThirdPartyAccountRepo(db *sql.DB) *ThirdPartyAccountRepo {
	return &ThirdPartyAccountRepo{db: db}
}

// Create 创建新的第三方账号记录
func (r *ThirdPartyAccountRepo) Create(ctx context.Context, account *model.ThirdPartyAccount) error {
	query := `
		INSERT INTO third_party_account (
			user_id, provider, open_id, union_id, nickname, avatar,
			gender, country_code, province, city,
			access_token, refresh_token, token_expires_at, raw_data,
			status, last_login_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, NOW()
		)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx, query,
		account.UserID, account.Provider, account.OpenID, account.UnionID,
		account.Nickname, account.Avatar, account.Gender, account.CountryCode,
		account.Province, account.City, account.AccessToken, account.RefreshToken,
		account.TokenExpiresAt, account.RawData.String, account.Status,
	).Scan(&account.ID)

	if err != nil {
		return fmt.Errorf("创建第三方账号失败: %w", err)
	}

	return nil
}

// UpdateAccountInfo 更新账号完整信息（扫码回调时使用）
func (r *ThirdPartyAccountRepo) UpdateAccountInfo(
	ctx context.Context,
	id int64,
	userID int64,
	openID string,
	nickname string,
	avatar string,
	gender int16,
	countryCode string,
	province string,
	city string,
	accessToken string,
	rawData sql.NullString,
) error {
	query := `
		UPDATE third_party_account
		SET user_id = $1,
		    open_id = $2,
		    nickname = $3,
		    avatar = COALESCE(NULLIF($4, ''), avatar),
		    gender = COALESCE($5, gender),
		    country_code = COALESCE(NULLIF($6, ''), country_code),
		    province = COALESCE(NULLIF($7, ''), province),
		    city = COALESCE(NULLIF($8, ''), city),
		    access_token = COALESCE(NULLIF($9, ''), access_token),
		    raw_data = $10::jsonb,
		    last_login_at = NOW(),
		    updated_at = NOW()
		WHERE id = $11 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		userID, openID, nickname, avatar, gender,
		countryCode, province, city, accessToken, rawData, id,
	)
	if err != nil {
		return fmt.Errorf("更新账号信息失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("账号不存在或已被删除")
	}

	return nil
}

// FindByQRCodeKey 根据二维码 Key 查找账号信息
// 通过 raw_data 中的 qrcode_key 字段查找（扫码登录时使用）
func (r *ThirdPartyAccountRepo) FindByQRCodeKey(ctx context.Context, qrcodeKey string) (*model.ThirdPartyAccount, error) {
	query := `
		SELECT 
			id, user_id, provider, open_id, union_id, nickname, avatar,
			gender, country_code, province, city,
			access_token, refresh_token, token_expires_at, raw_data,
			status, last_login_at, created_at, updated_at, deleted_at
		FROM third_party_account
		WHERE raw_data->>'qrcode_key' = $1
		  AND status = $2
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	var account model.ThirdPartyAccount
	err := r.db.QueryRowContext(ctx, query, qrcodeKey, model.ThirdPartyAccountStatusNormal).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.OpenID, &account.UnionID,
		&account.Nickname, &account.Avatar, &account.Gender, &account.CountryCode,
		&account.Province, &account.City, &account.AccessToken, &account.RefreshToken,
		&account.TokenExpiresAt, &account.RawData, &account.Status, &account.LastLoginAt,
		&account.CreatedAt, &account.UpdatedAt, &account.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("根据二维码 Key 查询第三方账号失败: %w", err)
	}

	return &account, nil
}

// FindByProviderAndOpenID 根据平台和 OpenID 查找账号
func (r *ThirdPartyAccountRepo) FindByProviderAndOpenID(ctx context.Context, provider, openID string) (*model.ThirdPartyAccount, error) {
	query := `
		SELECT 
			id, user_id, provider, open_id, union_id, nickname, avatar,
			gender, country_code, province, city,
			access_token, refresh_token, token_expires_at, raw_data,
			status, last_login_at, created_at, updated_at, deleted_at
		FROM third_party_account
		WHERE provider = $1 AND open_id = $2
		  AND status = $3
		  AND deleted_at IS NULL
		LIMIT 1
	`

	var account model.ThirdPartyAccount
	err := r.db.QueryRowContext(ctx, query, provider, openID, model.ThirdPartyAccountStatusNormal).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.OpenID, &account.UnionID,
		&account.Nickname, &account.Avatar, &account.Gender, &account.CountryCode,
		&account.Province, &account.City, &account.AccessToken, &account.RefreshToken,
		&account.TokenExpiresAt, &account.RawData, &account.Status, &account.LastLoginAt,
		&account.CreatedAt, &account.UpdatedAt, &account.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询第三方账号失败: %w", err)
	}

	return &account, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *ThirdPartyAccountRepo) UpdateLastLogin(ctx context.Context, id int64) error {
	query := `
		UPDATE third_party_account
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("更新最后登录时间失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("账号不存在或已被删除")
	}

	return nil
}

// UpdateTokenInfo 更新令牌信息
func (r *ThirdPartyAccountRepo) UpdateTokenInfo(ctx context.Context, id int64, accessToken, refreshToken string, expiresAt *time.Time) error {
	query := `
		UPDATE third_party_account
		SET access_token = $1,
		    refresh_token = COALESCE(NULLIF($2, ''), refresh_token),
		    token_expires_at = $3,
		    updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, accessToken, refreshToken, expiresAt, id)
	if err != nil {
		return fmt.Errorf("更新令牌信息失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("账号不存在或已被删除")
	}

	return nil
}
