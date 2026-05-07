# 音轨分离服务 - 用户服务 Token 认证说明

## 概述

AI 推理服务现在配置为验证**用户服务**生成的 JWT token，不再自己生成 token。

## 认证流程

### 1. 从用户服务获取 Token

用户需要先登录用户服务，获取 JWT token。

**用户登录接口**:
```
POST http://localhost:8080/api/v1/user/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "userId": 123,
    "username": "your_username"
  }
}
```

### 2. 使用 Token 调用音轨分离接口

**接口**: `POST /api/v1/separate/start`

**Header**:
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

**Body**:
```json
{
  "audio_url": "https://example.com/audio/song.mp3",
  "device_id": "device_456"
}
```

**说明**:
- ❌ 不需要传 `user_id` - 会自动从 token 中提取
- ✅ `userId` 字段会从用户服务的 token 中自动解析

## 配置说明

### JWT 密钥配置

AI 推理服务使用与用户服务相同的密钥来验证 token：

```python
# 从 services/user/etc/user-api.yaml
JWT_SECRET_KEY = "audio-ai-platform-secret-key-2024"
JWT_ALGORITHM = "HS256"
```

### Token 字段支持

服务支持以下 token 字段：
- `userId` - 用户服务的标准字段
- `user_id` - 兼容字段

## Apifox 使用步骤

### 步骤 1: 登录获取 Token

1. 打开用户登录接口
2. 输入用户名和密码
3. 发送请求
4. 复制响应中的 `token` 值

### 步骤 2: 调用音轨分离接口

1. 打开音轨分离接口
2. 在 **Header** 标签页添加：
   - Key: `Authorization`
   - Value: `Bearer <复制的 token>`
3. 在 **Body** 标签页输入：
   ```json
   {
     "audio_url": "https://example.com/audio/song.mp3",
     "device_id": "device_456"
   }
   ```
4. 发送请求

## 错误处理

### 401 - Token 无效

```json
{
  "detail": "Token 无效 (InvalidSignatureError)。请确保使用用户服务生成的 token。错误：Signature verification failed"
}
```

**原因**: Token 不是由用户服务生成的，或者密钥不匹配

**解决**: 
1. 确保使用用户服务登录接口获取的 token
2. 检查 token 是否完整复制

### 401 - Token 过期

```json
{
  "detail": "Token 已过期"
}
```

**解决**: 重新登录获取新的 token

### 401 - 缺少用户 ID 字段

```json
{
  "detail": "Token 无效，缺少用户 ID 字段 (userId 或 user_id)"
}
```

**原因**: Token 中没有包含用户 ID 信息

**解决**: 确保使用正确的用户服务生成的 token

## 技术实现

### Token 验证逻辑

```python
def verify_token(authorization: Optional[str] = Header(None)):
    # 1. 从 Header 提取 Bearer token
    scheme, token = authorization.split()
    
    # 2. 使用用户服务的密钥验证
    payload = jwt.decode(token, JWT_SECRET_KEY, algorithms=[JWT_ALGORITHM])
    
    # 3. 提取用户 ID（支持 userId 和 user_id）
    user_id = payload.get("userId") or payload.get("user_id")
    
    # 4. 返回用户 ID 供业务逻辑使用
    return str(user_id)
```

### 密钥配置

```python
# 使用用户服务的密钥
JWT_SECRET_KEY = "audio-ai-platform-secret-key-2024"
```

## 测试

### 使用真实用户 Token 测试

1. 登录用户服务获取 token
2. 在音轨分离接口中使用该 token
3. 验证是否可以成功创建任务

### 预期结果

- ✅ Token 验证成功
- ✅ 用户 ID 正确提取
- ✅ 任务创建成功
- ✅ 响应中包含正确的 user_id

## 常见问题

**Q: 为什么 Token 验证失败？**  
A: 最可能的原因是使用了错误的 token。请确保：
- 使用用户服务登录接口获取的 token
- Token 完整复制，没有截断
- Token 没有过期

**Q: 如何查看 Token 内容？**  
A: 可以使用 [jwt.io](https://jwt.io) 网站解码 token，查看其中的字段。

**Q: Token 有效期多久？**  
A: 由用户服务配置决定，通常是几小时到几天不等。

**Q: 可以在多个微服务中使用同一个 Token 吗？**  
A: 可以，只要微服务使用相同的密钥配置。

## 架构说明

```
┌─────────────┐
│   用户端     │
└────────────┘
       │ 1. 登录
       ▼
┌─────────────┐
│  用户服务    │─── 生成 Token (密钥：audio-ai-platform-secret-key-2024)
└──────┬──────┘
       │ 2. 返回 Token
       ▼
┌─────────────┐
│   用户端     │
└──────┬──────┘
       │ 3. 调用 API (携带 Token)
       ▼
┌─────────────┐
│ AI 推理服务   │─── 验证 Token (使用相同密钥)
└─────────────┘
```

## 更新日志

- **v2.1.0**: 改为验证用户服务生成的 token
  - 使用用户服务的 JWT 密钥
  - 支持 `userId` 和 `user_id` 字段
  - 删除本地生成 token 接口
  - 改进错误提示信息
