# WebSocket 音轨分离接口使用指南

## 📋 功能概述

用户通过 WebSocket 连接发送 `content_id`，服务端异步执行以下操作：

1. ✅ 验证 JWT Token
2. ✅ 根据 `content_id` 查询音频 URL
3. ✅ 创建分离任务记录
4. ✅ 调用 AI 模型进行异步分离
5. ✅ 保存分离后的各个音轨到 `audio_tracks` 表

## 🔗 连接信息

**WebSocket URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id={task_id}&token={jwt_token}
```

**参数说明**:
- `task_id` (必需): 任务 ID，用于标识本次分离任务
- `token` (必需): JWT 认证 token（从登录接口获取）

## 📤 发送消息格式

连接成功后，发送 JSON 消息：

```json
{
  "type": "separate_request",
  "content_id": 123
}
```

**参数说明**:
- `type`: 固定为 `"separate_request"`
- `content_id`: 音频内容 ID（从 `audio_contents` 表查询）

## 📥 接收响应格式

### 1. 任务已接收

```json
{
  "type": "separate_response",
  "task_id": "task_001",
  "status": "pending",
  "message": "任务已接收，正在查询音频信息",
  "content_id": 123
}
```

### 2. 任务已提交（连接关闭前）

```json
{
  "type": "separate_response",
  "task_id": "task_001",
  "status": "processing",
  "message": "分离任务已提交，请稍后查询任务列表查看结果",
  "content_id": 123
}
```

### 3. 错误响应

```json
{
  "type": "error",
  "task_id": "task_001",
  "message": "错误信息"
}
```

## 🔄 完整流程

```
┌─────────┐                ┌──────────────┐                ┌─────────────┐
│  前端   │                │  WebSocket   │                │  AI 模型服务 │
│         │                │   Handler    │                │             │
└────┬────┘                └──────┬───────┘                └──────┬──────┘
     │                           │                               │
     │ 1. 建立 WebSocket 连接     │                               │
     │    (task_id + token)      │                               │
     ├──────────────────────────>│                               │
     │                           │                               │
     │ 2. 发送 content_id        │                               │
     │    {type, content_id}     │                               │
     ├──────────────────────────>│                               │
     │                           │                               │
     │                           │ 3. 查询音频 URL               │
     │                           │    SELECT audio_url           │
     │                           │    FROM audio_contents        │
     │                           ├──────────────────────────────>│
     │                           │                               │
     │ 4. 回复：pending          │                               │
     │    "任务已接收"           │                               │
     │<──────────────────────────┤                               │
     │                           │                               │
     │                           │ 5. 创建任务记录               │
     │                           │    INSERT INTO                │
     │                           │    audio_separation_tasks     │
     │                           │                               │
     │ 6. 回复：processing       │                               │
     │    "任务已提交"           │                               │
     │<──────────────────────────┤                               │
     │                           │                               │
     │ (连接关闭)                │ 7. 异步分离                   │
     │                           │    - 下载音频                 │
     │                           │    - AI 分离                  │
     │                           │    - 保存音轨                 │
     │                           ├──────────────────────────────>│
     │                           │                               │
     │                           │ 8. 保存结果到 audio_tracks    │
     │                           │    INSERT INTO audio_tracks   │
     │                           │    (task_id, track_name,      │
     │                           │     track_url, ...)           │
     │                           │                               │
     └                           └                               └

```

## 🗄️ 数据库表结构

### audio_separation_tasks（分离任务表）

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | BIGSERIAL | 主键 |
| task_id | VARCHAR(100) | 任务 ID |
| user_id | BIGINT | 用户 ID |
| content_id | BIGINT | 音频内容 ID |
| audio_url | VARCHAR(2000) | 原始音频 URL |
| status | VARCHAR(20) | 状态（pending/processing/completed/failed） |
| progress | INT | 进度（0-100） |
| message | VARCHAR(500) | 状态消息 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### audio_tracks（音轨结果表）

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | BIGSERIAL | 主键 |
| task_id | VARCHAR(100) | 所属任务 ID |
| track_name | VARCHAR(50) | 音轨名称（vocals/drums/bass/other） |
| track_url | VARCHAR(2000) | 音轨文件 URL |
| file_size | BIGINT | 文件大小（字节） |
| duration | DOUBLE PRECISION | 时长（秒） |
| sample_rate | INT | 采样率 |
| channels | INT | 声道数 |
| format | VARCHAR(20) | 文件格式（wav） |
| order_index | INT | 排序索引 |
| created_at | TIMESTAMP | 创建时间 |

## 💡 Apifox 使用示例

### 1. 建立连接

**URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=task_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**请求头**（可选）:
```
Authorization: Bearer {token}
```

### 2. 发送消息

连接成功后，在"发送消息"区域输入：

```json
{
  "type": "separate_request",
  "content_id": 1
}
```

点击"发送"按钮。

### 3. 查看响应

在"消息列表"中查看服务端返回的消息：

```
[接收] {"type":"separate_response","task_id":"task_001","status":"pending","message":"任务已接收，正在查询音频信息","content_id":1}
[接收] {"type":"separate_response","task_id":"task_001","status":"processing","message":"分离任务已提交，请稍后查询任务列表查看结果","content_id":1}
[断开] 连接已关闭
```

## 🔍 查询任务状态

分离任务是异步执行的，可以通过查询任务列表接口查看进度：

**HTTP 接口**:
```
GET http://localhost:8004/api/v1/inference/tasks?status=processing&page=1&page_size=10
```

**响应示例**:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "tasks": [
      {
        "task_id": "task_001",
        "user_id": 1,
        "content_id": 1,
        "audio_url": "http://...",
        "status": "processing",
        "progress": 50,
        "message": "音频下载完成，正在分离...",
        "created_at": "2026-05-08T16:20:00+08:00",
        "updated_at": "2026-05-08T16:20:30+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
  }
}
```

##  音轨查询

分离完成后，查询 `audio_tracks` 表获取分离结果：

```sql
SELECT 
  id,
  track_name,
  track_url,
  file_size,
  duration,
  sample_rate,
  channels,
  format
FROM audio_tracks
WHERE task_id = 'task_001'
ORDER BY order_index;
```

**示例结果**:

| id | track_name | track_url | file_size | duration | sample_rate | channels | format |
|----|------------|-----------|-----------|----------|-------------|----------|--------|
| 1 | vocals | /static/tracks/task_001_vocals.wav | 5242880 | 180.5 | 44100 | 2 | wav |
| 2 | drums | /static/tracks/task_001_drums.wav | 3145728 | 180.5 | 44100 | 2 | wav |
| 3 | bass | /static/tracks/task_001_bass.wav | 2097152 | 180.5 | 44100 | 2 | wav |
| 4 | other | /static/tracks/task_001_other.wav | 1048576 | 180.5 | 44100 | 2 | wav |

## ⚠️ 注意事项

### 1. Token 有效期

确保使用的 JWT token 在有效期内，过期需要重新登录获取。

### 2. 异步处理

- WebSocket 连接在发送完请求后关闭
- 分离任务在后台异步执行
- 通过查询任务列表接口获取进度

### 3. 错误处理

常见错误及解决方案：

| 错误信息 | 原因 | 解决方案 |
|----------|------|----------|
| 认证失败：无效的 token | Token 错误或过期 | 重新登录获取新 token |
| 缺少 task_id 参数 | URL 中没有 task_id | 添加 task_id 参数 |
| content_id 必须大于 0 | content_id 无效 | 检查 content_id 是否正确 |
| 查询音频信息失败 | content_id 不存在 | 确认音频是否存在 |
| AI 分离失败 | 模型处理失败 | 检查日志，联系管理员 |

### 4. 性能优化

- 单个任务异步执行，不阻塞 WebSocket 连接
- 支持多个用户同时发起分离任务
- 数据库连接使用连接池

## 🚀 前端集成示例

### JavaScript/TypeScript

```typescript
class AudioSeparationClient {
  private ws: WebSocket | null = null;
  private taskCallback?: (response: any) => void;

  connect(token: string, taskId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = `ws://localhost:8004/api/v1/audio/separate/ws?task_id=${taskId}&token=${token}`;
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        console.log('WebSocket 连接成功');
        resolve();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket 连接失败', error);
        reject(error);
      };

      this.ws.onmessage = (event) => {
        const response = JSON.parse(event.data);
        console.log('收到响应:', response);
        
        if (this.taskCallback) {
          this.taskCallback(response);
        }

        // 收到 processing 响应后关闭连接
        if (response.status === 'processing') {
          this.disconnect();
        }
      };

      this.ws.onclose = () => {
        console.log('WebSocket 连接关闭');
      };
    });
  }

  sendSeparateRequest(contentId: number, callback?: (response: any) => void): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket 未连接');
    }

    this.taskCallback = callback;

    const request = {
      type: 'separate_request',
      content_id: contentId
    };

    this.ws.send(JSON.stringify(request));
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// 使用示例
async function separateAudio() {
  const client = new AudioSeparationClient();
  
  try {
    // 1. 连接 WebSocket
    await client.connect('your_jwt_token', 'task_001');
    
    // 2. 发送分离请求
    client.sendSeparateRequest(123, (response) => {
      console.log('任务状态:', response.status, response.message);
      
      if (response.status === 'processing') {
        // 任务已提交，轮询查询进度
        pollTaskStatus('task_001');
      }
    });
  } catch (error) {
    console.error('分离失败:', error);
  }
}

// 轮询任务状态
function pollTaskStatus(taskId: string) {
  const interval = setInterval(async () => {
    const response = await fetch(`http://localhost:8004/api/v1/inference/tasks?task_id=${taskId}`);
    const data = await response.json();
    
    console.log('任务进度:', data.data.tasks[0]?.progress, '%');
    
    if (data.data.tasks[0]?.status === 'completed') {
      clearInterval(interval);
      console.log('分离完成！');
      
      // 查询音轨结果
      queryTrackResults(taskId);
    }
  }, 2000);
}

// 查询音轨结果
async function queryTrackResults(taskId: string) {
  const response = await fetch(`http://localhost:8004/api/v1/audio/tracks?task_id=${taskId}`);
  const data = await response.json();
  
  console.log('分离结果:', data.data.tracks);
  // data.data.tracks 包含所有分离后的音轨信息
}
```

## 📝 总结

### 优势

✅ **简单易用**：只需发送 `content_id` 即可  
✅ **异步处理**：不阻塞用户操作  
✅ **实时反馈**：通过 WebSocket 即时响应  
✅ **可扩展**：支持多用户并发  
✅ **持久化**：结果保存到数据库  

### 使用流程

1. 用户登录获取 token
2. 前端建立 WebSocket 连接
3. 发送 `content_id`
4. 接收任务提交确认
5. 轮询查询任务进度
6. 获取分离结果

现在可以在 Apifox 中测试了！🎵
