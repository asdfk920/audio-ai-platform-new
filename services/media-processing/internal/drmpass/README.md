# drmpass（流媒体代理 DRM 头透传）

## 合规边界（实现侧）

- 本包**不涉及**解密、转码密钥、密钥派生或对音频载荷的语义解析。
- **上游 HTTP 响应**中前缀为 `x-drm-`（不区分大小写）的头，会**原样**复制到下发响应（本平台追加的头除外）。
- **`X-DRM-Platform-*`**：在配置 RSA 私钥且 `DRMPass.AttestSigning=true` 时写入，内容为对 DRM 快照的 **RSA-PSS + SHA256** 签名（设备可用公钥验签，参见 `VerifyPlatformSignature`）。

## HTTP 字段（与文档 JSON 对齐）

| 响应头 | 含义 |
|--------|------|
| `X-DRM-Type` | `widevine` / `fairplay` / `playready` / … |
| `X-DRM-Content-Id` | 内容 ID |
| `X-DRM-Key-Id` | Base64 KID |
| `X-DRM-Pssh` | Base64 PSSH |
| `X-DRM-License-Url` | 许可证服务器 |
| `X-DRM-Auth-Token` | 短期授权令牌（敏感，勿打满日志） |
| `X-DRM-Copyright` | 版权声明 |
| `X-DRM-Usage-Policy` | 策略文本 |
| `X-DRM-Signature` | 上游侧签名（如有） |
| `X-DRM-Platform-Signature` | 代理对快照的 RSA-PSS 签名（可选） |
| `X-DRM-Platform-Issuer` | 签发方标识 |
| `X-DRM-Proxy-Timestamp` | RFC3339Nano，纳入签名校验 |

生成测试密钥：`openssl genrsa -out drm_proxy.pem 2048`  

提取公钥给设备：`openssl rsa -in drm_proxy.pem -pubout -out drm_proxy_pub.pem`

## 配置（`etc/media-processing.yaml`）

参见 `DRMPass`：`AttestSigning`、`Issuer`、`SigningKeyPath` / `SigningKeyPEM`。
