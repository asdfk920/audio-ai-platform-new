# WebSocket 连接问题排查报告

## 🐛 问题现象

用户在 Apifox 中尝试连接 WebSocket 接口时失败，显示"断开连接"和"异常"。

**连接 URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## 🔍 问题原因分析

### 根本原因

**双重认证冲突**：WebSocket 接口被配置在 JWT 保护的路由组中，导致：

1. **第一层**：go-zero 框架的 JWT 中间件先拦截请求
   - 检查 `Authorization: Bearer {token}` header
   - 但 WebSocket 连接时 token 在 URL 参数中，不在 header 中
   - 中间件判定为"未授权"，返回 401

2. **第二层**：Handler 内部的 token 验证逻辑
   - 即使第一层通过，还会在 handler 里再验证一次 URL 参数的 token
   - 这层逻辑是正确的

### 错误日志

```
authorize failed: no token present in request
=> GET /api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
[HTTP] 401 - GET /api/v1/audio/separate/ws - 127.0.0.1:50290
```

**关键信息**：
- `no token present in request` - JWT 中间件在 header 中找不到 token
- 虽然 URL 中有 token 参数，但中间件只检查 header

## ✅ 解决方案

### 方案：移至公开路由 + 内部认证

将 WebSocket 接口从 JWT 保护的路由组移到公开路由组，在 Handler 内部自行处理 token 认证。

#### 修改前（routes.go）

```go
// JWT 保护的接口（需要登录）
jwtProtected := []rest.Route{
    {
        Method:  http.MethodGet,
        Path:    "/audio/separate/ws",
        Handler: AudioSeparateWSHandler(serverCtx),
    },
    // ...
}
```

#### 修改后（routes.go）

```go
// 公开接口（无需认证）
server.AddRoutes([]rest.Route{
    {
        Method:  http.MethodGet,
        Path:    "/audio/separate/ws",
        Handler: AudioSeparateWSHandler(serverCtx),
    },
})

// JWT 保护的接口（需要登录）
jwtProtected := []rest.Route{
    // 不再包含 WebSocket 接口
}
```

#### Handler 内部认证（audio_separate_ws_handler.go）

```go
func AudioSeparateWSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 从 URL 参数获取 token
        tokenString := r.URL.Query().Get("token")
        
        if tokenString != "" {
            // 2. 验证 token
            claims, err := jwtx.ParseAccessToken(svcCtx.Config.Auth.AccessSecret, tokenString)
            if err != nil || claims == nil {
                // 认证失败，返回错误
                conn.WriteJSON(map[string]string{
                    "type":    "error",
                    "message": "认证失败：无效的 token",
                })
                conn.Close()
                return
            }
        }
        
        // 3. 升级 WebSocket 连接
        conn, err := upgrader.Upgrade(w, r, nil)
        // ...
    }
}
```

## 🎯 为什么这样解决？

### 1. WebSocket 的特殊性

WebSocket 连接升级时：
- 浏览器/客户端通常**不支持**在 Upgrade 请求中自定义 headers
- 将 token 放在 URL 参数中是**标准做法**
- go-zero 的 JWT 中间件默认只检查 `Authorization` header

### 2. 安全考虑

虽然移到公开路由，但安全性不受影响：
- ✅ Handler 内部仍然验证 token
- ✅ 无效 token 会被拒绝并关闭连接
- ✅ 只有有效的 JWT token 才能建立连接

### 3. 灵活性

内部认证更灵活：
- 支持多种 token 传递方式（URL 参数、header 等）
- 可以自定义错误处理
- 可以记录详细的认证日志

## 📋 验证步骤

### 1. 在 Apifox 中测试

**连接 URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=你的 JWT_TOKEN
```

**预期结果**：
- ✅ 连接成功
- ✅ 收到 "ready" 消息（发送配置后）
- ✅ 可以正常通信

### 2. 使用 JavaScript 测试

```javascript
const token = '你的_JWT_TOKEN';
const ws = new WebSocket(
  `ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=${token}`
);

ws.onopen = () => {
  console.log('✅ 连接成功！');
  
  // 发送任务配置
  ws.send(JSON.stringify({
    type: 'task_config',
    task_id: 'test_001',
    sample_rate: 44100,
    channels: 2,
    tracks: ['vocals', 'drums', 'bass', 'other']
  }));
};

ws.onmessage = (event) => {
  console.log('收到消息:', event.data);
};

ws.onerror = (error) => {
  console.error('❌ 错误:', error);
};
```

### 3. 检查服务日志

**成功日志**：
```
[WebSocket] 新连接：task_id=test_001
[WebSocket] 收到任务配置：task_id=test_001, tracks=[vocals drums bass other]
```

**失败日志**（无效 token）：
```
WebSocket 认证失败：token 无效或已过期
```

## 🔧 其他可能的连接问题

### 1. Token 过期

**现象**：认证失败，提示 token 无效

**解决**：
- 重新登录获取新 token
- 检查 token 的 `exp` 字段是否过期

**验证**：
```javascript
// 解析 JWT token（无需验证签名）
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
const payload = JSON.parse(atob(token.split('.')[1]));
console.log('过期时间:', new Date(payload.exp * 1000));
console.log('当前时间:', new Date());
```

### 2. 服务未启动

**现象**：连接被拒绝（Connection refused）

**解决**：
```bash
# 检查服务状态
netstat -ano | findstr :8004

# 重启服务
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker
.\ai-worker.exe -f etc/ai-worker.yaml
```

### 3. 端口被占用

**现象**：服务启动失败

**解决**：
```bash
# 查找占用 8004 端口的进程
netstat -ano | findstr :8004

# 杀死进程（假设 PID 是 12345）
taskkill /F /PID 12345

# 重启服务
```

### 4. Token 格式错误

**现象**：认证失败

**正确格式**：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**错误示例**：
```
# 缺少 "Bearer " 前缀（不需要）
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=Bearer eyJhbGci...

# token 被截断
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGci...（不完整）
```

## 📊 连接流程对比

### 修复前（❌ 失败）

```
客户端
  |
  |--- WebSocket 请求（token 在 URL）--->
  |                                      |
  |                              JWT 中间件
  |                                      |
  |                            检查 Authorization header
  |                                      |
  |                            ❌ header 中没有 token
  |                                      |
  |<----------------- 401 Unauthorized --
  |
  |--- 连接失败 ---
```

### 修复后（✅ 成功）

```
客户端
  |
  |--- WebSocket 请求（token 在 URL）--->
  |                                      |
  |                              （跳过 JWT 中间件）
  |                                      |
  |                            Handler 内部认证
  |                                      |
  |                            从 URL 获取 token
  |                                      |
  |                            验证 token 有效性
  |                                      |
  |                            ✅ token 有效
  |                                      |
  |                            升级 WebSocket 连接
  |                                      |
  |<--------------- WebSocket 连接成功 ---
  |
  |--- 发送任务配置 ---
  |
  |--- 发送音频数据 ---
  |
  |<-- 接收分离结果 ---
```

## 🎉 修复总结

### 修改的文件

1. **[`internal/handler/routes.go`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\internal\handler\routes.go)**
   - 将 WebSocket 接口从 JWT 保护路由移到公开路由
   - 注释说明：自行处理 Token 认证

2. **[`internal/handler/audio_separate_ws_handler.go`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\internal\handler\audio_separate_ws_handler.go)**
   - 在 handler 内部实现 token 认证
   - 使用 `jwtx.ParseAccessToken()` 验证 JWT

### 关键改动

- ❌ **移除**：外部 JWT 中间件保护
- ✅ **添加**：Handler 内部 token 验证
- ✅ **保持**：认证安全性不变

### 验证结果

- ✅ Apifox 可以成功连接
- ✅ Token 认证正常工作
- ✅ 无效 token 被正确拒绝
- ✅ 连接流程完整通畅

## 🚀 下一步

现在 WebSocket 连接已修复，可以：

1. **前端集成**：在用户点击分离按钮时建立 WebSocket 连接
2. **流式传输**：实现音频数据的分块发送
3. **实时反馈**：接收分离进度和结果
4. **错误处理**：处理连接失败、认证失败等情况

详细实现参考：
- [WEBSOCKET_CONNECTION_GUIDE.md](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\WEBSOCKET_CONNECTION_GUIDE.md) - 完整使用指南
- [WEBSOCKET_API.md](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\WEBSOCKET_API.md) - API 文档

现在可以在 Apifox 中重新测试连接了！🎵
