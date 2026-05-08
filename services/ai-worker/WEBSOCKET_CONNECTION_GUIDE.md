# WebSocket 音轨分离连接指南

## 🎯 连接方式

用户在前端页面点击分离按钮时，通过 WebSocket 连接到推理服务进行音轨分离。

### 连接 URL

```
ws://localhost:8004/api/v1/audio/separate/ws?task_id={task_id}&token={jwt_token}
```

**参数说明**：
- `task_id`：任务唯一标识（必填）
- `token`：用户登录后获取的 JWT Token（必填）

## 🔑 获取 Token

用户登录后，从登录接口获取 JWT Token：

```bash
POST http://localhost:8080/api/v1/user/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应**：
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expire": 1715251200
  }
}
```

## 💻 前端实现示例

### JavaScript/TypeScript

```typescript
class AudioSeparator {
  private ws: WebSocket | null = null;
  private taskConfig: any;
  private audioChunks: ArrayBuffer[] = [];

  constructor(taskId: string, token: string) {
    this.taskConfig = {
      type: 'task_config',
      task_id: taskId,
      sample_rate: 44100,
      channels: 2,
      tracks: ['vocals', 'drums', 'bass', 'other']
    };
    
    // 建立 WebSocket 连接
    const url = `ws://localhost:8004/api/v1/audio/separate/ws?task_id=${taskId}&token=${token}`;
    this.ws = new WebSocket(url);
    this.ws.binaryType = 'arraybuffer';
    
    this.setupEventHandlers();
  }

  private setupEventHandlers() {
    this.ws!.onopen = () => this.onOpen();
    this.ws!.onmessage = (event) => this.onMessage(event);
    this.ws!.onerror = (error) => this.onError(error);
    this.ws!.onclose = () => this.onClose();
  }

  private onOpen() {
    console.log('WebSocket 连接已建立');
    // 发送任务配置
    this.ws!.send(JSON.stringify(this.taskConfig));
  }

  private onMessage(event: MessageEvent) {
    if (event.data instanceof Blob) {
      // 接收音轨二进制数据
      this.handleBinaryData(event.data);
    } else {
      // 接收 JSON 消息
      const message = JSON.parse(event.data);
      this.handleTextMessage(message);
    }
  }

  private handleTextMessage(message: any) {
    switch (message.type) {
      case 'ready':
        console.log('服务端已准备就绪，开始发送音频数据');
        this.sendAudioFile();
        break;
        
      case 'result_header':
        console.log('分离完成，音轨数量:', message.track_count);
        console.log('音轨列表:', message.tracks);
        break;
        
      case 'completed':
        console.log('所有数据接收完成');
        this.close();
        break;
        
      case 'error':
        console.error('错误:', message.message);
        this.close();
        break;
    }
  }

  private handleBinaryData(blob: Blob) {
    blob.arrayBuffer().then(buffer => {
      console.log('收到音轨数据，大小:', buffer.byteLength);
      // 保存为 WAV 文件
      this.saveTrack(buffer);
    });
  }

  private async sendAudioFile() {
    // 从 input 元素获取音频文件
    const input = document.getElementById('audioFile') as HTMLInputElement;
    const file = input!.files![0];
    
    if (!file) {
      console.error('请选择音频文件');
      return;
    }

    const reader = new FileReader();
    const chunkSize = 64 * 1024; // 64KB
    let offset = 0;

    const sendNextChunk = () => {
      if (offset >= file.size) {
        // 发送结束标记
        this.ws!.send(JSON.stringify({ type: 'audio_end' }));
        console.log('音频数据发送完成');
        return;
      }

      const chunk = file.slice(offset, offset + chunkSize);
      reader.readAsArrayBuffer(chunk);
    };

    reader.onload = (e) => {
      const chunk = e.target!.result as ArrayBuffer;
      this.ws!.send(chunk);
      offset += chunkSize;
      console.log(`发送进度：${offset}/${file.size}`);
      sendNextChunk();
    };

    sendNextChunk();
  }

  private saveTrack(buffer: ArrayBuffer) {
    // 下载 WAV 文件
    const blob = new Blob([buffer], { type: 'audio/wav' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `track_${Date.now()}.wav`;
    a.click();
    URL.revokeObjectURL(url);
  }

  private onError(error: any) {
    console.error('WebSocket 错误:', error);
  }

  private onClose() {
    console.log('WebSocket 连接已关闭');
    this.ws = null;
  }

  public close() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// 使用示例
// 用户点击分离按钮时
document.getElementById('separateBtn')!.addEventListener('click', () => {
  const taskId = `task_${Date.now()}`;
  const token = localStorage.getItem('jwt_token'); // 从本地存储获取 token
  
  const separator = new AudioSeparator(taskId, token!);
});
```

### Vue 3 示例

```vue
<template>
  <div class="audio-separator">
    <input type="file" @change="handleFileSelect" accept="audio/*" />
    <button @click="startSeparation" :disabled="!audioFile || isProcessing">
      {{ isProcessing ? '分离中...' : '开始分离' }}
    </button>
    
    <div v-if="isProcessing" class="progress">
      <div>状态：{{ status }}</div>
      <div>进度：{{ progress }}%</div>
    </div>
    
    <div v-if="tracks.length > 0" class="tracks">
      <h3>分离结果</h3>
      <div v-for="(track, index) in tracks" :key="index">
        <audio :src="track.url" controls></audio>
        <span>{{ track.name }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const audioFile = ref<File | null>(null);
const isProcessing = ref(false);
const status = ref('');
const progress = ref(0);
const tracks = ref<Array<{ name: string; url: string }>>([]);
let ws: WebSocket | null = null;

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  audioFile.value = target.files?.[0] || null;
};

const startSeparation = () => {
  if (!audioFile.value) return;
  
  isProcessing.value = true;
  status.value = '正在连接服务...';
  progress.value = 0;
  tracks.value = [];
  
  const taskId = `task_${Date.now()}`;
  const token = localStorage.getItem('jwt_token');
  
  if (!token) {
    alert('请先登录');
    isProcessing.value = false;
    return;
  }
  
  // 建立 WebSocket 连接
  const url = `ws://localhost:8004/api/v1/audio/separate/ws?task_id=${taskId}&token=${token}`;
  ws = new WebSocket(url);
  ws.binaryType = 'arraybuffer';
  
  ws.onopen = () => {
    console.log('连接已建立');
    status.value = '连接成功，发送配置...';
    
    // 发送任务配置
    ws!.send(JSON.stringify({
      type: 'task_config',
      task_id: taskId,
      sample_rate: 44100,
      channels: 2,
      tracks: ['vocals', 'drums', 'bass', 'other']
    }));
  };
  
  ws.onmessage = (event) => {
    if (event.data instanceof Blob) {
      handleBinaryData(event.data);
    } else {
      const message = JSON.parse(event.data);
      handleTextMessage(message);
    }
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error);
    status.value = '连接错误';
    isProcessing.value = false;
  };
  
  ws.onclose = () => {
    console.log('连接已关闭');
    isProcessing.value = false;
  };
};

const handleTextMessage = (message: any) => {
  switch (message.type) {
    case 'ready':
      status.value = '服务端已准备，开始发送音频...';
      progress.value = 10;
      sendAudioFile();
      break;
      
    case 'result_header':
      status.value = '分离完成，正在下载音轨...';
      progress.value = 90;
      console.log('音轨数量:', message.track_count);
      break;
      
    case 'completed':
      status.value = '完成！';
      progress.value = 100;
      ws?.close();
      break;
      
    case 'error':
      status.value = `错误：${message.message}`;
      isProcessing.value = false;
      break;
  }
};

const handleBinaryData = (blob: Blob) => {
  const url = URL.createObjectURL(blob);
  tracks.value.push({
    name: `track_${tracks.value.length + 1}`,
    url: url
  });
};

const sendAudioFile = () => {
  if (!audioFile.value) return;
  
  const reader = new FileReader();
  const chunkSize = 64 * 1024;
  let offset = 0;
  
  const sendNextChunk = () => {
    if (offset >= audioFile.value!.size) {
      ws!.send(JSON.stringify({ type: 'audio_end' }));
      status.value = '音频发送完成，正在处理...';
      progress.value = 50;
      return;
    }
    
    const chunk = audioFile.value!.slice(offset, offset + chunkSize);
    reader.readAsArrayBuffer(chunk);
  };
  
  reader.onload = (e) => {
    ws!.send(e.target!.result as ArrayBuffer);
    offset += chunkSize;
    progress.value = 10 + Math.round((offset / audioFile.value!.size) * 40);
    sendNextChunk();
  };
  
  sendNextChunk();
};
</script>
```

### React 示例

```tsx
import React, { useState, useRef } from 'react';

const AudioSeparator: React.FC = () => {
  const [audioFile, setAudioFile] = useState<File | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [status, setStatus] = useState('');
  const [tracks, setTracks] = useState<Array<{ name: string; url: string }>>([]);
  const wsRef = useRef<WebSocket | null>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    setAudioFile(e.target.files?.[0] || null);
  };

  const startSeparation = () => {
    if (!audioFile) return;
    
    setIsProcessing(true);
    setStatus('正在连接服务...');
    setTracks([]);
    
    const taskId = `task_${Date.now()}`;
    const token = localStorage.getItem('jwt_token');
    
    if (!token) {
      alert('请先登录');
      setIsProcessing(false);
      return;
    }
    
    const url = `ws://localhost:8004/api/v1/audio/separate/ws?task_id=${taskId}&token=${token}`;
    const ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';
    wsRef.current = ws;
    
    ws.onopen = () => {
      setStatus('连接成功，发送配置...');
      ws.send(JSON.stringify({
        type: 'task_config',
        task_id: taskId,
        sample_rate: 44100,
        channels: 2,
        tracks: ['vocals', 'drums', 'bass', 'other']
      }));
    };
    
    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        handleBinaryData(event.data);
      } else {
        handleTextMessage(JSON.parse(event.data));
      }
    };
    
    ws.onerror = () => {
      setStatus('连接错误');
      setIsProcessing(false);
    };
    
    ws.onclose = () => {
      setIsProcessing(false);
    };
  };

  const handleTextMessage = (message: any) => {
    switch (message.type) {
      case 'ready':
        setStatus('服务端已准备，开始发送音频...');
        sendAudioFile();
        break;
      case 'result_header':
        setStatus('分离完成，正在下载音轨...');
        break;
      case 'completed':
        setStatus('完成！');
        wsRef.current?.close();
        break;
      case 'error':
        setStatus(`错误：${message.message}`);
        setIsProcessing(false);
        break;
    }
  };

  const handleBinaryData = (blob: Blob) => {
    const url = URL.createObjectURL(blob);
    setTracks(prev => [...prev, {
      name: `track_${prev.length + 1}`,
      url: url
    }]);
  };

  const sendAudioFile = () => {
    const reader = new FileReader();
    const chunkSize = 64 * 1024;
    let offset = 0;
    
    const sendNextChunk = () => {
      if (offset >= audioFile!.size) {
        wsRef.current!.send(JSON.stringify({ type: 'audio_end' }));
        setStatus('音频发送完成，正在处理...');
        return;
      }
      
      const chunk = audioFile!.slice(offset, offset + chunkSize);
      reader.readAsArrayBuffer(chunk);
    };
    
    reader.onload = (e) => {
      wsRef.current!.send(e.target!.result as ArrayBuffer);
      offset += chunkSize;
      sendNextChunk();
    };
    
    sendNextChunk();
  };

  return (
    <div>
      <input type="file" onChange={handleFileSelect} accept="audio/*" />
      <button onClick={startSeparation} disabled={!audioFile || isProcessing}>
        {isProcessing ? '分离中...' : '开始分离'}
      </button>
      {isProcessing && <div>{status}</div>}
      {tracks.map((track, i) => (
        <div key={i}>
          <audio src={track.url} controls />
          <span>{track.name}</span>
        </div>
      ))}
    </div>
  );
};

export default AudioSeparator;
```

## 📋 完整流程

1. **用户登录** → 获取 JWT Token
2. **选择音频文件** → 准备分离
3. **点击分离按钮** → 建立 WebSocket 连接
4. **发送任务配置** → 告知服务端分离参数
5. **接收 ready 消息** → 开始发送音频数据
6. **分块发送音频** → 流式传输
7. **发送 audio_end** → 通知处理
8. **接收 result_header** → 获取音轨信息
9. **接收二进制数据** → 保存各轨音频
10. **接收 completed** → 完成分离

## ⚠️ 注意事项

1. **Token 有效期**：确保 token 未过期
2. **连接复用**：一个任务一个连接，完成后关闭
3. **错误处理**：捕获并处理所有错误
4. **文件选择**：支持常见音频格式（MP3、WAV、FLAC 等）
5. **分块大小**：建议 64KB，平衡性能和内存

## 🔍 调试技巧

### 1. 检查 Token
```javascript
console.log('Token:', localStorage.getItem('jwt_token'));
```

### 2. 监听所有消息
```javascript
const originalSend = WebSocket.prototype.send;
WebSocket.prototype.send = function(data) {
  console.log('发送:', data);
  return originalSend.call(this, data);
};

ws.onmessage = (event) => {
  console.log('收到:', event.data);
  // ...
};
```

### 3. 查看连接状态
```javascript
console.log('WebSocket 状态:', ws.readyState);
// 0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED
```

## 🎉 测试步骤

1. 登录获取 token
2. 打开前端页面
3. 选择音频文件
4. 点击"分离"按钮
5. 观察控制台日志
6. 下载分离后的音轨

现在 WebSocket 接口已经支持 Token 认证，可以在前端安全使用了！🎵
