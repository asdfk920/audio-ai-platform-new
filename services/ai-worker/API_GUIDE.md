# AI Worker 音频分离 API 使用指南

## 📋 概述

本指南介绍如何在前端实现音频分离功能，包括：
1. 上传音频文件
2. 创建分离任务
3. 查询任务状态
4. 获取分离结果

## 🔗 API 端点

- **服务地址**: `http://localhost:8004`
- **API 版本**: `/api/v1`
- **Swagger 文档**: `http://localhost:8004/swagger/`

## 📤 1. 上传音频文件

### 接口信息
- **URL**: `POST /api/v1/audio/upload`
- **Content-Type**: `multipart/form-data`
- **认证**: 可选（通过 `X-User-ID` Header 传递用户 ID）

### 请求参数
| 参数名 | 类型 | 位置 | 必填 | 说明 |
|--------|------|------|------|------|
| file | File | formData | 是 | 音频文件（支持 mp3, wav, flac, ogg, aac, m4a） |
| X-User-ID | int64 | header | 否 | 用户 ID |

### 请求示例

#### 使用 FormData（前端）
```javascript
// 选择文件
const fileInput = document.getElementById('audioFile');
const file = fileInput.files[0];

// 创建 FormData
const formData = new FormData();
formData.append('file', file);

// 上传文件
const response = await fetch('http://localhost:8004/api/v1/audio/upload', {
  method: 'POST',
  headers: {
    'X-User-ID': '123456' // 可选
  },
  body: formData
});

const result = await response.json();
console.log(result);
```

#### 使用 cURL
```bash
curl -X POST "http://localhost:8004/api/v1/audio/upload" \
  -H "X-User-ID: 123456" \
  -F "file=@/path/to/audio.mp3"
```

### 响应示例
```json
{
  "audio_id": "a1b2c3d4e5f6",
  "audio_url": "/static-media/uploads/123456/a1b2c3d4e5f6.mp3",
  "file_name": "my-song.mp3",
  "file_size": 5242880,
  "duration": 0,
  "upload_at": "2024-01-15T10:30:00Z",
  "message": "上传成功"
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| audio_id | string | 音频文件 ID（用于创建分离任务） |
| audio_url | string | 音频文件访问 URL |
| file_name | string | 原始文件名 |
| file_size | int64 | 文件大小（字节） |
| duration | int64 | 音频时长（秒） |
| upload_at | string | 上传时间（RFC3339 格式） |
| message | string | 返回消息 |

---

## 🚀 2. 创建音轨分离任务

### 接口信息
- **URL**: `POST /api/v1/inference/start`
- **Content-Type**: `application/json`
- **认证**: 可选（通过 `X-User-ID` Header 传递用户 ID）

### 请求参数
| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| audio_url | string | 是 | - | 音频文件 URL（从上传接口获取） |
| model_type | string | 否 | bsroformer | 模型类型：bsroformer |
| output_tracks | string[] | 否 | ["vocals","drums","bass","other"] | 输出音轨列表 |
| callback_url | string | 否 | - | 完成后回调地址 |

### 请求示例

#### JavaScript
```javascript
// 使用上传后获取的 audio_url
const audioUrl = result.audio_url; // 从上传接口获取

const response = await fetch('http://localhost:8004/api/v1/inference/start', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-User-ID': '123456'
  },
  body: JSON.stringify({
    audio_url: audioUrl,
    model_type: 'bsroformer',
    output_tracks: ['vocals', 'drums', 'bass', 'other']
  })
});

const task = await response.json();
console.log(task);
```

#### cURL
```bash
curl -X POST "http://localhost:8004/api/v1/inference/start" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 123456" \
  -d '{
    "audio_url": "/static-media/uploads/123456/a1b2c3d4e5f6.mp3",
    "model_type": "bsroformer",
    "output_tracks": ["vocals", "drums", "bass", "other"]
  }'
```

### 响应示例
```json
{
  "task_id": "task_20240115103000_1234567890",
  "status": "pending",
  "message": "任务创建成功",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| task_id | string | 任务 ID（用于查询进度和结果） |
| status | string | 任务状态：pending, processing, completed, failed |
| message | string | 返回消息 |
| created_at | string | 创建时间 |

---

## 🔍 3. 查询任务状态

### 接口信息
- **URL**: `GET /api/v1/inference/query?task_id={task_id}`
- **Method**: GET
- **认证**: 可选

### 请求参数
| 参数名 | 类型 | 位置 | 必填 | 说明 |
|--------|------|------|------|------|
| task_id | string | query | 是 | 任务 ID |

### 请求示例

#### JavaScript
```javascript
const taskId = task.task_id; // 从创建任务接口获取

const response = await fetch(
  `http://localhost:8004/api/v1/inference/query?task_id=${taskId}`,
  {
    method: 'GET',
    headers: {
      'X-User-ID': '123456'
    }
  }
);

const status = await response.json();
console.log(status);
```

#### cURL
```bash
curl -X GET "http://localhost:8004/api/v1/inference/query?task_id=task_20240115103000_1234567890"
```

### 响应示例
```json
{
  "task_id": "task_20240115103000_1234567890",
  "status": "processing",
  "progress": 50,
  "message": "正在分离音轨...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| task_id | string | 任务 ID |
| status | string | 任务状态 |
| progress | int | 进度百分比（0-100） |
| message | string | 状态描述 |
| created_at | string | 创建时间 |

### 轮询示例（前端）
```javascript
async function pollTaskStatus(taskId) {
  const maxAttempts = 60; // 最多轮询 60 次
  let attempts = 0;
  
  while (attempts < maxAttempts) {
    const response = await fetch(
      `http://localhost:8004/api/v1/inference/query?task_id=${taskId}`
    );
    const status = await response.json();
    
    console.log(`进度：${status.progress}% - ${status.message}`);
    
    if (status.status === 'completed') {
      console.log('✅ 分离完成！');
      return status;
    }
    
    if (status.status === 'failed') {
      console.error('❌ 分离失败:', status.message);
      throw new Error(status.message);
    }
    
    // 等待 2 秒后再次查询
    await new Promise(resolve => setTimeout(resolve, 2000));
    attempts++;
  }
  
  throw new Error('任务超时');
}
```

---

## 📥 4. 获取分离结果

### 接口信息
- **URL**: `GET /api/v1/inference/result?task_id={task_id}`
- **Method**: GET
- **认证**: 可选

### 请求参数
| 参数名 | 类型 | 位置 | 必填 | 说明 |
|--------|------|------|------|------|
| task_id | string | query | 是 | 任务 ID |

### 请求示例

#### JavaScript
```javascript
const taskId = task.task_id;

const response = await fetch(
  `http://localhost:8004/api/v1/inference/result?task_id=${taskId}`,
  {
    method: 'GET',
    headers: {
      'X-User-ID': '123456'
    }
  }
);

const result = await response.json();
console.log(result);

// 播放分离后的音轨
result.tracks.forEach(track => {
  console.log(`${track.label}: ${track.url}`);
  // 创建音频播放器
  const audio = new Audio(`http://localhost:8004${track.url}`);
  // audio.play();
});
```

#### cURL
```bash
curl -X GET "http://localhost:8004/api/v1/inference/result?task_id=task_20240115103000_1234567890"
```

### 响应示例
```json
{
  "task_id": "task_20240115103000_1234567890",
  "status": "completed",
  "original_audio_url": "/static-media/uploads/0/original.wav",
  "model_type": "bsroformer",
  "tracks": [
    {
      "name": "vocals",
      "label": "人声",
      "url": "/static-media/results/task_20240115103000_1234567890/vocals.wav",
      "duration": 180,
      "size": 15728640,
      "format": "wav",
      "sample_rate": 44100
    },
    {
      "name": "drums",
      "label": "鼓",
      "url": "/static-media/results/task_20240115103000_1234567890/drums.wav",
      "duration": 180,
      "size": 12582912,
      "format": "wav",
      "sample_rate": 44100
    },
    {
      "name": "bass",
      "label": "贝斯",
      "url": "/static-media/results/task_20240115103000_1234567890/bass.wav",
      "duration": 180,
      "size": 10485760,
      "format": "wav",
      "sample_rate": 44100
    },
    {
      "name": "other",
      "label": "其他",
      "url": "/static-media/results/task_20240115103000_1234567890/other.wav",
      "duration": 180,
      "size": 8388608,
      "format": "wav",
      "sample_rate": 44100
    }
  ],
  "created_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:35:00Z"
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| task_id | string | 任务 ID |
| status | string | 最终状态：completed |
| original_audio_url | string | 原始音频 URL |
| model_type | string | 使用的模型 |
| tracks | TrackInfo[] | 分离后的音轨列表 |
| created_at | string | 创建时间 |
| completed_at | string | 完成时间 |

### TrackInfo 字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| name | string | 音轨名称（vocals/drums/bass/other） |
| label | string | 显示标签（人声/鼓/贝斯/其他） |
| url | string | 音频文件下载地址 |
| duration | int64 | 时长（秒） |
| size | int64 | 文件大小（字节） |
| format | string | 格式（wav/mp3） |
| sample_rate | int | 采样率 |

---

## 🎯 完整前端示例

### HTML + JavaScript 完整示例
```html
<!DOCTYPE html>
<html>
<head>
  <title>音频分离工具</title>
  <style>
    .container { max-width: 800px; margin: 50px auto; padding: 20px; }
    .progress-bar { width: 100%; height: 30px; background: #f0f0f0; border-radius: 15px; overflow: hidden; }
    .progress-fill { height: 100%; background: #4CAF50; transition: width 0.3s; }
    .track { margin: 10px 0; padding: 10px; background: #f9f9f9; border-radius: 5px; }
    audio { width: 100%; margin-top: 5px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>🎵 音频分离工具</h1>
    
    <!-- 上传区域 -->
    <div class="upload-section">
      <input type="file" id="audioFile" accept="audio/*" />
      <button onclick="uploadAndSeparate()">上传并分离</button>
    </div>
    
    <!-- 进度区域 -->
    <div id="progress" style="display: none; margin-top: 20px;">
      <div class="progress-bar">
        <div id="progressFill" class="progress-fill" style="width: 0%"></div>
      </div>
      <p id="progressText">正在处理...</p>
    </div>
    
    <!-- 结果区域 -->
    <div id="results" style="display: none; margin-top: 20px;">
      <h2>分离结果</h2>
      <div id="trackList"></div>
    </div>
  </div>

  <script>
    let currentTaskId = null;
    
    async function uploadAndSeparate() {
      const fileInput = document.getElementById('audioFile');
      const file = fileInput.files[0];
      
      if (!file) {
        alert('请选择音频文件');
        return;
      }
      
      try {
        // 1. 上传文件
        const formData = new FormData();
        formData.append('file', file);
        
        const uploadResp = await fetch('http://localhost:8004/api/v1/audio/upload', {
          method: 'POST',
          body: formData
        });
        
        const uploadResult = await uploadResp.json();
        console.log('上传成功:', uploadResult);
        
        // 2. 创建分离任务
        const createResp = await fetch('http://localhost:8004/api/v1/inference/start', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            audio_url: uploadResult.audio_url,
            model_type: 'bsroformer',
            output_tracks: ['vocals', 'drums', 'bass', 'other']
          })
        });
        
        const task = await createResp.json();
        currentTaskId = task.task_id;
        console.log('任务创建成功:', task);
        
        // 3. 显示进度条
        document.getElementById('progress').style.display = 'block';
        
        // 4. 轮询任务状态
        await pollTaskStatus(task.task_id);
        
      } catch (error) {
        console.error('错误:', error);
        alert('处理失败：' + error.message);
      }
    }
    
    async function pollTaskStatus(taskId) {
      const maxAttempts = 60;
      let attempts = 0;
      
      while (attempts < maxAttempts) {
        const resp = await fetch(
          `http://localhost:8004/api/v1/inference/query?task_id=${taskId}`
        );
        const status = await resp.json();
        
        // 更新进度条
        const progressFill = document.getElementById('progressFill');
        const progressText = document.getElementById('progressText');
        progressFill.style.width = status.progress + '%';
        progressText.textContent = status.message;
        
        if (status.status === 'completed') {
          // 5. 获取结果
          await fetchResults(taskId);
          return;
        }
        
        if (status.status === 'failed') {
          throw new Error(status.message);
        }
        
        await new Promise(resolve => setTimeout(resolve, 2000));
        attempts++;
      }
      
      throw new Error('任务超时');
    }
    
    async function fetchResults(taskId) {
      const resp = await fetch(
        `http://localhost:8004/api/v1/inference/result?task_id=${taskId}`
      );
      const result = await resp.json();
      
      // 显示结果
      const trackList = document.getElementById('trackList');
      trackList.innerHTML = '';
      
      result.tracks.forEach(track => {
        const trackDiv = document.createElement('div');
        trackDiv.className = 'track';
        trackDiv.innerHTML = `
          <strong>${track.label}</strong> (${track.name})<br/>
          <audio controls src="http://localhost:8004${track.url}"></audio>
        `;
        trackList.appendChild(trackDiv);
      });
      
      document.getElementById('results').style.display = 'block';
      document.getElementById('progress').style.display = 'none';
    }
  </script>
</body>
</html>
```

---

## 🔧 注意事项

1. **文件大小限制**: 最大 50MB
2. **支持的音频格式**: mp3, wav, flac, ogg, aac, m4a
3. **处理时间**: 根据音频长度和 GPU 性能，通常需要 1-5 分钟
4. **并发限制**: 默认最多 3 个并发任务
5. **文件存储**: 上传的文件保存在 `./data/ai-worker-objects/uploads/` 目录
6. **结果存储**: 分离结果保存在 `./data/ai-worker-objects/results/` 目录

---

## 📞 技术支持

- **Swagger 文档**: http://localhost:8004/swagger/
- **健康检查**: GET http://localhost:8004/api/v1/health
