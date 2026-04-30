package passwd

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// 密码哈希算法标识（写入 users.password_algo）。
const (
	AlgoBcryptConcat = "bcrypt_concat" // 现有：bcrypt(salt + password)
	AlgoArgon2id     = "argon2id"
)

// SecurityConfig 哈希参数（来自服务配置）。
type SecurityConfig struct {
	PasswordHashAlgo string
	BcryptCost       int
	Argon2Time       uint32
	Argon2Memory     uint32
	Argon2Threads    uint8
	Argon2KeyLen     uint32
}

// DefaultSecurityConfig 默认与历史数据兼容。
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		PasswordHashAlgo: AlgoBcryptConcat,
		BcryptCost:       BcryptCost,
		Argon2Time:       3,
		Argon2Memory:     64 * 1024,
		Argon2Threads:    4,
		Argon2KeyLen:     32,
	}
}

// NormalizeAlgo 未知算法回退到 bcrypt_concat，保证登录可校验历史用户。
func NormalizeAlgo(s string) string {
	v := strings.TrimSpace(strings.ToLower(s))
	switch v {
	case "", AlgoBcryptConcat, AlgoArgon2id:
		if v == "" {
			return AlgoBcryptConcat
		}
		return v
	default:
		return AlgoBcryptConcat
	}
}

// HashPasswordWithConfig 按配置生成 salt、hash、算法名。
func HashPasswordWithConfig(password string, cfg SecurityConfig) (salt, hash, algo string, err error) {
	algo = NormalizeAlgo(cfg.PasswordHashAlgo)
	switch algo {
	case AlgoArgon2id:
		return hashArgon2id(password, cfg)
	default:
		return hashBcryptConcat(password, effectiveBcryptCost(cfg))
	}
}

func effectiveBcryptCost(cfg SecurityConfig) int {
	if cfg.BcryptCost < bcrypt.MinCost {
		return BcryptCost
	}
	if cfg.BcryptCost > bcrypt.MaxCost {
		return bcrypt.MaxCost
	}
	return cfg.BcryptCost
}

func hashBcryptConcat(password string, cost int) (salt, hash, algo string, err error) {
	salt, err = GenerateSalt()
	if err != nil {
		return "", "", "", err
	}
	salted := salt + password
	b, err := bcrypt.GenerateFromPassword([]byte(salted), cost)
	if err != nil {
		return "", "", "", err
	}
	return salt, string(b), AlgoBcryptConcat, nil
}

func hashArgon2id(password string, cfg SecurityConfig) (salt, hash, algo string, err error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", "", err
	}
	t := cfg.Argon2Time
	if t == 0 {
		t = 3
	}
	mem := cfg.Argon2Memory
	if mem == 0 {
		mem = 64 * 1024
	}
	th := cfg.Argon2Threads
	if th == 0 {
		th = 4
	}
	kl := cfg.Argon2KeyLen
	if kl == 0 {
		kl = 32
	}
	key := argon2.IDKey([]byte(password), saltBytes, t, mem, th, kl)
	salt = hex.EncodeToString(saltBytes)
	hash = base64.RawStdEncoding.EncodeToString(key)
	return salt, hash, AlgoArgon2id, nil
}

// VerifyPasswordWithConfig 按算法校验；algo 为空视为 bcrypt_concat。
func VerifyPasswordWithConfig(algo, salt, password, hashed string, cfg SecurityConfig) bool {
	a := NormalizeAlgo(algo)
	switch a {
	case AlgoArgon2id:
		return verifyArgon2id(salt, password, hashed, cfg)
	default:
		return VerifyPassword(salt, password, hashed)
	}
}

func verifyArgon2id(saltHex, password, hashB64 string, cfg SecurityConfig) bool {
	saltBytes, err := hex.DecodeString(saltHex)
	if err != nil || len(saltBytes) == 0 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil || len(want) == 0 {
		return false
	}
	t := cfg.Argon2Time
	if t == 0 {
		t = 3
	}
	mem := cfg.Argon2Memory
	if mem == 0 {
		mem = 64 * 1024
	}
	th := cfg.Argon2Threads
	if th == 0 {
		th = 4
	}
	kl := uint32(len(want))
	if kl == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), saltBytes, t, mem, th, kl)
	return subtle.ConstantTimeCompare(got, want) == 1
}
