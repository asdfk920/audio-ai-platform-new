// Package crypto 提供音频私有格式加密/解密功能
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

const (
	// AASPMagic 私有格式魔数 "AASP" (Audio AI Secure Protocol)
	AASPMagic = "AASP"
	// AASPVersion 私有格式版本号
	AASPVersion = uint32(1)
	// AASPHeaderSize 私有格式头部大小（92字节）
	AASPHeaderSize = 92
	// AASPKeyHashSize 密钥哈希大小（bcrypt哈希固定60字节）
	AASPKeyHashSize = 60
	// AASPKeyLength 音频密钥长度（16字节 = 128位）
	AASPKeyLength = 16
	// AESKeyLength AES密钥长度（32字节 = 256位）
	AESKeyLength = 32
)

// FileType 文件类型枚举
type FileType uint32

const (
	FileTypeAudio FileType = 1 // 音频
	FileTypeVideo FileType = 2 // 视频
	FileTypeImage FileType = 3 // 图片
	FileTypeCover FileType = 4 // 封面
)

// AASPHeader 私有格式文件头部结构（固定92字节）
type AASPHeader struct {
	Magic        [4]byte  // 魔数 "AASP" (4字节)
	Version      uint32   // 版本号 (4字节)
	KeyHash      [60]byte // bcrypt(key) 密钥校验 (60字节)
	OriginalSize uint64   // 原始文件大小 (8字节)
	FileType     uint32   // 文件类型：1音频 2视频 3图片 4封面 (4字节)
	Reserved     [12]byte // 保留字段 (12字节)
}

// GenerateAudioKey 生成16位随机音频密钥
func GenerateAudioKey() (string, error) {
	key := make([]byte, AASPKeyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("生成随机密钥失败: %v", err)
	}
	return fmt.Sprintf("%X", key), nil
}

// DeriveAESKey 从音频密钥派生AES-256密钥
func DeriveAESKey(audioKey string) ([]byte, error) {
	hash := sha256.Sum256([]byte(audioKey))
	return hash[:], nil
}

// EncryptFile 加密任意文件为私有格式
func EncryptFile(plainData []byte, audioKey string, fileType FileType) ([]byte, error) {
	if len(plainData) == 0 {
		return nil, fmt.Errorf("文件数据不能为空")
	}

	// 1. 生成密钥哈希（bcrypt）
	keyHash, err := bcrypt.GenerateFromPassword([]byte(audioKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密钥哈希失败: %v", err)
	}

	// 2. 派生AES密钥
	aesKey, err := DeriveAESKey(audioKey)
	if err != nil {
		return nil, fmt.Errorf("派生AES密钥失败: %v", err)
	}

	// 3. AES-256-GCM加密
	encryptedData, err := AESEncrypt(plainData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES加密失败: %v", err)
	}

	// 4. 构建私有格式文件
	var buf bytes.Buffer

	// 写入Header
	header := AASPHeader{
		Version:      AASPVersion,
		OriginalSize: uint64(len(plainData)),
		FileType:     uint32(fileType),
	}
	copy(header.Magic[:], AASPMagic)
	copy(header.KeyHash[:], keyHash)

	if err := binary.Write(&buf, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("写入头部失败: %v", err)
	}

	// 写入加密数据
	if _, err := buf.Write(encryptedData); err != nil {
		return nil, fmt.Errorf("写入加密数据失败: %v", err)
	}

	return buf.Bytes(), nil
}

// EncryptAudio 加密音频数据为私有格式（向后兼容）
func EncryptAudio(plainData []byte, audioKey string, duration uint32) ([]byte, error) {
	return EncryptFile(plainData, audioKey, FileTypeAudio)
}

// DecryptAudio 解密私有格式音频数据
func DecryptAudio(aaspData []byte, audioKey string) ([]byte, error) {
	if len(aaspData) < AASPHeaderSize {
		return nil, fmt.Errorf("私有格式数据不完整")
	}

	// 1. 解析Header
	var header AASPHeader
	reader := bytes.NewReader(aaspData[:AASPHeaderSize])
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("解析头部失败: %v", err)
	}

	// 2. 验证魔数
	if string(header.Magic[:]) != AASPMagic {
		return nil, fmt.Errorf("无效的私有格式文件")
	}

	// 3. 验证版本号
	if header.Version != AASPVersion {
		return nil, fmt.Errorf("不支持的私有格式版本: %d", header.Version)
	}

	// 4. 验证密钥
	if err := bcrypt.CompareHashAndPassword(header.KeyHash[:], []byte(audioKey)); err != nil {
		return nil, fmt.Errorf("密钥校验失败")
	}

	// 5. 提取加密数据
	encryptedData := aaspData[AASPHeaderSize:]

	// 6. 派生AES密钥并解密
	aesKey, err := DeriveAESKey(audioKey)
	if err != nil {
		return nil, fmt.Errorf("派生AES密钥失败: %v", err)
	}

	plainData, err := AESDecrypt(encryptedData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES解密失败: %v", err)
	}

	return plainData, nil
}

// AESEncrypt AES-256-GCM加密
func AESEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES密码块失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt AES-256-GCM解密
func AESDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES密码块失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	return plaintext, nil
}

// ParseAASPHeader 解析私有格式文件头部
func ParseAASPHeader(data []byte) (*AASPHeader, error) {
	if len(data) < AASPHeaderSize {
		return nil, fmt.Errorf("数据长度不足")
	}

	var header AASPHeader
	reader := bytes.NewReader(data[:AASPHeaderSize])
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("解析头部失败: %v", err)
	}

	if string(header.Magic[:]) != AASPMagic {
		return nil, fmt.Errorf("无效的私有格式文件")
	}

	return &header, nil
}

// GetAASPFileInfo 从私有格式数据中获取文件信息
func GetAASPFileInfo(data []byte) (*AASPHeader, error) {
	return ParseAASPHeader(data)
}
