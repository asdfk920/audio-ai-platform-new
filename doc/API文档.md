# API 文档

## 1. 通用说明

### 1.1 基础信息

- 协议: HTTPS
- 编码: UTF-8
- 数据格式: JSON
- API 版本: v1

### 1.2 请求头

```
Content-Type: application/json
Authorization: Bearer {access_token}  # 需要认证的接口
```

### 1.3 响应格式

#### 成功响应
```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

#### 错误响应
```json
{
  "code": 1001,
  "msg": "error message",
  "data": null
}
```

### 1.4 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 用户不存在 |
| 1002 | 密码错误 |
| 1003 | 用户已存在 |
| 1004 | Token 无效 |
| 1005 | Token 过期 |
| 2001 | 设备不存在 |
| 2002 | 设备已绑定 |
| 2003 | 设备离线 |
| 3001 | 内容不存在 |
| 3002 | 上传失败 |
| 3003 | 处理失败 |
| 9001 | 数据库错误 |
| 9002 | Redis 错误 |
| 9003 | 参数错误 |
| 9004 | 系统错误 |

## 2. 用户服务 API

### 2.1 用户注册

**接口地址**: `POST /api/v1/user/register`

**请求参数**:
```json
{
  "email": "user@example.com",
  "password": "password123",
  "mobile": "13800138000",
  "nickname": "用户昵称"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码（6-20位） |
| mobile | string | 否 | 手机号 |
| nickname | string | 否 | 昵称 |

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 1001
  }
}
```

### 2.2 用户登录

**接口地址**: `POST /api/v1/user/login`

**说明**：本接口为**账号密码**或**验证码**登录（请求体为 `LoginReq`：如 `account` / `email` / `mobile` 与 `password` 或 `verify_code`，需先按场景调用验证码发送接口）。**微信登录不使用本接口的 JSON 参数**，请使用 [§2.2.1 微信登录（OAuth2）](#221-微信登录oauth2)。

**请求参数**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 1001,
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

### 2.2.1 微信登录（OAuth2）

微信登录走标准 OAuth2 网页授权：**先 GET 发起授权（302 跳转微信）**，用户同意后**微信浏览器回调**用户服务，通过 **Query** 传 `code`（及 `state`），**无 JSON Body**。

| 步骤 | 方法与路径 | 传参方式 | 字段与说明 |
|------|------------|----------|------------|
| 发起授权 | `GET /api/v1/user/oauth/wechat` | 无 Body、无必填 Query | 客户端无需传业务字段；服务端 **302** 到微信授权页，授权 URL 中带 `state`（服务端写入 Redis） |
| 授权回调 | `GET /api/v1/user/oauth/wechat/callback` | **URL Query（form）** | **`code`**（必填）：微信返回的临时授权码；**`state`**：微信原样带回；若 `state` 非空，服务端会校验，失败则提示重新发起登录 |

**回调成功响应**（与密码登录一致，`OAuthLoginResp`）：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "user_id": 1001,
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

**配置**：用户服务需配置微信 `AppId`、`AppSecret`，且微信开放平台中的回调地址与部署一致（见 `services/user` 配置中的 OAuth 段及 `WeChatCallbackURL` 拼接逻辑）。

### 2.3 用户资料查询说明

用户微服务已**下线** `GET /api/v1/user/info`（避免对外暴露按 user_id 查资料的入口）。客户端请使用：

- 登录 / 刷新 token / 修改资料 / 绑定等接口返回的 `UserInfo` 字段作为展示数据；或  
- 管理后台：`GET /api/v1/platform-user/{userId}`（脱敏详情，需管理员 JWT）。

### 2.4 刷新 Token

**接口地址**: `POST /api/v1/user/refresh`

**请求参数**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

## 3. 设备服务 API

### 3.1 绑定设备

**接口地址**: `POST /api/v1/device/bind`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "device_sn": "SN123456789"
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "device_id": 2001
  }
}
```

### 3.2 获取设备列表

**接口地址**: `GET /api/v1/device/list`

**请求头**:
```
Authorization: Bearer {access_token}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "device_id": 2001,
        "device_sn": "SN123456789",
        "model": "Model-A",
        "firmware_version": "v1.0.0",
        "status": "online",
        "battery": 85,
        "volume": 50
      }
    ]
  }
}
```

### 3.3 设备心跳

**接口地址**: `POST /api/v1/device/heartbeat`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "device_id": 2001,
  "battery": 85,
  "volume": 50,
  "play_status": "playing"
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

### 3.4 获取播放列表

**接口地址**: `GET /api/v1/device/playlist?device_id=2001`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | int64 | 是 | 设备 ID |

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "content_id": 3001,
        "title": "音频标题",
        "url": "https://cdn.example.com/audio/xxx.mp3",
        "duration": 180
      }
    ]
  }
}
```

### 3.5 解绑设备

**接口地址**: `POST /api/v1/device/unbind`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "device_id": 2001
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

### 3.6 发送设备指令

**接口地址**: `POST /api/v1/device/command`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "device_id": 2001,
  "command_type": "play",
  "command_content": {
    "content_id": 3001
  }
}
```

**指令类型**:
- `play`: 播放
- `pause`: 暂停
- `volume`: 调节音量
- `reboot`: 重启

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "command_id": 4001
  }
}
```

## 4. 内容服务 API

### 4.1 获取上传地址

**接口地址**: `POST /api/v1/content/upload`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "file_name": "audio.wav",
  "file_size": 1024000
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "upload_url": "https://s3.amazonaws.com/bucket/raw/user_id/uuid.wav?signature=...",
    "raw_content_id": 5001
  }
}
```

**上传流程**:
1. 调用此接口获取预签名 URL
2. 使用 PUT 方法直接上传文件到 S3
3. 上传完成后，系统自动触发 AI 处理

### 4.2 获取内容列表

**接口地址**: `GET /api/v1/content/list?page=1&page_size=10`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int32 | 是 | 页码（从 1 开始） |
| page_size | int32 | 是 | 每页数量 |

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "content_id": 3001,
        "title": "音频标题",
        "duration": 180,
        "cdn_url": "https://cdn.example.com/audio/xxx.mp3",
        "status": "online",
        "created_at": "2024-03-11T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

### 4.3 获取处理状态

**接口地址**: `GET /api/v1/content/status?raw_content_id=5001`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| raw_content_id | int64 | 是 | 原始内容 ID |

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "raw_content_id": 5001,
    "status": "processing",
    "progress": 50
  }
}
```

**状态说明**:
- `uploaded`: 已上传
- `processing`: 处理中
- `ready`: 处理完成
- `failed`: 处理失败

### 4.4 更新内容信息

**接口地址**: `PUT /api/v1/content/{content_id}`

**请求头**:
```
Authorization: Bearer {access_token}
```

**请求参数**:
```json
{
  "title": "新标题",
  "status": "online"
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

### 4.5 删除内容

**接口地址**: `DELETE /api/v1/content/{content_id}`

**请求头**:
```
Authorization: Bearer {access_token}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

## 5. 管理后台 API

### 5.1 用户管理

#### 获取用户列表

**接口地址**: `GET /api/v1/admin/users?page=1&page_size=10`

**请求头**:
```
Authorization: Bearer {admin_token}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "user_id": 1001,
        "email": "user@example.com",
        "nickname": "用户昵称",
        "status": 1,
        "created_at": "2024-03-11T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

#### 禁用/启用用户

**接口地址**: `PUT /api/v1/admin/users/{user_id}/status`

**请求参数**:
```json
{
  "status": 0
}
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

### 5.2 设备管理

#### 获取设备列表

**接口地址**: `GET /api/v1/admin/devices?page=1&page_size=10`

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "device_id": 2001,
        "device_sn": "SN123456789",
        "model": "Model-A",
        "status": "online",
        "user_id": 1001,
        "created_at": "2024-03-11T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

### 5.3 内容审核

#### 获取待审核内容

**接口地址**: `GET /api/v1/admin/contents/pending?page=1&page_size=10`

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "content_id": 3001,
        "title": "音频标题",
        "user_id": 1001,
        "status": "pending",
        "created_at": "2024-03-11T10:00:00Z"
      }
    ],
    "total": 50
  }
}
```

#### 审核内容

**接口地址**: `POST /api/v1/admin/contents/{content_id}/review`

**请求参数**:
```json
{
  "action": "approve",
  "reason": "审核通过"
}
```

**action 说明**:
- `approve`: 通过
- `reject`: 拒绝

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

### 5.4 数据统计

#### 获取统计数据

**接口地址**: `GET /api/v1/admin/statistics`

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total_users": 10000,
    "total_devices": 5000,
    "total_contents": 20000,
    "online_devices": 3000,
    "today_uploads": 100,
    "today_plays": 5000
  }
}
```

## 6. 调用示例

### 6.1 cURL 示例

#### 用户注册
```bash
curl -X POST https://api.example.com/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### 用户登录
```bash
curl -X POST https://api.example.com/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### 更新用户昵称（返回完整 UserInfo）
```bash
curl -X PUT https://api.example.com/api/v1/user/info \
  -H "Authorization: Bearer {access_token}" \
  -H "Content-Type: application/json" \
  -d '{"user_id":1001,"nickname":"新昵称"}'
```

### 6.2 JavaScript 示例

```javascript
// 用户登录
async function login(email, password) {
  const response = await fetch('https://api.example.com/api/v1/user/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ email, password })
  });

  const data = await response.json();
  if (data.code === 0) {
    localStorage.setItem('access_token', data.data.access_token);
    return data.data;
  } else {
    throw new Error(data.msg);
  }
}

// 获取设备列表
async function getDevices() {
  const token = localStorage.getItem('access_token');
  const response = await fetch('https://api.example.com/api/v1/device/list', {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });

  const data = await response.json();
  return data.data.list;
}
```

### 6.3 Go 示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
    Data struct {
        UserId       int64  `json:"user_id"`
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresIn    int64  `json:"expires_in"`
    } `json:"data"`
}

func login(email, password string) (*LoginResponse, error) {
    req := LoginRequest{
        Email:    email,
        Password: password,
    }

    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "https://api.example.com/api/v1/user/login",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result LoginResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## 7. 限流说明

### 7.1 限流规则

- 未认证接口: 100 次/分钟/IP
- 已认证接口: 1000 次/分钟/用户
- 上传接口: 10 次/分钟/用户

### 7.2 限流响应

当触发限流时，返回 HTTP 429 状态码：

```json
{
  "code": 9005,
  "msg": "rate limit exceeded",
  "data": {
    "retry_after": 60
  }
}
```

## 8. Webhook 通知

### 8.1 内容处理完成通知

当 AI 处理完成后，系统会向配置的 Webhook URL 发送通知：

**请求方法**: `POST`

**请求体**:
```json
{
  "event": "content.processed",
  "timestamp": 1678512000,
  "data": {
    "raw_content_id": 5001,
    "content_id": 3001,
    "status": "ready",
    "cdn_url": "https://cdn.example.com/audio/xxx.mp3"
  }
}
```

### 8.2 设备离线通知

当设备超过 60 秒未上报心跳时：

**请求体**:
```json
{
  "event": "device.offline",
  "timestamp": 1678512000,
  "data": {
    "device_id": 2001,
    "device_sn": "SN123456789",
    "last_seen": 1678511940
  }
}
```
