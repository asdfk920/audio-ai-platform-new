package memberpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateCallbackSign 生成支付回调签名
// 用于测试环境模拟支付平台生成签名
// 参数：
//   - orderNo: 平台订单号
//   - tradeNo: 第三方支付平台交易号
//   - secret: 回调验签密钥（配置文件中 MockCallbackSecret）
//
// 返回：
//   - sign: HMAC-SHA256 签名（16 进制字符串）
func GenerateCallbackSign(orderNo, tradeNo, secret string) string {
	if secret == "" {
		secret = "default_mock_secret_key_2024"
	}

	// 构造待签名字符串：order_no|trade_no
	signContent := fmt.Sprintf("%s|%s", orderNo, tradeNo)

	// 使用 HMAC-SHA256 计算签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signContent))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyCallbackSign 验证支付回调签名
// 用于手动验证签名是否合法
// 参数：
//   - orderNo: 平台订单号
//   - tradeNo: 第三方支付平台交易号
//   - sign: 待验证的签名
//   - secret: 回调验签密钥
//
// 返回：
//   - valid: 签名是否有效
func VerifyCallbackSign(orderNo, tradeNo, sign, secret string) bool {
	expectedSign := GenerateCallbackSign(orderNo, tradeNo, secret)
	return hmac.Equal([]byte(expectedSign), []byte(sign))
}
