# 音频分离完整流程文档

## 📋 概述

本文档介绍用户在前端点击音轨分离按钮后，后端完整的处理流程。

## 🔄 完整流程

### 1. 用户操作

1. 用户在前端页面浏览音频列表
2. 点击某个音频的"音轨分离"按钮
3. 前端获取该音频的 `content_id`

### 2. 发起分离任务

**前端调用：**
```javascript
POST http://localhost:8004/api/v1/audio/separate
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "content_id": 3
}
```

**后端处理流程：**

#### Step 1: 验证请求
- 解析 JWT Token 获取 `user_id`
- 验证 `content_id` 有效性

#### Step 2: 查询音频 URL
```go
SELECT audio_url FROM content 
WHERE id = $1 AND is_deleted = 0 AND status = 1
```

#### Step 3: 生成任务 ID
```go
task_id = fmt.Sprintf("task_%d_%d_%d", user_id, content_id, timestamp)
```

#### Step 4: 创建任务记录
```sql
INSERT INTO audio_separation_tasks (
    task_id, user_id, content_id, audio_url, 
    status, progress, message, created_at, updated_at
) VALUES (
    'task_xxx', 123, 3, 'http://...',
    'pending', 0, '任务已创建', NOW(), NOW()
)
```

#### Step 5: 异步处理任务
启动 goroutine 执行分离流程

### 3. 音频分离处理流程

#### 3.1 下载音频文件

**状态更新：**
- status: `processing`
- progress: `10`
- message: `正在下载音频文件...`

**下载过程：**
```go
// 创建临时目录
tempDir = /tmp/audio-separation/task_xxx/

// 从 URL 下载
response = http.get(audio_url)
file = create(tempDir + "/input.wav")
copy(response.body, file)
```

**状态更新：**
- progress: `30`
- message: `音频下载完成，正在分离...`

#### 3.2 调用 AI 模型分离

**提交任务到 AI 服务队列：**
```go
job = &SeparationJob{
    ID: task_id,
    InputPath: "/tmp/audio-separation/task_xxx/input.wav",
    Config: aiService.GetConfig(),
    Result: make(chan),
    Error: make(chan),
    Progress: make(chan),
}

aiService.SubmitJob(job)
```

**AI 服务处理：**
1. 从队列获取任务
2. 加载音频文件
3. 使用 Demucs 模型进行分离
4. 生成 4 条音轨：vocals, drums, bass, other
5. 保存分离结果到临时文件

**进度更新：**
- progress: `50-70` (通过 Progress channel 实时更新)

#### 3.3 保存分离结果

**创建结果目录：**
```
static-media/results/task_xxx/
```

**复制音轨文件：**
- `vocals.wav` - 人声音轨
- `drums.wav` - 鼓点音轨
- `bass.wav` - 贝斯音轨
- `other.wav` - 其他音轨

**生成 ZIP 压缩包：**
```
static-media/results/task_xxx/task_xxx.zip
```

**状态更新：**
- progress: `80`
- message: `分离完成，正在保存结果...`

#### 3.4 更新任务状态

**最终状态：**
```sql
UPDATE audio_separation_tasks
SET status = 'completed',
    progress = 100,
    message = '分离完成',
    result_url = '/static-media/results/task_xxx/task_xxx.zip',
    updated_at = NOW()
WHERE task_id = 'task_xxx'
```

### 4. 查询任务列表

**前端调用：**
```javascript
GET http://localhost:8004/api/v1/inference/tasks
Authorization: Bearer {jwt_token}
```

**响应示例：**
```json
{
  "tasks": [
    {
      "task_id": "task_123_3_xxx",
      "content_id": 3,
      "audio_url": "http://localhost:8000/audio/xxx.mp3",
      "status": "completed",
      "progress": 100,
      "message": "分离完成",
      "result_url": "/static-media/results/task_123_3_xxx/task_123_3_xxx.zip",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:31:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

### 5. 下载分离结果

**前端调用：**
```javascript
GET http://localhost:8004/static-media/results/task_xxx/task_xxx.zip
```

**下载后解压：**
```
task_xxx.zip
├── vocals.wav    # 人声
├── drums.wav     # 鼓点
├── bass.wav      # 贝斯
└── other.wav     # 其他
```

## 📊 任务状态流转

```
pending (0%)
  ↓
processing (10%) - 下载音频
  ↓
processing (30%) - 开始分离
  ↓
processing (50-70%) - AI 推理中
  ↓
processing (80%) - 保存结果
  ↓
completed (100%) - 完成
```

**异常流程：**
```
任何状态 → failed (0-99%) - 发生错误
```

## 🗄️ 数据库表结构

### audio_separation_tasks

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| task_id | VARCHAR(100) | 任务唯一 ID |
| user_id | BIGINT | 用户 ID |
| content_id | BIGINT | 内容 ID |
| audio_url | VARCHAR(2000) | 原始音频 URL |
| status | VARCHAR(20) | 状态 |
| progress | INTEGER | 进度 0-100 |
| message | TEXT | 消息 |
| result_url | VARCHAR(2000) | 结果 URL |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 索引

```sql
CREATE INDEX idx_audio_tasks_task_id ON audio_separation_tasks(task_id);
CREATE INDEX idx_audio_tasks_user_id ON audio_separation_tasks(user_id);
CREATE INDEX idx_audio_tasks_status ON audio_separation_tasks(status);
CREATE INDEX idx_audio_tasks_created_at ON audio_separation_tasks(created_at);
```

## 🔧 错误处理

### 常见错误及处理

#### 1. 音频下载失败
```
错误：下载请求失败：404 Not Found
处理：更新任务状态为 failed，message="下载音频失败：404"
```

#### 2. AI 模型分离失败
```
错误：模型推理失败：内存不足
处理：更新任务状态为 failed，message="AI 分离失败：内存不足"
```

#### 3. 保存结果失败
```
错误：磁盘空间不足
处理：更新任务状态为 failed，message="保存结果失败：磁盘空间不足"
```

#### 4. 任务超时
```
错误：分离超时（10 分钟）
处理：更新任务状态为 failed，message="分离超时"
```

## 📁 目录结构

```
ai-worker/
├── static-media/
│   └── results/
│       └── task_xxx/
│           ├── vocals.wav
│           ├── drums.wav
│           ├── bass.wav
│           ├── other.wav
│           └── task_xxx.zip
├── tmp/
│   └── audio-separation/
│       └── task_xxx/
│           └── input.wav
└── migrations/
    └── 001_create_audio_separation_tasks.sql
```

##  关键代码

### 发起分离

```go
// internal/logic/audio_separate_logic.go
func (l *AudioSeparateLogic) SeparateAudio(r *AudioSeparateReq, userID int64) (*AudioSeparateResp, error) {
    // 1. 验证 content_id
    // 2. 查询音频 URL
    // 3. 生成 task_id
    // 4. 保存到数据库
    // 5. 异步处理：go l.processSeparation(taskID, audioURL)
}
```

### 异步处理

```go
func (l *AudioSeparateLogic) processSeparation(taskID, audioURL string) {
    // 1. 更新状态：processing, progress=10
    // 2. 下载音频
    // 3. 更新状态：processing, progress=30
    // 4. 调用 AI 模型
    // 5. 更新状态：processing, progress=80
    // 6. 保存结果
    // 7. 更新状态：completed, progress=100
}
```

### 查询任务列表

```go
// internal/logic/task_query_logic.go
func (l *TaskListLogic) GetTaskList(req *TaskListReq, userID int64) (*TaskListResp, error) {
    // 1. 验证分页参数
    // 2. 查询数据库：SELECT * WHERE user_id = ?
    // 3. 返回任务列表和分页信息
}
```

## 🚀 性能优化

### 1. 并发控制
- 最大并发任务数：3（可配置）
- 任务队列容量：6（2 倍并发数）

### 2. 资源清理
- 临时文件自动删除
- 完成后清理输入文件
- 定期清理过期结果

### 3. 数据库优化
- 使用连接池
- 建立索引加速查询
- 分页查询避免全表扫描

## 📝 前端集成示例

### Vue 3 示例

```vue
<template>
  <div>
    <button @click="startSeparation(contentId)">音轨分离</button>
    
    <div v-if="currentTask">
      <h4>分离进度</h4>
      <progress :value="progress" max="100"></progress>
      <span>{{ progress }}%</span>
      <p>{{ message }}</p>
      
      <a v-if="completed" :href="resultUrl" download>下载结果</a>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const currentTask = ref(null)
const progress = ref(0)
const message = ref('')
const completed = ref(false)
const resultUrl = ref('')

// 发起分离
async function startSeparation(contentId) {
  const token = localStorage.getItem('jwt_token')
  
  const response = await fetch('/api/v1/audio/separate', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ content_id: contentId })
  })
  
  const data = await response.json()
  console.log('任务已创建:', data.task_id)
  
  // 开始轮询进度
  pollProgress(data.task_id)
}

// 轮询进度
function pollProgress(taskId) {
  const interval = setInterval(async () => {
    const token = localStorage.getItem('jwt_token')
    
    const response = await fetch(`/api/v1/inference/tasks?task_id=${taskId}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    const data = await response.json()
    const task = data.tasks[0]
    
    if (task) {
      progress.value = task.progress
      message.value = task.message
      
      if (task.status === 'completed') {
        completed.value = true
        resultUrl.value = task.result_url
        clearInterval(interval)
      } else if (task.status === 'failed') {
        message.value = '分离失败：' + task.message
        clearInterval(interval)
      }
    }
  }, 2000)
}
</script>
```

## 🔗 相关接口

- **发起分离**: `POST /api/v1/audio/separate`
- **任务列表**: `GET /api/v1/inference/tasks`
- **健康检查**: `GET /api/v1/health`

## 📖 参考资料

- [Demucs 模型文档](https://github.com/facebookresearch/demucs)
- [BSRoformer 模型文档](https://github.com/ZFTurbo/BSRoFormer)
- [PostgreSQL 官方文档](https://www.postgresql.org/docs/)
