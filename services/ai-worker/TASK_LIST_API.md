# 查询分离任务列表接口文档

## 📋 接口说明

**`GET /api/v1/inference/tasks`** - 查询用户的音频分离任务列表

- **功能**: 查询用户发起的所有音频分离任务，支持分页和状态筛选
- **认证**: JWT Bearer Token（必需）
- **权限**: 用户只能查询自己的任务
- **Handler**: `handler.TaskListHandler`
- **Logic**: `logic.TaskListLogic.GetTaskList`

## 🔐 安全机制

### 用户隔离
- 查询时只返回 `user_id` 匹配当前用户的任务
- 数据库查询条件：`WHERE user_id = ?`
- 用户无法查看其他用户的任务

## 📤 请求参数

### Query 参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | int | 否 | 1 | 页码（从 1 开始） |
| page_size | int | 否 | 20 | 每页数量（最大 100） |
| status | string | 否 | - | 状态筛选：pending/processing/completed/failed |

### 请求示例

#### 1. 获取第一页（默认 20 条）

```
GET /api/v1/inference/tasks?page=1&page_size=20
```

#### 2. 筛选已完成的任务

```
GET /api/v1/inference/tasks?status=completed
```

#### 3. 获取第二页，每页 10 条

```
GET /api/v1/inference/tasks?page=2&page_size=10
```

#### 4. 筛选处理中的任务

```
GET /api/v1/inference/tasks?status=processing
```

## 📊 响应示例

### 成功响应

```json
{
  "tasks": [
    {
      "task_id": "task_123456_27_1673875200000000000",
      "content_id": 27,
      "audio_url": "http://127.0.0.1:8000/audio/test1.mp3",
      "status": "completed",
      "progress": 100,
      "message": "分离完成",
      "result_url": "http://localhost:8004/static-media/results/123456/task_123456_27.zip",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:31:00Z"
    },
    {
      "task_id": "task_123456_28_1673875100000000000",
      "content_id": 28,
      "audio_url": "http://127.0.0.1:8000/audio/test2.mp3",
      "status": "processing",
      "progress": 45,
      "message": "正在分离音频...",
      "result_url": "",
      "created_at": "2024-01-15T10:25:00Z",
      "updated_at": "2024-01-15T10:26:00Z"
    },
    {
      "task_id": "task_123456_26_1673875000000000000",
      "content_id": 26,
      "audio_url": "http://127.0.0.1:8000/audio/test3.mp3",
      "status": "pending",
      "progress": 0,
      "message": "任务已创建",
      "result_url": "",
      "created_at": "2024-01-15T10:20:00Z",
      "updated_at": "2024-01-15T10:20:00Z"
    }
  ],
  "total": 15,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

### 响应字段说明

#### 顶层字段

| 字段 | 类型 | 说明 |
|------|------|------|
| tasks | array | 任务列表 |
| total | int64 | 总记录数 |
| page | int | 当前页码 |
| page_size | int | 每页数量 |
| total_pages | int | 总页数 |

#### 任务对象字段

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务唯一标识 |
| content_id | int64 | 内容 ID |
| audio_url | string | 原始音频 URL |
| status | string | 状态：pending/processing/completed/failed |
| progress | int | 进度：0-100 |
| message | string | 任务消息 |
| result_url | string | 分离结果 URL（完成后才有） |
| created_at | string | 创建时间（RFC3339 格式） |
| updated_at | string | 更新时间（RFC3339 格式） |

### 错误响应

#### 未授权

```json
{
  "code": "TOKEN_INVALID",
  "msg": "未授权访问"
}
```

#### 服务器错误

```json
{
  "code": 500,
  "msg": "查询任务列表失败：数据库连接失败"
}
```

##  测试方法

### 方法 1: 使用 cURL

```bash
# 设置 Token
TOKEN="your_jwt_token_here"

# 获取所有任务（第一页）
curl -G "http://localhost:8004/api/v1/inference/tasks" \
  -H "Authorization: Bearer ${TOKEN}" \
  --data-urlencode "page=1" \
  --data-urlencode "page_size=20"

# 筛选已完成的任务
curl -G "http://localhost:8004/api/v1/inference/tasks" \
  -H "Authorization: Bearer ${TOKEN}" \
  --data-urlencode "status=completed"

# 获取第二页
curl -G "http://localhost:8004/api/v1/inference/tasks" \
  -H "Authorization: Bearer ${TOKEN}" \
  --data-urlencode "page=2" \
  --data-urlencode "page_size=10"
```

### 方法 2: 使用 Apifox/Postman

**配置：**
- **方法**: GET
- **URL**: `http://localhost:8004/api/v1/inference/tasks`
- **Headers**: 
  - `Authorization`: `Bearer {your_jwt_token}`
- **Params**（可选）:
  - `page`: 1
  - `page_size`: 20
  - `status`: completed/processing/pending/failed

### 方法 3: 前端调用示例

```javascript
// 获取任务列表
async function getTaskList(page = 1, pageSize = 20, status = '') {
  const token = localStorage.getItem('jwt_token');
  
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: pageSize.toString()
  });
  
  if (status) {
    params.append('status', status);
  }
  
  try {
    const response = await fetch(
      `http://localhost:8004/api/v1/inference/tasks?${params}`,
      {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      }
    );
    
    const result = await response.json();
    
    if (response.ok) {
      console.log('任务列表:', result);
      // 渲染任务列表
      renderTaskList(result);
      return result;
    } else {
      console.error('查询失败:', result);
      alert('查询失败：' + result.message);
    }
  } catch (error) {
    console.error('请求错误:', error);
    alert('网络错误，请稍后重试');
  }
}

// 渲染任务列表
function renderTaskList(data) {
  const container = document.getElementById('task-list');
  container.innerHTML = '';
  
  data.tasks.forEach(task => {
    const taskElement = document.createElement('div');
    taskElement.className = 'task-item';
    taskElement.innerHTML = `
      <div class="task-header">
        <span class="task-id">${task.task_id}</span>
        <span class="task-status status-${task.status}">${task.status}</span>
      </div>
      <div class="task-progress">
        <progress value="${task.progress}" max="100"></progress>
        <span>${task.progress}%</span>
      </div>
      <div class="task-message">${task.message}</div>
      <div class="task-time">
        <small>创建时间：${new Date(task.created_at).toLocaleString()}</small>
      </div>
      ${task.result_url ? `
        <a href="${task.result_url}" class="download-btn" download>下载结果</a>
      ` : ''}
    `;
    container.appendChild(taskElement);
  });
  
  // 渲染分页
  renderPagination(data);
}

// 渲染分页
function renderPagination(data) {
  const pagination = document.getElementById('pagination');
  pagination.innerHTML = '';
  
  for (let i = 1; i <= data.total_pages; i++) {
    const button = document.createElement('button');
    button.textContent = i;
    button.className = i === data.page ? 'active' : '';
    button.onclick = () => loadPage(i);
    pagination.appendChild(button);
  }
}

// 加载指定页
function loadPage(page) {
  getTaskList(page, 20);
}

// 筛选任务状态
function filterByStatus(status) {
  getTaskList(1, 20, status);
}
```

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
console.log('任务已创建:', task_id);
```

### 2. 查看任务列表

```javascript
// GET /api/v1/inference/tasks
const taskList = await fetch(
  'http://localhost:8004/api/v1/inference/tasks?page=1&page_size=20',
  {
    headers: { 'Authorization': `Bearer ${token}` }
  }
).then(r => r.json());

console.log('我的任务列表:', taskList);
```

### 3. 筛选特定状态的任务

```javascript
// 查看正在处理的任务
const processingTasks = await fetch(
  'http://localhost:8004/api/v1/inference/tasks?status=processing',
  {
    headers: { 'Authorization': `Bearer ${token}` }
  }
).then(r => r.json());

console.log('处理中的任务:', processingTasks);
```

### 4. 分页加载

```javascript
// 加载第二页
const page2 = await fetch(
  'http://localhost:8004/api/v1/inference/tasks?page=2&page_size=10',
  {
    headers: { 'Authorization': `Bearer ${token}` }
  }
).then(r => r.json());

console.log('第二页任务:', page2);
```

## 📊 任务状态说明

| 状态 | 说明 | 进度 | 操作 |
|------|------|------|------|
| `pending` | 等待中 | 0 | 等待系统处理 |
| `processing` | 处理中 | 1-99 | 查看进度 |
| `completed` | 已完成 | 100 | 下载结果 |
| `failed` | 失败 | 0-99 | 查看错误信息 |

## 🗄️ 数据库查询逻辑

### 查询条件

```sql
-- 基础查询（只查询当前用户的任务）
SELECT 
    task_id, content_id, audio_url, status, progress, message, result_url,
    created_at, updated_at
FROM audio_separation_tasks
WHERE user_id = ?  -- 强制匹配当前用户
ORDER BY created_at DESC
LIMIT ? OFFSET ?

-- 带状态筛选
SELECT ...
WHERE user_id = ? AND status = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?

-- 查询总数
SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = ?
```

### 索引优化

- `idx_audio_tasks_user_id` - 加速用户任务查询
- `idx_audio_tasks_status` - 加速状态筛选
- `idx_audio_tasks_created_at` - 加速时间排序

## ⚠️ 注意事项

1. **必须携带 JWT Token**
   - 没有 Token 会返回 401 未授权
   
2. **只能查询自己的任务**
   - 查询时强制匹配 `user_id`
   - 无法查看其他用户的任务

3. **分页限制**
   - 默认每页 20 条
   - 最大每页 100 条
   - 超过限制会自动调整为 100

4. **排序规则**
   - 按创建时间倒序排列（最新的在前）

5. **状态筛选**
   - 可选值：pending, processing, completed, failed
   - 不传则返回所有状态

## 🔗 相关接口

- **发起分离**: `POST /api/v1/audio/separate`
- **任务列表**: `GET /api/v1/inference/tasks`
- **健康检查**: `GET /api/v1/health`

## 🚀 后续优化建议

### 1. 批量操作

```javascript
// 批量删除任务
DELETE /api/v1/inference/tasks/batch
{
  "task_ids": ["task_1", "task_2", "task_3"]
}
```

### 2. 任务统计

```javascript
// 获取任务统计信息
GET /api/v1/inference/stats

响应示例：
{
  "total": 50,
  "pending": 5,
  "processing": 10,
  "completed": 30,
  "failed": 5
}
```

### 3. WebSocket 实时更新

替代轮询，使用 WebSocket 接收任务状态更新：

```javascript
const ws = new WebSocket('ws://localhost:8004/ws/tasks');
ws.onmessage = (event) => {
  const task = JSON.parse(event.data);
  updateTaskInList(task);
};
```
