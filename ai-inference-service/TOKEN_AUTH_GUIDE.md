# 音轨分离服务 - Token 认证使用说明

## 概述

服务已更新为使用 JWT Token 进行用户认证，不再需要在请求体中传递 `user_id`。

## 认证流程

### 1. 获取 Token

**接口**: `POST /api/v1/auth/generate-token`

**参数**:
- `user_id`: 用户 ID（必填）
- `device_id`: 设备 ID（可选）

**请求示例**:
```bash
POST http://localhost:8004/api/v1/auth/generate-token?user_id=user_123
```

**响应示例**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcl8xMjMiLCJkZXZpY2VfaWQiOiJkZXZpY2VfNDU2IiwiZXhwIjoxNzE1MTg0MDAwfQ.xxx",
  "expires_in": 86400,
  "token_type": "Bearer"
}
```

### 2. 使用 Token 调用音轨分离接口

**接口**: `POST /api/v1/separate/start`

**Header**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Body** (不再需要 user_id):
```json
{
  "audio_url": "https://example.com/audio/song.mp3",
  "device_id": "device_456"
}
```

**响应示例**:
```json
{
  "success": true,
  "task_id": "task_20260507185554_xxxxx",
  "message": "任务创建成功，正在处理",
  "data": {
    "task_id": "task_20260507185554_xxxxx",
    "user_id": "user_123",  // 从 token 中自动提取
    "device_id": "device_456",
    "audio_url": "https://example.com/audio/song.mp3",
    "status": "pending",
    ...
  }
}
```

## Apifox 配置步骤

### 步骤 1: 生成 Token

1. 创建一个新的请求
2. 方法：POST
3. URL: `http://localhost:8004/api/v1/auth/generate-token`
4. 在 Params 中添加：
   - `user_id`: `user_123`
   - `device_id`: `device_456` (可选)
5. 发送请求，复制返回的 token

### 步骤 2: 配置音轨分离接口

1. 创建新的请求
2. 方法：POST
3. URL: `http://localhost:8004/api/v1/separate/start`
4. 在 **Header** 中添加：
   - Key: `Authorization`
   - Value: `Bearer <复制的 token>`
5. 在 **Body** 中选择 `raw` -> `JSON`，输入：
```json
{
  "audio_url": "https://example.com/audio/song.mp3",
  "device_id": "device_456"
}
```
6. 发送请求

## 错误处理

### 401 - 缺少认证信息
```json
{
  "detail": "缺少认证信息，请在 Header 中添加 Authorization: Bearer <token>"
}
```

### 401 - Token 无效
```json
{
  "detail": "Token 无效"
}
```

### 401 - Token 过期
```json
{
  "detail": "Token 已过期"
}
```

## 代码示例

### Python
```python
import requests

# 1. 获取 token
auth_response = requests.post(
    "http://localhost:8004/api/v1/auth/generate-token",
    params={"user_id": "user_123"}
)
token = auth_response.json()["token"]

# 2. 调用音轨分离接口
headers = {
    "Authorization": f"Bearer {token}",
    "Content-Type": "application/json"
}

payload = {
    "audio_url": "https://example.com/audio/song.mp3",
    "device_id": "device_456"
}

response = requests.post(
    "http://localhost:8004/api/v1/separate/start",
    json=payload,
    headers=headers
)

task_id = response.json()["task_id"]
print(f"任务 ID: {task_id}")
```

### JavaScript
```javascript
// 1. 获取 token
const authResponse = await fetch(
    'http://localhost:8004/api/v1/auth/generate-token?user_id=user_123'
);
const authData = await authResponse.json();
const token = authData.token;

// 2. 调用音轨分离接口
const response = await fetch(
    'http://localhost:8004/api/v1/separate/start',
    {
        method: 'POST',
        headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            audio_url: 'https://example.com/audio/song.mp3',
            device_id: 'device_456'
        })
    }
);

const result = await response.json();
console.log(`任务 ID: ${result.task_id}`);
```

## 安全建议

1. **生产环境**: 使用强密钥（环境变量）
   ```python
   JWT_SECRET_KEY = os.getenv("JWT_SECRET_KEY", "strong-random-secret")
   ```

2. **Token 过期时间**: 根据业务需求调整
   ```python
   expire = datetime.utcnow() + timedelta(hours=1)  # 1 小时
   ```

3. **HTTPS**: 生产环境使用 HTTPS 传输

4. **Token 刷新**: 实现 token 刷新机制

## 测试

运行测试脚本验证认证功能：
```bash
python test_token_auth.py
```

测试内容包括：
- ✓ 生成 Token
- ✓ 使用 Token 调用 API
- ✓ 无 Token 时的拒绝
- ✓ 无效 Token 的拒绝

## 变更说明

### 之前（已废弃）
```json
POST /api/v1/separate/start
{
  "audio_url": "...",
  "user_id": "user_123",  // ❌ 不再需要
  "device_id": "device_456"
}
```

### 现在（推荐）
```
Header: Authorization: Bearer <token>
```
```json
POST /api/v1/separate/start
{
  "audio_url": "...",
  "device_id": "device_456"  // user_id 从 token 中提取
}
```

## 常见问题

**Q: Token 有效期多久？**  
A: 默认 24 小时，可以在生成 token 时调整。

**Q: 如何刷新 Token？**  
A: Token 过期后，重新调用 `/api/v1/auth/generate-token` 接口获取新的 token。

**Q: 可以在多个设备上使用同一个 Token 吗？**  
A: 可以，token 与 user_id 绑定，不限制设备。

**Q: 如何撤销 Token？**  
A: 当前版本不支持主动撤销，Token 会在过期后自动失效。
