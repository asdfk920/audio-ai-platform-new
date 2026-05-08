# WebSocket 音轨分离接口 - 完整使用指南

## ✅ 修复完成

### 问题已解决
1. ✅ **路由配置错误** - WebSocket 路由移到独立公开组，不再被 JWT 中间件拦截
2. ✅ **从 content 表获取音频列表** - 连接后自动返回可分离的音频列表
3. ✅ **完整的消息协议** - 清晰的消息类型和格式

---

## 🔗 连接信息

**WebSocket URL**:
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=你的任务ID&token=你的JWT_TOKEN
```

**参数说明**:
- `task_id` (可选): 任务标识，不传则自动生成
- `token` (可选): JWT 认证 token，支持 URL 参数或 Authorization Header

---

## 📋 完整工作流程

### 步骤 1: 建立连接

在 Apifox 中：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_001&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

点击"连接"

### 步骤 2: 自动接收音频列表

连接成功后，服务端会自动发送音频列表：

```json
{
  "type": "audio_list",
  "message": "获取到 17 条音频记录",
  "data": {
    "audio_list": [
      {
        "id": 1,
        "title": "001",
        "audio_url": "https://p26-flow-image...",
        "duration_sec": 5,
        "artist": "01"
      },
      {
        "id": 2,
        "title": "nihao",
        "audio_url": "/audio/test1.mp3",
        "duration_sec": 180,
        "artist": "lisi"
      },
      {
        "id": 4,
        "title": "免费测试内容",
        "audio_url": "/audio/test1.mp3",
        "duration_sec": 180,
        "artist": "测试艺术"
      }
    ],
    "total": 17
  }
}
```

### 步骤 3: 选择音频并发起分离

从列表中选择一个 audio_id（例如 id=2），发送：

```json
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}
```

### 步骤 4: 接收状态更新

**状态 1**: 任务接收
```json
{
  "type": "status",
  "message": "任务已接收，正在查询音频信息"
}
```

**状态 2**: 分离完成
```json
{
  "type": "separation_complete",
  "message": "分离成功！",
  "data": {
    "task_id": "test_001",
    "content_id": 2,
    "tracks": [
      {
        "id": 10,
        "track_name": "vocals",
        "track_url": "/static/tracks/test_001_vocals.wav"
      },
      {
        "id": 11,
        "track_name": "drums",
        "track_url": "/static/tracks/test_001_drums.wav"
      },
      {
        "id": 12,
        "track_name": "bass",
        "track_url": "/static/tracks/test_001_bass.wav"
      },
      {
        "id": 13,
        "track_name": "other",
        "track_url": "/static/tracks/test_001_other.wav"
      }
    ]
  }
}
```

### 步骤 5: 错误处理

如果出现错误：
```json
{
  "type": "error",
  "error": "查询音频失败：sql: no rows in result set"
}
```

---

## 🔄 可选操作

### 刷新音频列表

随时可以重新获取音频列表：

```json
{
  "type": "get_audio_list"
}
```

响应：
```json
{
  "type": "audio_list",
  "message": "获取到 17 条音频记录",
  "data": { ... }
}
```

---

## 📊 数据库表结构

### audio_contents（内容表 - 音频来源）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT8 | 主键 |
| title | VARCHAR(500) | 标题 |
| cover_url | VARCHAR(1000) | 封面图 |
| audio_url | VARCHAR(2000) | **音频文件 URL** |
| duration_sec | INT4 | 时长（秒） |
| artist | VARCHAR(100) | 艺术家 |

### audio_separation_tasks（分离任务表）

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | VARCHAR(100) | 任务 ID |
| user_id | BIGINT | 用户 ID |
| content_id | BIGINT | 内容 ID |
| audio_url | VARCHAR(2000) | 原始音频 URL |
| status | VARCHAR(20) | 状态 |
| progress | INT | 进度 0-100 |
| message | VARCHAR(500) | 状态消息 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### audio_tracks（音轨结果表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| task_id | VARCHAR(100) | 任务 ID |
| track_name | VARCHAR(50) | 音轨名称 |
| track_url | VARCHAR(2000) | 音轨 URL |
| file_size | BIGINT | 文件大小 |
| duration | DOUBLE PRECISION | 时长 |
| sample_rate | INT | 采样率 |
| channels | INT | 声道数 |
| format | VARCHAR(20) | 格式 |
| order_index | INT | 排序索引 |
| created_at | TIMESTAMP | 创建时间 |

---

## 💡 消息类型说明

| Type | 方向 | 说明 |
|------|------|------|
| `audio_list` | 服务端→客户端 | 音频列表 |
| `status` | 服务端→客户端 | 任务状态更新 |
| `separation_complete` | 服务端→客户端 | 分离完成 + 结果 |
| `error` | 服务端→客户端 | 错误信息 |
| `separate` | 客户端→服务端 | 发起分离请求 |
| `get_audio_list` | 客户端→服务端 | 获取音频列表 |

---

## ⚠️ 常见问题

### Q1: 连接失败（断开连接/异常）
**原因**: JWT 中间件拦截  
**解决方案**: 已修复！WebSocket 路由现在独立于 JWT 中间件

### Q2: 收不到音频列表
**检查**:
- 数据库中是否有数据：`SELECT COUNT(*) FROM audio_contents;`
- 是否有 audio_url：`SELECT * FROM audio_contents WHERE audio_url IS NOT NULL;`

### Q3: 分离失败
**可能原因**:
- content_id 不存在
- AI 模型未初始化
- 音频文件无法访问

**查看日志**:
```bash
# 在终端查看实时日志
tail -f ai-worker.log
```

---

## 🎯 测试步骤总结

1. **Apifox 连接**:
   ```
   ws://localhost:8004/api/v1/audio/separate/ws?task_id=my_task&token=你的TOKEN
   ```

2. **等待自动响应**:
   - 收到 `audio_list` 消息，包含所有可用音频

3. **选择并分离**:
   ```json
   {"type":"separate","data":{"content_id":2}}
   ```

4. **等待结果**:
   - 收到 `separation_complete` 消息，包含分离后的各个音轨

5. **查询数据库验证**:
   ```sql
   SELECT * FROM audio_tracks WHERE task_id = 'my_task';
   ```

---

## 🚀 服务状态

✅ 服务运行中：`http://localhost:8004`  
✅ WebSocket 接口：`/api/v1/audio/separate/ws`  
✅ 从 content 表获取音频列表  
✅ 异步分离 + 实时返回结果  
✅ 同时保存到数据库  

**现在可以在 Apifox 中测试了！** 🎵
