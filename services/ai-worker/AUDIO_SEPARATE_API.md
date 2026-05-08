# 音频分离接口使用指南

## 📋 接口说明

### 新增接口：发起音频分离任务

**`POST /api/v1/audio/separate`**

- **功能**: 根据 content_id 查询数据库获取音频 URL，并发起 AI 分离任务
- **认证**: JWT Bearer Token（必需）
- **Handler**: `handler.AudioSeparateHandler`
- **Logic**: `logic.AudioSeparateLogic.SeparateAudio`

### 原有接口：上传音频文件

**`POST /api/v1/audio/upload`**

- **功能**: 用户上传本地音频文件
- **认证**: JWT Bearer Token（必需）

## 🏗️ 使用场景对比

### 场景 1：用户点击已有音频的分离按钮（使用新接口）

```
前端音频列表 → 用户点击"分离"按钮 → 传递 content_id → 后端查询数据库 → 发起分离
```

**请求示例：**
```json
POST /api/v1/audio/separate
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "content_id": 27
}
```

**响应示例：**
```json
{
  "task_id": "task_123456_27_1673875200000000000",
  "content_id": 27,
  "audio_url": "http://127.0.0.1:8000/audio/test1.mp3",
  "status": "pending",
  "message": "分离任务已创建，正在处理中",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 场景 2：用户上传本地音频文件（使用原接口）

```
前端选择本地文件 → 上传文件 → 后端保存并返回 audio_id → 后续可发起分离
```

**请求示例：**
```
POST /api/v1/audio/upload
Authorization: Bearer {jwt_token}
Content-Type: multipart/form-data

file: [音频文件]
```

## 🔐 认证流程

1. 用户登录获取 JWT Token
2. 请求时携带 `Authorization: Bearer {token}`
3. 后端验证 Token，提取用户 ID
4. 根据用户 ID 和 content_id 查询数据库

## 📤 测试方法

### 方法 1: 使用 cURL

```bash
# 设置 Token
TOKEN="your_jwt_token_here"

# 发起分离任务（content_id = 27）
curl -X POST "http://localhost:8004/api/v1/audio/separate" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"content_id": 27}'
```

### 方法 2: 使用 Apifox/Postman

**配置：**
- **方法**: POST
- **URL**: `http://localhost:8004/api/v1/audio/separate`
- **Headers**: 
  - `Authorization`: `Bearer {your_jwt_token}`
- **Body** (JSON):
  ```json
  {
    "content_id": 27
  }
  ```

### 方法 3: 前端调用示例

```javascript
// 用户点击分离按钮
async function handleSeparate(contentId) {
  const token = localStorage.getItem('jwt_token');
  
  try {
    const response = await fetch('http://localhost:8004/api/v1/audio/separate', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        content_id: contentId
      })
    });
    
    const result = await response.json();
    
    if (response.ok) {
      console.log('分离任务已创建:', result);
      // 保存 task_id 用于后续查询进度
      const taskId = result.task_id;
      // 可以轮询查询任务状态
      pollTaskStatus(taskId);
    } else {
      console.error('创建失败:', result);
      alert('创建分离任务失败：' + result.message);
    }
  } catch (error) {
    console.error('请求错误:', error);
    alert('网络错误，请稍后重试');
  }
}

// 轮询任务状态
async function pollTaskStatus(taskId) {
  const token = localStorage.getItem('jwt_token');
  
  const response = await fetch(`http://localhost:8004/api/v1/inference/query?task_id=${taskId}`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  
  const status = await response.json();
  console.log('任务状态:', status);
  
  // 如果未完成，继续轮询
  if (status.status !== 'completed') {
    setTimeout(() => pollTaskStatus(taskId), 2000);
  }
}
```

## 📊 数据库查询逻辑

后端会执行以下 SQL 查询：

```sql
SELECT audio_url 
FROM content 
WHERE id = ? AND is_deleted = 0 AND status = 1
```

**条件说明：**
- `id = ?`: 匹配传入的 content_id
- `is_deleted = 0`: 未删除的记录
- `status = 1`: 状态正常

如果查询失败，可能原因：
1. content_id 不存在
2. 音频已被删除（is_deleted = 1）
3. 音频状态异常（status ≠ 1）

## 🎯 完整流程示例

### 用户操作流程

1. **前端展示音频列表**
   ```javascript
   // 从 content 服务获取音频列表
   const audioList = await fetch('http://localhost:8000/api/v1/content/list');
   ```

2. **用户点击分离按钮**
   ```html
   <div v-for="audio in audioList" :key="audio.id">
     <h3>{{ audio.title }}</h3>
     <button @click="handleSeparate(audio.id)">分离</button>
   </div>
   ```

3. **发起分离请求**
   ```javascript
   async function handleSeparate(contentId) {
     const response = await fetch('http://localhost:8004/api/v1/audio/separate', {
       method: 'POST',
       headers: {
         'Authorization': `Bearer ${token}`,
         'Content-Type': 'application/json'
       },
       body: JSON.stringify({ content_id: contentId })
     });
     
     const { task_id, audio_url } = await response.json();
     
     // 保存任务 ID，用于后续查询
     localStorage.setItem('current_task_id', task_id);
     
     // 开始轮询任务状态
     startPolling(task_id);
   }
   ```

4. **查询任务进度**
   ```javascript
   async function startPolling(taskId) {
     const interval = setInterval(async () => {
       const response = await fetch(
         `http://localhost:8004/api/v1/inference/query?task_id=${taskId}`,
         { headers: { 'Authorization': `Bearer ${token}` } }
       );
       
       const result = await response.json();
       
       if (result.status === 'completed') {
         clearInterval(interval);
         // 任务完成，显示结果
         showResult(result);
       }
     }, 2000);
   }
   ```

## 🔧 配置说明

### 数据库配置

在 `etc/ai-worker.yaml` 中添加：

```yaml
Database:
  Host: localhost
  Port: 5432
  User: postgres
  Password: postgres
  DBName: audio_platform
```

### 默认值

如果配置文件未指定，使用以下默认值：
- Host: `localhost`
- Port: `5432`
- User: `postgres`
- Password: `postgres`
- DBName: `audio_platform`

## 📝 错误处理

### 常见错误响应

**1. content_id 无效**
```json
{
  "code": 400,
  "msg": "content_id 无效"
}
```

**2. 音频不存在**
```json
{
  "code": 400,
  "msg": "音频不存在或已删除"
}
```

**3. 数据库连接失败**
```json
{
  "code": 400,
  "msg": "连接数据库失败：..."
}
```

**4. 未授权访问**
```json
{
  "code": "TOKEN_INVALID",
  "msg": "未授权访问"
}
```

## 🚀 后续开发

接下来需要实现：

1. **查询任务状态接口** - `GET /api/v1/inference/query?task_id=xxx`
2. **获取分离结果接口** - `GET /api/v1/inference/result?task_id=xxx`
3. **实际调用 AI 模型** - 在 `SeparateAudio` 方法中集成 BSRoFormer

## 💡 实现建议

### 1. 任务状态管理

建议使用 Redis 存储任务状态：

```go
// 创建任务时
redis.Set(ctx, fmt.Sprintf("task:%s", taskID), json.Marshal(taskInfo), 5*time.Minute)

// 查询任务时
taskInfo := redis.Get(ctx, fmt.Sprintf("task:%s", taskID))
```

### 2. 异步处理

使用后台 goroutine 处理 AI 分离：

```go
go func() {
  // 下载音频文件
  audioPath := downloadAudio(audioURL)
  
  // 调用 AI 模型分离
  result := aiModel.Separate(audioPath)
  
  // 更新任务状态
  updateTaskStatus(taskID, "completed", result)
}()
```

### 3. 文件下载

从 content 服务下载音频：

```go
func downloadAudio(audioURL string) (string, error) {
  resp, err := http.Get(audioURL)
  if err != nil {
    return "", err
  }
  defer resp.Body.Close()
  
  // 保存到临时目录
  tmpFile := fmt.Sprintf("/tmp/audio_%s.mp3", uuid.New().String())
  out, err := os.Create(tmpFile)
  if err != nil {
    return "", err
  }
  defer out.Close()
  
  _, err = io.Copy(out, resp.Body)
  return tmpFile, err
}
```

## 🔗 相关接口

- **上传音频**: `POST /api/v1/audio/upload`
- **查询任务**: `GET /api/v1/inference/query` (待实现)
- **获取结果**: `GET /api/v1/inference/result` (待实现)
- **健康检查**: `GET /api/v1/health`
