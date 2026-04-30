package passwd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	SaltLen     = 16
	BcryptCost = 10
)

// GenerateSalt 生成随机盐
func GenerateSalt() (string, error) {
	b := make([]byte, SaltLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashPassword 使用盐对密码做 bcrypt 哈希（salt+password 再哈希，盐单独存储）
func HashPassword(salt, password string) (string, error) {
	salted := salt + password
	hashed, err := bcrypt.GenerateFromPassword([]byte(salted), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword 校验密码
func VerifyPassword(salt, password, hashedPassword string) bool {
	salted := salt + password
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(salted))
	return err == nil
}

// HashPasswordWithNewSalt 生成新盐并哈希密码，返回盐与哈希值
func HashPasswordWithNewSalt(password string) (salt, hashed string, err error) {
	salt, err = GenerateSalt()
	if err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}
	hashed, err = HashPassword(salt, password)
	if err != nil {
		return "", "", fmt.Errorf("hash password: %w", err)
	}
	return salt, hashed, nil
}
