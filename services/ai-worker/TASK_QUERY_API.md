# 查询分离任务接口文档

## 📋 接口说明

**`GET /api/v1/inference/query`** - 查询音频分离任务状态

- **功能**: 查询用户发起的音频分离任务的状态和进度
- **认证**: JWT Bearer Token（必需）
- **权限**: 用户只能查询自己的任务
- **Handler**: `handler.TaskQueryHandler`
- **Logic**: `logic.TaskQueryLogic.QueryTask`

## 🔐 安全机制

### 用户隔离
- 查询时会验证 `task_id` 和 `user_id`（从 JWT Token 中提取）
- 即使用户知道其他用户的 task_id，也无法查询到
- 数据库查询条件：`WHERE task_id = ? AND user_id = ?`

## 📤 测试方法

### 方法 1: 使用 cURL

```bash
# 设置 Token
TOKEN="your_jwt_token_here"
TASK_ID="task_123456_27_1673875200000000000"

# 查询任务状态
curl -G "http://localhost:8004/api/v1/inference/query" \
  -H "Authorization: Bearer ${TOKEN}" \
  --data-urlencode "task_id=${TASK_ID}"
```

### 方法 2: 使用 Apifox/Postman

**配置：**
- **方法**: GET
- **URL**: `http://localhost:8004/api/v1/inference/query?task_id={task_id}`
- **Headers**: 
  - `Authorization`: `Bearer {your_jwt_token}`
- **Params**:
  - `task_id`: 任务 ID（必填）

### 方法 3: 前端调用示例

```javascript
// 查询任务状态
async function queryTaskStatus(taskId) {
  const token = localStorage.getItem('jwt_token');
  
  try {
    const response = await fetch(
      `http://localhost:8004/api/v1/inference/query?task_id=${taskId}`,
      {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      }
    );
    
    const result = await response.json();
    
    if (response.ok) {
      console.log('任务状态:', result);
      // 根据状态更新 UI
      updateTaskUI(result);
    } else {
      console.error('查询失败:', result);
      alert('查询失败：' + result.message);
    }
  } catch (error) {
    console.error('请求错误:', error);
    alert('网络错误，请稍后重试');
  }
}

// 轮询任务状态（每 2 秒查询一次）
function startPolling(taskId) {
  const interval = setInterval(async () => {
    const task = await queryTaskStatus(taskId);
    
    if (task.status === 'completed' || task.status === 'failed') {
      clearInterval(interval);
      // 任务完成或失败，停止轮询
      handleTaskFinished(task);
    } else {
      // 更新进度条
      updateProgress(task.progress);
    }
  }, 2000);
}
```

## 📊 响应示例

### 成功响应（任务进行中）

```json
{
  "task_id": "task_123456_27_1673875200000000000",
  "user_id": 123456,
  "content_id": 27,
  "audio_url": "http://127.0.0.1:8000/audio/test1.mp3",
  "status": "processing",
  "progress": 45,
  "message": "正在分离音频...",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:15Z"
}
```

### 成功响应（任务完成）

```json
{
  "task_id": "task_123456_27_1673875200000000000",
  "user_id": 123456,
  "content_id": 27,
  "audio_url": "http://127.0.0.1:8000/audio/test1.mp3",
  "status": "completed",
  "progress": 100,
  "message": "分离完成",
  "result_url": "http://localhost:8004/static-media/results/123456/task_123456_27.zip",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:31:00Z"
}
```

### 成功响应（任务失败）

```json
{
  "task_id": "task_123456_27_1673875200000000000",
  "user_id": 123456,
  "content_id": 27,
  "audio_url": "http://127.0.0.1:8000/audio/test1.mp3",
  "status": "failed",
  "progress": 30,
  "message": "音频文件下载失败",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:20Z"
}
```

### 错误响应（任务不存在）

```json
{
  "code": 400,
  "msg": "任务不存在或无权查看"
}
```

### 错误响应（未授权）

```json
{
  "code": "TOKEN_INVALID",
  "msg": "未授权访问"
}
```

## 📝 任务状态说明

| 状态 | 说明 | 进度 |
|------|------|------|
| `pending` | 等待中 | 0 |
| `processing` | 处理中 | 1-99 |
| `completed` | 已完成 | 100 |
| `failed` | 失败 | 0-99 |

## 🔄 完整使用流程

### 1. 发起分离任务

```javascript
// POST /api/v1/audio/separate
const response = await fetch('http://localhost:8004/api/v1/audio/separate', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ content_id: 27 })
});

const { task_id } = await response.json();
// 保存 task_id 用于后续查询
localStorage.setItem('current_task_id', task_id);
```

### 2. 轮询查询进度

```javascript
// 每 2 秒查询一次
const interval = setInterval(async () => {
  const task = await queryTaskStatus(task_id);
  
  // 更新进度条
  document.getElementById('progress').value = task.progress;
  
  if (task.status === 'completed') {
    clearInterval(interval);
    // 显示下载按钮
    showDownloadButton(task.result_url);
  } else if (task.status === 'failed') {
    clearInterval(interval);
    // 显示错误信息
    showError(task.message);
  }
}, 2000);
```

### 3. 下载分离结果

```javascript
// 任务完成后，下载结果
function downloadResult(resultUrl) {
  const a = document.createElement('a');
  a.href = resultUrl;
  a.download = 'separated_audio.zip';
  a.click();
}
```

## 🗄️ 数据库表结构

### audio_separation_tasks 表

```sql
CREATE TABLE audio_separation_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(100) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,          -- 用户 ID（任务所有者）
    content_id BIGINT NOT NULL,        -- 内容 ID
    audio_url VARCHAR(2000) NOT NULL,  -- 原始音频 URL
    status VARCHAR(20) NOT NULL,       -- pending/processing/completed/failed
    progress INTEGER NOT NULL DEFAULT 0, -- 0-100
    message TEXT,                      -- 任务消息
    result_url VARCHAR(2000),          -- 分离结果 URL
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### 索引

- `idx_audio_tasks_task_id` - 加速 task_id 查询
- `idx_audio_tasks_user_id` - 加速用户任务查询
- `idx_audio_tasks_status` - 加速状态筛选
- `idx_audio_tasks_created_at` - 加速时间排序

## 🔒 安全验证

### 1. JWT Token 验证

```go
// 中间件验证 Token
auth := r.Header.Get("Authorization")
claims, err := jwtx.ParseAccessToken(secret, token)
if err != nil {
    return "未授权访问"
}
userID := claims.UserID
```

### 2. 用户权限验证

```go
// 查询时强制匹配 user_id
query := `
    SELECT * FROM audio_separation_tasks
    WHERE task_id = $1 AND user_id = $2
`
// 即使用户 A 知道用户 B 的 task_id，也无法查询到
```

## ⚠️ 注意事项

1. **必须携带 JWT Token**
   - 没有 Token 会返回 401 未授权
   
2. **只能查询自己的任务**
   - 查询时会验证 `task_id` 和 `user_id`
   - 尝试查询他人任务会返回"任务不存在或无权查看"

3. **轮询间隔建议**
   - 建议 2-3 秒查询一次
   - 避免过于频繁（会增加服务器压力）

4. **任务超时处理**
   - 建议前端设置超时时间（如 5 分钟）
   - 超时后提示用户"任务处理时间过长，请稍后查询"

## 🚀 后续优化

### 1. WebSocket 实时推送

替代轮询，使用 WebSocket 实时推送进度：

```javascript
const ws = new WebSocket('ws://localhost:8004/ws/tasks/' + taskId);
ws.onmessage = (event) => {
  const task = JSON.parse(event.data);
  updateProgress(task.progress);
};
```

### 2. 批量查询

一次性查询多个任务状态：

```javascript
POST /api/v1/inference/query/batch
{
  "task_ids": ["task_1", "task_2", "task_3"]
}
```

### 3. 任务列表

查询用户的所有任务（分页）：

```javascript
GET /api/v1/inference/tasks?page=1&page_size=20
```

## 🔗 相关接口

- **发起分离**: `POST /api/v1/audio/separate`
- **查询进度**: `GET /api/v1/inference/query`
- **健康检查**: `GET /api/v1/health`
