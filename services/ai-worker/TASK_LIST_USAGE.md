# 查询任务列表接口说明

## 📋 接口信息

**接口**: `GET /api/v1/inference/tasks`

**功能**: 查询用户的音频分离任务列表

**认证**: JWT Bearer Token（必需）

**权限**: 用户只能查询自己的任务

## 🔧 测试方法

### Apifox 配置

**请求配置：**
- **方法**: GET
- **URL**: `http://localhost:8004/api/v1/inference/tasks`
- **Headers**: 
  - `Authorization`: `Bearer {your_jwt_token}`

**可选参数：**
- `page`: 页码（默认 1）
- `page_size`: 每页数量（默认 20，最大 100）
- `status`: 状态筛选（pending/processing/completed/failed）

### 测试示例

#### 1. 查询所有任务

```bash
curl -X GET "http://localhost:8004/api/v1/inference/tasks" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 2. 分页查询

```bash
curl -X GET "http://localhost:8004/api/v1/inference/tasks?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 3. 状态筛选

```bash
curl -X GET "http://localhost:8004/api/v1/inference/tasks?status=completed" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 📊 响应示例

### 成功响应

```json
{
  "tasks": [
    {
      "task_id": "task_123_1_001",
      "content_id": 1,
      "audio_url": "http://localhost:8000/audio/test1.mp3",
      "status": "completed",
      "progress": 100,
      "message": "Completed",
      "result_url": "/static-media/results/task_123_1_001/task_123_1_001.zip",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:31:00Z"
    },
    {
      "task_id": "task_123_2_002",
      "content_id": 2,
      "audio_url": "http://localhost:8000/audio/test2.mp3",
      "status": "processing",
      "progress": 50,
      "message": "Processing...",
      "result_url": "",
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:05:00Z"
    },
    {
      "task_id": "task_123_3_003",
      "content_id": 3,
      "audio_url": "http://localhost:8000/audio/test3.mp3",
      "status": "pending",
      "progress": 0,
      "message": "Task created",
      "result_url": "",
      "created_at": "2024-01-15T11:20:00Z",
      "updated_at": "2024-01-15T11:20:00Z"
    }
  ],
  "total": 3,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| tasks | array | 任务列表 |
| tasks[].task_id | string | 任务唯一 ID |
| tasks[].content_id | int64 | 内容 ID |
| tasks[].audio_url | string | 原始音频 URL |
| tasks[].status | string | 状态：pending/processing/completed/failed |
| tasks[].progress | int | 进度：0-100 |
| tasks[].message | string | 任务消息 |
| tasks[].result_url | string | 结果 URL（完成后才有） |
| tasks[].created_at | string | 创建时间 |
| tasks[].updated_at | string | 更新时间 |
| total | int64 | 总记录数 |
| page | int | 当前页码 |
| page_size | int | 每页数量 |
| total_pages | int | 总页数 |

## 🔐 用户隔离机制

查询时会自动从 JWT Token 中提取 `user_id`，只返回该用户的任务。

**SQL 查询示例：**
```sql
SELECT * FROM audio_separation_tasks
WHERE user_id = 123  -- 从 JWT Token 中提取
ORDER BY created_at DESC
LIMIT 20 OFFSET 0
```

**测试用户隔离：**
- 用户 A（user_id=123）只能看到自己的 3 个任务
- 用户 B（user_id=456）只能看到自己的 1 个任务
- 即使用户 A 知道 user B 的 task_id，也无法查询到

## 📝 测试数据

当前数据库中有以下测试数据：

### 用户 123 的任务
1. task_123_1_001 - completed (100%)
2. task_123_2_002 - processing (50%)
3. task_123_3_003 - pending (0%)

### 用户 456 的任务
1. task_456_1_004 - completed (100%)

## 🎯 使用场景

### 1. 查看我的所有任务
```
GET /api/v1/inference/tasks
```

### 2. 查看已完成的任务
```
GET /api/v1/inference/tasks?status=completed
```

### 3. 查看正在处理的任务
```
GET /api/v1/inference/tasks?status=processing
```

### 4. 分页加载
```
GET /api/v1/inference/tasks?page=2&page_size=10
```

## 💻 前端集成示例

### JavaScript/TypeScript

```typescript
interface Task {
  task_id: string;
  content_id: number;
  status: string;
  progress: number;
  message: string;
  result_url: string;
  created_at: string;
  updated_at: string;
}

interface TaskListResponse {
  tasks: Task[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

async function getTaskList(
  page: number = 1, 
  pageSize: number = 20, 
  status?: string
): Promise<TaskListResponse> {
  const token = localStorage.getItem('jwt_token');
  
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: pageSize.toString()
  });
  
  if (status) {
    params.append('status', status);
  }
  
  const response = await fetch(
    `http://localhost:8004/api/v1/inference/tasks?${params}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );
  
  if (!response.ok) {
    throw new Error('查询失败');
  }
  
  return await response.json();
}

// 使用示例
async function loadTasks() {
  try {
    const data = await getTaskList(1, 20);
    console.log('任务列表:', data);
    
    // 渲染任务列表
    renderTaskList(data.tasks);
  } catch (error) {
    console.error('加载失败:', error);
  }
}
```

### Vue 3 示例

```vue
<template>
  <div class="task-list">
    <h2>我的任务</h2>
    
    <!-- 状态筛选 -->
    <div class="filters">
      <button @click="filterStatus('')">全部</button>
      <button @click="filterStatus('pending')">等待中</button>
      <button @click="filterStatus('processing')">处理中</button>
      <button @click="filterStatus('completed')">已完成</button>
    </div>
    
    <!-- 任务列表 -->
    <div v-for="task in tasks" :key="task.task_id" class="task-item">
      <h3>{{ task.task_id }}</h3>
      <p>状态：{{ task.status }}</p>
      <p>进度：{{ task.progress }}%</p>
      <p>消息：{{ task.message }}</p>
      
      <progress :value="task.progress" max="100"></progress>
      
      <a v-if="task.result_url" :href="task.result_url" download>
        下载结果
      </a>
    </div>
    
    <!-- 分页 -->
    <div class="pagination">
      <button @click="loadPage(page - 1)" :disabled="page <= 1">上一页</button>
      <span>第 {{ page }} / {{ totalPages }} 页</span>
      <button @click="loadPage(page + 1)" :disabled="page >= totalPages">下一页</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Task {
  task_id: string
  content_id: number
  status: string
  progress: number
  message: string
  result_url: string
  created_at: string
  updated_at: string
}

const tasks = ref<Task[]>([])
const page = ref(1)
const totalPages = ref(1)
const currentStatus = ref('')

async function loadTasks(status: string = '', pageNum: number = 1) {
  const token = localStorage.getItem('jwt_token')
  
  const params = new URLSearchParams({
    page: pageNum.toString(),
    page_size: '20'
  })
  
  if (status) {
    params.append('status', status)
  }
  
  const response = await fetch(
    `http://localhost:8004/api/v1/inference/tasks?${params}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  )
  
  const data = await response.json()
  tasks.value = data.tasks
  page.value = data.page
  totalPages.value = data.total_pages
}

function filterStatus(status: string) {
  currentStatus.value = status
  page.value = 1
  loadTasks(status, 1)
}

function loadPage(pageNum: number) {
  loadTasks(currentStatus.value, pageNum)
}

onMounted(() => {
  loadTasks()
})
</script>
```

## 🔗 相关接口

- **发起分离**: `POST /api/v1/audio/separate`
- **任务列表**: `GET /api/v1/inference/tasks`
- **健康检查**: `GET /api/v1/health`

## ⚠️ 注意事项

1. **必须携带 JWT Token**
   - 没有 Token 会返回 401 未授权

2. **只能查询自己的任务**
   - 查询时强制匹配 `user_id`
   - 无法查看其他用户的任务

3. **分页限制**
   - 默认每页 20 条
   - 最大每页 100 条

4. **排序规则**
   - 按创建时间倒序排列（最新的在前）

## 🐛 常见问题

### Q: 查询返回空列表？
A: 检查 JWT Token 是否正确，确认该用户是否有任务记录

### Q: 如何查看其他用户的任务？
A: 出于安全考虑，不支持查看其他用户的任务

### Q: 任务列表最多显示多少条？
A: 默认 20 条，可通过 `page_size` 参数调整，最大 100 条

### Q: 如何实时更新任务状态？
A: 使用轮询（每 2-3 秒查询一次）或 WebSocket（需额外实现）
