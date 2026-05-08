# WebSocket 连接问题诊断与修复报告

## 🔍 问题诊断过程

### 核心问题 1: 路由配置错误 ✅ 已修复

**症状**: 404 错误
```
[HTTP] 404 - GET /api/v1/audio/separate/ws
```

**根本原因**: 
- WebSocket 路由被注册在**公开路由组**中，但公开路由组**没有** `/api/v1` 前缀
- 实际路径是 `/audio/separate/ws` 而不是 `/api/v1/audio/separate/ws`

**修复方案**:
```go
// 修改前（错误）
server.AddRoutes([]rest.Route{
    {
        Path: "/audio/separate/ws",  // 实际路径：/audio/separate/ws
        Handler: AudioSeparateWSHandler(serverCtx),
    },
})

// 修改后（正确）
// 将 WebSocket 路由移到 JWT 保护的路由组，自动添加 /api/v1 前缀
jwtProtected := []rest.Route{
    {
        Path: "/audio/separate/ws",  // 实际路径：/api/v1/audio/separate/ws
        Handler: AudioSeparateWSHandler(serverCtx),
    },
}
server.AddRoutes(
    rest.WithPrefix("/api/v1"),
    rest.WithMiddleware(auth.Middleware(...), jwtProtected...),
)
```

### 核心问题 2: 认证头支持不足 ✅ 已修复

**症状**: 用户从 Header 传递 token，但服务端只支持 URL 参数

**修复方案**:
```go
// 支持两种 token 传递方式
tokenString = r.URL.Query().Get("token")
if tokenString == "" {
    tokenString = r.Header.Get("Authorization")
}
```

### 核心问题 3: 缺少诊断日志 ✅ 已添加

**修复方案**: 添加详细的日志输出
```go
log.Printf("[WebSocket] 从 Header 获取 token: %s", tokenString)
log.Printf("[WebSocket] 尝试认证，token 前缀：%s...", tokenString[:min(20, len(tokenString))])
log.Printf("[WebSocket] 认证成功：user_id=%d", userID)
log.Printf("[WebSocket] 准备升级连接...")
log.Printf("[WebSocket] 连接升级成功")
```

### 核心问题 4: 跨域配置 ✅ 已确认

**检查**:
```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 允许跨域
    },
}
```
✅ 已正确配置，允许所有来源

### 核心问题 5: 异步处理导致连接过早关闭 ✅ 已修复

**症状**: WebSocket 连接在分离完成前就关闭，用户收不到结果

**修复方案**:
- 改为**同步处理**分离任务
- 保持 WebSocket 连接直到分离完成
- 通过 WebSocket 返回完整的分离结果

## 📋 修复后的完整流程

```
┌─────────┐                ┌──────────────┐                ┌─────────────┐
│  前端   │                │  WebSocket   │                │  AI 模型服务 │
│         │                │   Handler    │                │             │
└────┬────┘                └─────────────┘                ──────┬──────
     │                           │                               │
     │ 1. 建立 WebSocket 连接     │                               │
     │    (task_id + token)      │                               │
     ├──────────────────────────>│                               │
     │                           │                               │
     │                           │ ✓ 验证 token                  │
     │                           │ ✓ 记录详细日志                │
     │                           │                               │
     │ 2. 发送 content_id        │                               │
     │    {type, content_id}     │                               │
     ├──────────────────────────>│                               │
     │                           │                               │
     │                           │ 3. 查询音频 URL               │
     │                           │    SELECT audio_url           │
     │                           │    FROM audio_contents        │
     │                           │                               │
     │ 4. 回复：pending          │                               │
     │    "任务已接收"           │                               │
     │<──────────────────────────┤                               │
     │                           │                               │
     │                           │ 5. 创建任务记录               │
     │                           │    INSERT INTO                │
     │                           │    audio_separation_tasks     │
     │                           │                               │
     │                           │ 6. 调用 AI 模型分离           │
     │                           │    (同步执行，保持连接)       │
     │                           ├──────────────────────────────>│
     │                           │                               │
     │                           │ 7. 保存音轨结果               │
     │                           │    INSERT INTO audio_tracks   │
     │                           │                               │
     │ 8. 回复：completed        │                               │
     │    + 完整音轨列表         │                               │
     │<──────────────────────────┤                               │
     │                           │                               │
     │ (连接保持直到收到结果)    │                               │
     │                           │                               │
     └                           └                               └

```

##  测试步骤

### 步骤 1: 检查服务状态

```bash
# 检查端口监听
netstat -ano | findstr :8004

# 应该看到:
# TCP    0.0.0.0:8004           0.0.0.0:0              LISTENING       12345
```

### 步骤 2: 在 Apifox 中连接

**连接 URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**或者使用 Header**:
- URL: `ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001`
- Header: `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`

### 步骤 3: 发送分离请求

连接成功后，发送：
```json
{
  "type": "separate_request",
  "content_id": 1
}
```

### 步骤 4: 查看响应序列

**预期响应 1**: 任务已接收
```json
{
  "type": "separate_response",
  "task_id": "test_001",
  "status": "pending",
  "message": "任务已接收，正在查询音频信息",
  "content_id": 1
}
```

**预期响应 2**: 分离完成（带音轨列表）
```json
{
  "type": "separate_response",
  "task_id": "test_001",
  "status": "completed",
  "message": "分离完成",
  "content_id": 1,
  "tracks": [
    {
      "id": 1,
      "track_name": "vocals",
      "track_url": "/static/tracks/test_001_vocals.wav",
      "file_size": 5242880,
      "duration": 180.5,
      "sample_rate": 44100,
      "channels": 2,
      "format": "wav"
    },
    {
      "id": 2,
      "track_name": "drums",
      "track_url": "/static/tracks/test_001_drums.wav",
      "file_size": 3145728,
      "duration": 180.5,
      "sample_rate": 44100,
      "channels": 2,
      "format": "wav"
    },
    {
      "id": 3,
      "track_name": "bass",
      "track_url": "/static/tracks/test_001_bass.wav",
      "file_size": 2097152,
      "duration": 180.5,
      "sample_rate": 44100,
      "channels": 2,
      "format": "wav"
    },
    {
      "id": 4,
      "track_name": "other",
      "track_url": "/static/tracks/test_001_other.wav",
      "file_size": 1048576,
      "duration": 180.5,
      "sample_rate": 44100,
      "channels": 2,
      "format": "wav"
    }
  ]
}
```

### 步骤 5: 查看服务日志

**预期日志**:
```
[WebSocket] 从 Header 获取 token: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
[WebSocket] 尝试认证，token 前缀：eyJhbGciOiJIUzI1NiIs...
[WebSocket] 认证成功：user_id=1
[WebSocket] 准备升级连接...
[WebSocket] 连接升级成功
[WebSocket] 新连接：task_id=test_001, user_id=1, remote_addr=127.0.0.1:xxxxx
[WebSocket] 收到分离请求：task_id=test_001, content_id=1
[分离任务] 查询到音频 URL：task_id=test_001, content_id=1, url=http://...
[分离任务] AI 分离完成：task_id=test_001, tracks=4
[分离任务] 保存音轨：task_id=test_001, track=vocals, id=1
[分离任务] 保存音轨：task_id=test_001, track=drums, id=2
[分离任务] 保存音轨：task_id=test_001, track=bass, id=3
[分离任务] 保存音轨：task_id=test_001, track=other, id=4
[分离任务] 共保存 4 个音轨：task_id=test_001
[分离任务] 发送完成通知：task_id=test_001, tracks=4
```

## ⚠️ 常见错误及解决方案

### 错误 1: 404 Not Found

**原因**: 路径错误或服务未启动

**解决**:
1. 检查 URL 路径：`/api/v1/audio/separate/ws`
2. 确认服务已启动：`netstat -ano | findstr :8004`
3. 检查服务日志是否有启动错误

### 错误 2: 401 Unauthorized

**原因**: Token 无效或过期

**解决**:
1. 重新登录获取新 token
2. 检查 token 是否正确传递（URL 参数或 Header）
3. 查看服务日志中的认证信息

### 错误 3: 连接被拒绝

**原因**: 服务未监听 0.0.0.0 或防火墙拦截

**解决**:
1. 检查服务配置：`Host: 0.0.0.0`
2. 关闭防火墙或添加例外
3. 检查是否有其他程序占用 8004 端口

### 错误 4: 查询音频失败

**原因**: content_id 不存在

**解决**:
```sql
-- 检查 audio_contents 表
SELECT id, title, audio_url FROM audio_contents LIMIT 10;
```

### 错误 5: AI 分离失败

**原因**: AI 模型未正确初始化或音频文件不存在

**解决**:
1. 检查服务启动日志中的模型初始化信息
2. 确认音频文件可以访问
3. 查看详细的错误日志

## 📊 修复总结

### 修复的问题

| 问题 | 状态 | 修复方案 |
|------|------|----------|
| 路由配置错误 (404) | ✅ 已修复 | 将 WebSocket 路由移到 JWT 保护组 |
| Token 传递方式单一 | ✅ 已修复 | 支持 URL 参数和 Header 两种方式 |
| 缺少诊断日志 | ✅ 已添加 | 添加详细的认证和连接日志 |
| 跨域配置 | ✅ 已确认 | CheckOrigin 返回 true |
| 连接过早关闭 | ✅ 已修复 | 同步处理，等待分离完成 |
| 结果未返回 | ✅ 已实现 | 通过 WebSocket 返回完整音轨列表 |

### 改进的功能

1. **路由正确性**: WebSocket 路由现在正确注册在 `/api/v1/audio/separate/ws`
2. **认证灵活性**: 支持 URL 参数和 Header 两种 token 传递方式
3. **诊断能力**: 详细的日志输出，便于问题排查
4. **用户体验**: 保持连接直到分离完成，实时返回结果
5. **数据持久化**: 同时保存到数据库和返回给前端

### 核心代码变更

#### 1. 路由配置 ([`routes.go`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\internal\handler\routes.go))
```go
// 将 WebSocket 路由移到 JWT 保护组
jwtProtected := []rest.Route{
    {
        Path: "/audio/separate/ws",
        Handler: AudioSeparateWSHandler(serverCtx),
    },
}
```

#### 2. Token 认证 ([`audio_separate_ws_handler.go`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\internal\handler\audio_separate_ws_handler.go))
```go
// 支持两种 token 传递方式
tokenString = r.URL.Query().Get("token")
if tokenString == "" {
    tokenString = r.Header.Get("Authorization")
}
```

#### 3. 同步处理
```go
// 同步处理分离任务（保持连接等待结果）
s.processSeparation(req.ContentID)
// 不再提前关闭连接
```

#### 4. 返回完整结果
```go
// 通过 WebSocket 返回分离结果
s.sendMessage(SeparateResponse{
    Type:      "separate_response",
    Status:    "completed",
    Tracks:    trackInfos,  // 完整的音轨列表
})
```

## 🎯 现在可以测试了

服务已启动在 `http://localhost:8004`，按照上述测试步骤在 Apifox 中连接并测试！

**连接 URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=你的 JWT_TOKEN
```

**发送消息**:
```json
{
  "type": "separate_request",
  "content_id": 1
}
```

**预期结果**: 收到 `completed` 状态响应，包含完整的音轨列表 🎵
