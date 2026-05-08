# WebSocket 音轨分离接口文档

## 📡 接口概述

通过 WebSocket 长连接实现音轨分离，支持流式音频数据传输，实时返回分离结果。

### 特点
- ✅ **流式传输** - 支持分块发送音频数据
- ✅ **实时反馈** - 每个阶段都有状态通知
- ✅ **高效传输** - 二进制数据直接传输，无需编码
- ✅ **多轨分离** - 支持人声、伴奏、鼓、贝斯等多轨分离

## 🔗 建立连接

### 请求

```
GET ws://localhost:8004/api/v1/audio/separate/ws?task_id={task_id}
```

**参数说明**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务唯一标识 |

### 响应

**成功**：WebSocket 连接建立

**失败**：连接关闭，返回错误消息

## 📤 消息协议

### 消息类型

所有文本消息均为 JSON 格式，包含 `type` 字段用于区分消息类型。

#### 1. 任务配置（客户端 → 服务端）

```json
{
  "type": "task_config",
  "task_id": "task_001",
  "sample_rate": 44100,
  "channels": 2,
  "tracks": ["vocals", "drums", "bass", "other"]
}
```

**参数说明**：
| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 固定为 "task_config" |
| task_id | string | 任务 ID（与 URL 参数一致） |
| sample_rate | number | 采样率（Hz） |
| channels | number | 声道数（1=单声道，2=立体声） |
| tracks | array | 需要分离的音轨列表 |

**支持的音轨**：
- `vocals` - 人声
- `drums` - 鼓
- `bass` - 贝斯
- `other` - 其他
- `piano` - 钢琴
- `guitar` - 吉他

#### 2. 音频数据（客户端 → 服务端）

**二进制消息**（WebSocket Binary Frame）

```
[Binary Data: 音频 PCM 数据或 WAV 文件]
```

**说明**：
- 可以分多次发送（流式）
- 也可以一次性发送完整文件
- 支持 PCM 原始数据或 WAV 文件格式

#### 3. 结束标记（客户端 → 服务端）

```json
{
  "type": "audio_end"
}
```

**说明**：通知服务端音频数据已发送完毕，开始处理。

### 服务端响应消息

#### 1. 准备就绪（Ready）

```json
{
  "type": "ready",
  "task_id": "task_001",
  "message": "准备接收音频数据"
}
```

#### 2. 结果头（Result Header）

```json
{
  "type": "result_header",
  "task_id": "task_001",
  "status": "success",
  "track_count": 4,
  "tracks": [
    {
      "name": "vocals",
      "format": "wav",
      "size": 10485760,
      "order": 0
    },
    {
      "name": "drums",
      "format": "wav",
      "size": 5242880,
      "order": 1
    },
    {
      "name": "bass",
      "format": "wav",
      "size": 3145728,
      "order": 2
    },
    {
      "name": "other",
      "format": "wav",
      "size": 4194304,
      "order": 3
    }
  ]
}
```

**参数说明**：
| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 固定为 "result_header" |
| task_id | string | 任务 ID |
| status | string | 状态：success/failed |
| track_count | number | 音轨数量 |
| tracks | array | 音轨信息列表 |

**tracks 数组**：
| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 音轨名称 |
| format | string | 音频格式（wav） |
| size | number | 数据大小（字节） |
| order | number | 发送顺序 |

#### 3. 音轨数据（服务端 → 客户端）

**二进制消息**（WebSocket Binary Frame）

```
[Binary Data: 单轨音频 WAV 数据]
```

**说明**：
- 按 `result_header.tracks` 中的顺序发送
- 每个音轨一个独立的二进制消息
- 发送顺序：vocals → drums → bass → other

#### 4. 完成消息（服务端 → 客户端）

```json
{
  "type": "completed",
  "task_id": "task_001",
  "message": "分离完成"
}
```

#### 5. 错误消息（双向）

```json
{
  "type": "error",
  "task_id": "task_001",
  "message": "错误描述"
}
```

## 🔄 完整通信流程

```
客户端                              服务端
  |                                   |
  |-------- WebSocket 连接 ----------->|
  |                                   |
  |------- task_config (JSON) ------->|
  |                                   |
  |<--------- ready (JSON) -----------|
  |                                   |
  |---- 音频数据 chunk1 (Binary) ---->|
  |---- 音频数据 chunk2 (Binary) ---->|
  |---- 音频数据 chunk3 (Binary) ---->|
  |                                   |
  |------- audio_end (JSON) --------->|
  |                                   |
  |     [AI 模型处理中...]             |
  |                                   |
  |<---- result_header (JSON) --------|
  |<--- vocals 音轨 (Binary) ---------|
  |<--- drums 音轨 (Binary) ----------|
  |<--- bass 音轨 (Binary) -----------|
  |<--- other 音轨 (Binary) ----------|
  |                                   |
  |<------ completed (JSON) ----------|
  |                                   |
  |<------- 关闭连接 -----------------|
```

## 💻 代码示例

### JavaScript (浏览器)

```javascript
// 1. 建立 WebSocket 连接
const ws = new WebSocket('ws://localhost:8004/api/v1/audio/separate/ws?task_id=task_001');

ws.binaryType = 'arraybuffer';

// 2. 连接打开
ws.onopen = () => {
  console.log('WebSocket 连接已建立');
  
  // 发送任务配置
  const config = {
    type: 'task_config',
    task_id: 'task_001',
    sample_rate: 44100,
    channels: 2,
    tracks: ['vocals', 'drums', 'bass', 'other']
  };
  ws.send(JSON.stringify(config));
};

// 3. 接收消息
ws.onmessage = (event) => {
  // 判断是文本还是二进制
  if (event.data instanceof Blob) {
    // 二进制数据（音轨）
    handleBinaryData(event.data);
  } else {
    // 文本消息（JSON）
    const message = JSON.parse(event.data);
    handleMessage(message);
  }
};

function handleMessage(message) {
  switch (message.type) {
    case 'ready':
      console.log('服务端已准备就绪');
      // 开始发送音频数据
      sendAudioData();
      break;
      
    case 'result_header':
      console.log('分离完成，音轨数量:', message.track_count);
      console.log('音轨列表:', message.tracks);
      // 准备接收二进制数据
      break;
      
    case 'completed':
      console.log('所有数据接收完成');
      ws.close();
      break;
      
    case 'error':
      console.error('错误:', message.message);
      ws.close();
      break;
  }
}

function handleBinaryData(blob) {
  // 接收音轨数据
  blob.arrayBuffer().then(buffer => {
    console.log('收到音轨数据，大小:', buffer.byteLength);
    // 保存为 WAV 文件
    saveToFile(buffer);
  });
}

// 4. 发送音频数据
function sendAudioData() {
  // 从文件读取或录音获取音频数据
  const audioFile = document.getElementById('audioFile').files[0];
  const reader = new FileReader();
  
  // 分块发送（每块 64KB）
  const chunkSize = 64 * 1024;
  let offset = 0;
  
  function sendNextChunk() {
    if (offset >= audioFile.size) {
      // 发送结束标记
      ws.send(JSON.stringify({ type: 'audio_end' }));
      return;
    }
    
    const chunk = audioFile.slice(offset, offset + chunkSize);
    reader.readAsArrayBuffer(chunk);
  }
  
  reader.onload = (e) => {
    ws.send(e.target.result);
    offset += chunkSize;
    console.log('发送进度:', offset, '/', audioFile.size);
    sendNextChunk();
  };
  
  sendNextChunk();
}

function saveToFile(buffer) {
  // 下载 WAV 文件
  const blob = new Blob([buffer], { type: 'audio/wav' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'track.wav';
  a.click();
  URL.revokeObjectURL(url);
}

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('WebSocket 连接已关闭');
};
```

### Python

```python
import websocket
import json
import threading

class AudioSeparator:
    def __init__(self, task_id):
        self.task_id = task_id
        self.ws = None
        self.tracks = {}
        self.current_track_index = 0
        self.track_info = []
        
    def connect(self):
        """建立 WebSocket 连接"""
        url = f"ws://localhost:8004/api/v1/audio/separate/ws?task_id={self.task_id}"
        self.ws = websocket.WebSocketApp(
            url,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        self.ws.run_forever()
        
    def on_open(self, ws):
        """连接打开"""
        print("WebSocket 连接已建立")
        
        # 发送任务配置
        config = {
            "type": "task_config",
            "task_id": self.task_id,
            "sample_rate": 44100,
            "channels": 2,
            "tracks": ["vocals", "drums", "bass", "other"]
        }
        ws.send(json.dumps(config))
        
    def on_message(self, ws, message):
        """接收消息"""
        # 判断是文本还是二进制
        if isinstance(message, str):
            # 文本消息（JSON）
            data = json.loads(message)
            self.handle_text_message(data)
        else:
            # 二进制消息（音轨数据）
            self.handle_binary_message(message)
            
    def handle_text_message(self, data):
        """处理文本消息"""
        msg_type = data.get("type")
        
        if msg_type == "ready":
            print("服务端已准备就绪")
            # 开始发送音频数据
            self.send_audio_data()
            
        elif msg_type == "result_header":
            print(f"分离完成，音轨数量：{data['track_count']}")
            self.track_info = data["tracks"]
            print("音轨列表:", self.track_info)
            
        elif msg_type == "completed":
            print("所有数据接收完成")
            self.ws.close()
            
        elif msg_type == "error":
            print(f"错误：{data['message']}")
            self.ws.close()
            
    def handle_binary_message(self, data):
        """处理二进制消息（音轨数据）"""
        if self.current_track_index < len(self.track_info):
            track_name = self.track_info[self.current_track_index]["name"]
            self.tracks[track_name] = data
            print(f"收到音轨：{track_name}, 大小：{len(data)} 字节")
            self.current_track_index += 1
            
    def send_audio_data(self):
        """发送音频数据"""
        # 读取音频文件
        with open("input.wav", "rb") as f:
            audio_data = f.read()
            
        # 分块发送（每块 64KB）
        chunk_size = 64 * 1024
        offset = 0
        
        while offset < len(audio_data):
            chunk = audio_data[offset:offset + chunk_size]
            self.ws.send(chunk, opcode=websocket.ABNF.OPCODE_BINARY)
            offset += chunk_size
            print(f"发送进度：{offset}/{len(audio_data)}")
            
        # 发送结束标记
        self.ws.send(json.dumps({"type": "audio_end"}))
        print("音频数据发送完成")
        
    def on_error(self, ws, error):
        """错误处理"""
        print(f"WebSocket 错误：{error}")
        
    def on_close(self, ws, close_status_code, close_msg):
        """连接关闭"""
        print(f"WebSocket 连接已关闭：{close_status_code} - {close_msg}")
        
    def save_tracks(self, output_dir="."):
        """保存音轨到文件"""
        import os
        os.makedirs(output_dir, exist_ok=True)
        
        for track_name, track_data in self.tracks.items():
            filename = os.path.join(output_dir, f"{track_name}.wav")
            with open(filename, "wb") as f:
                f.write(track_data)
            print(f"已保存：{filename}")

# 使用示例
if __name__ == "__main__":
    separator = AudioSeparator(task_id="task_001")
    separator.connect()
    separator.save_tracks(output_dir="./output")
```

### Go

```go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type TaskConfig struct {
	Type       string   `json:"type"`
	TaskID     string   `json:"task_id"`
	SampleRate int      `json:"sample_rate"`
	Channels   int      `json:"channels"`
	Tracks     []string `json:"tracks"`
}

type ResultHeader struct {
	Type       string `json:"type"`
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	TrackCount int    `json:"track_count"`
	Tracks     []struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Size   int    `json:"size"`
		Order  int    `json:"order"`
	} `json:"tracks"`
}

type Message struct {
	Type    string `json:"type"`
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

func main() {
	// 1. 建立 WebSocket 连接
	url := "ws://localhost:8004/api/v1/audio/separate/ws?task_id=task_001"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	fmt.Println("WebSocket 连接已建立")

	// 2. 发送任务配置
	config := TaskConfig{
		Type:       "task_config",
		TaskID:     "task_001",
		SampleRate: 44100,
		Channels:   2,
		Tracks:     []string{"vocals", "drums", "bass", "other"},
	}
	conn.WriteJSON(config)

	// 3. 读取音频文件
	audioData, err := ioutil.ReadFile("input.wav")
	if err != nil {
		log.Fatal("读取音频失败:", err)
	}

	// 4. 分块发送音频数据
	chunkSize := 64 * 1024
	for offset := 0; offset < len(audioData); offset += chunkSize {
		end := offset + chunkSize
		if end > len(audioData) {
			end = len(audioData)
		}
		chunk := audioData[offset:end]
		conn.WriteMessage(websocket.BinaryMessage, chunk)
		fmt.Printf("发送进度：%d/%d\n", end, len(audioData))
	}

	// 5. 发送结束标记
	conn.WriteJSON(map[string]string{"type": "audio_end"})
	fmt.Println("音频数据发送完成")

	// 6. 接收结果
	trackIndex := 0
	var trackInfo []struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Size   int    `json:"size"`
		Order  int    `json:"order"`
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取消息失败:", err)
			break
		}

		if messageType == websocket.BinaryMessage {
			// 接收音轨数据
			if trackIndex < len(trackInfo) {
				trackName := trackInfo[trackIndex].Name
				filename := fmt.Sprintf("%s.wav", trackName)
				ioutil.WriteFile(filename, message, 0644)
				fmt.Printf("已保存：%s (%d 字节)\n", filename, len(message))
				trackIndex++
			}
		} else {
			// 文本消息
			var msg Message
			json.Unmarshal(message, &msg)

			switch msg.Type {
			case "ready":
				fmt.Println("服务端已准备就绪")

			case "result_header":
				var result ResultHeader
				json.Unmarshal(message, &result)
				fmt.Printf("分离完成，音轨数量：%d\n", result.TrackCount)
				trackInfo = result.Tracks

			case "completed":
				fmt.Println("所有数据接收完成")
				return

			case "error":
				fmt.Printf("错误：%s\n", msg.Message)
				return
			}
		}
	}
}
```

##  使用场景

### 1. 实时音频分离
- 用户上传音频文件
- 实时显示分离进度
- 下载分离后的各轨音频

### 2. 流式处理
- 边录音边分离
- 适合直播场景
- 低延迟处理

### 3. 批量处理
- 复用 WebSocket 连接
- 连续发送多个任务
- 减少连接开销

## ⚠️ 注意事项

1. **连接超时**
   - 长时间无消息会自动断开
   - 建议设置心跳机制

2. **数据大小**
   - 音频文件较大时建议分块发送
   - 避免一次性发送超大文件

3. **错误处理**
   - 捕获所有错误并妥善处理
   - 支持重连机制

4. **资源清理**
   - 完成后及时关闭连接
   - 释放内存和文件句柄

## 🔍 调试技巧

### 1. 使用 WebSocket 调试工具
- Chrome DevTools Network 面板
- Postman WebSocket 功能
- wscat 命令行工具

### 2. 日志记录
```javascript
// 记录所有消息
ws.onmessage = (event) => {
  console.log('收到消息:', event.data);
  // ...
};

const originalSend = ws.send;
ws.send = function(data) {
  console.log('发送消息:', data);
  originalSend.call(this, data);
};
```

### 3. 性能监控
```javascript
const startTime = Date.now();

ws.onmessage = (event) => {
  const elapsed = Date.now() - startTime;
  console.log(`[${elapsed}ms] 收到消息`);
};
```

## 📊 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 连接建立时间 | < 100ms | WebSocket 握手时间 |
| 首包延迟 | < 500ms | 发送配置到收到 ready |
| 处理速度 | > 10x realtime | 处理速度是音频时长的 10 倍 |
| 传输效率 | > 90% | 有效数据占比 |

##  与 HTTP 接口对比

| 特性 | WebSocket | HTTP |
|------|-----------|------|
| 连接方式 | 长连接 | 短连接 |
| 数据传输 | 双向实时 | 单向请求 - 响应 |
| 进度反馈 | 实时 | 需轮询 |
| 适用场景 | 流式、实时 | 文件上传下载 |
| 开销 | 低（复用连接） | 高（每次新建） |

## 📚 相关资源

- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [BS-RoFormer 模型](https://github.com/karaokenerds/python-audio-separator)
