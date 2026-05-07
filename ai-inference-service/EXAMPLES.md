# BSRoformer SCNet 使用示例

## 快速开始

### 1. 部署服务

**Windows:**
```bash
# 运行一键部署脚本
deploy.bat
```

**Linux/Mac:**
```bash
# 构建并启动
docker-compose build
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 2. 验证服务

```bash
# 健康检查
curl http://localhost:8004/health

# 预期响应
{
  "status": "healthy",
  "model_name": "bsroformer_scnet",
  "device": "cuda:0",
  "gpu_available": true
}
```

### 3. 使用 Python 测试

```bash
# 准备测试音频（放在 data/input/test.mp3）
python test_api.py
```

## API 使用示例

### 方法 1: 使用 cURL

```bash
# 音轨分离
curl -X POST http://localhost:8004/api/v1/separate \
  -F "file=@song.mp3" \
  -H "Content-Type: multipart/form-data"

# 下载结果
curl http://localhost:8004/api/v1/download/output/song_vocals.wav \
  -o vocals.wav

curl http://localhost:8004/api/v1/download/output/song_instrumental.wav \
  -o instrumental.wav
```

### 方法 2: 使用 Python

```python
import requests

# 1. 上传音频进行分离
def separate_audio(file_path):
    url = "http://localhost:8004/api/v1/separate"
    
    with open(file_path, "rb") as f:
        files = {"file": f}
        response = requests.post(url, files=files)
    
    if response.status_code == 200:
        result = response.json()
        print(f"分离成功！耗时：{result['processing_time']:.2f}秒")
        return result["output_files"]
    else:
        print(f"失败：{response.text}")
        return None

# 2. 下载分离结果
def download_file(file_path, save_path):
    url = f"http://localhost:8004/api/v1/download/{file_path}"
    
    response = requests.get(url)
    with open(save_path, "wb") as f:
        f.write(response.content)
    
    print(f"已保存：{save_path}")

# 使用示例
if __name__ == "__main__":
    # 分离音频
    files = separate_audio("song.mp3")
    
    if files:
        # 下载人声
        download_file(files["vocals"], "vocals_output.wav")
        
        # 下载伴奏
        download_file(files["instrumental"], "instrumental_output.wav")
```

### 方法 3: 使用 JavaScript/Node.js

```javascript
const axios = require('axios');
const fs = require('fs');
const FormData = require('form-data');

async function separateAudio(filePath) {
    const form = new FormData();
    form.append('file', fs.createReadStream(filePath));
    
    try {
        // 1. 分离音频
        const response = await axios.post(
            'http://localhost:8004/api/v1/separate',
            form,
            { headers: form.getHeaders() }
        );
        
        console.log('分离成功:', response.data);
        
        // 2. 下载结果
        const vocals = await axios.get(
            `http://localhost:8004${response.data.output_files.vocals}`,
            { responseType: 'stream' }
        );
        
        vocals.data.pipe(fs.createWriteStream('vocals.wav'));
        
    } catch (error) {
        console.error('失败:', error.message);
    }
}

// 使用示例
separateAudio('song.mp3');
```

## 批量处理示例

### Python 批量处理

```python
import requests
from pathlib import Path
import time

def batch_separate(input_dir, output_dir):
    """批量处理音频文件"""
    
    # 获取所有音频文件
    audio_files = list(Path(input_dir).glob("*.mp3"))
    audio_files.extend(Path(input_dir).glob("*.wav"))
    
    print(f"找到 {len(audio_files)} 个音频文件")
    
    for i, audio_file in enumerate(audio_files, 1):
        print(f"\n[{i}/{len(audio_files)}] 处理：{audio_file.name}")
        
        try:
            # 上传分离
            with open(audio_file, "rb") as f:
                response = requests.post(
                    "http://localhost:8004/api/v1/separate",
                    files={"file": f}
                )
            
            if response.status_code == 200:
                result = response.json()
                print(f"✓ 成功，耗时：{result['processing_time']:.2f}秒")
                
                # 下载结果
                for stem, file_path in result["output_files"].items():
                    download_url = f"http://localhost:8004/api/v1/download/{file_path}"
                    resp = requests.get(download_url)
                    
                    save_path = Path(output_dir) / f"{audio_file.stem}_{stem}.wav"
                    with open(save_path, "wb") as out_f:
                        out_f.write(resp.content)
                    
                    print(f"  - 保存：{save_path}")
            else:
                print(f"✗ 失败：{response.text}")
        
        except Exception as e:
            print(f"✗ 错误：{e}")
        
        # 避免请求过快
        time.sleep(0.5)

# 使用示例
batch_separate("input_songs", "output_songs")
```

## 性能优化

### 1. 调整批处理大小

编辑 `.env` 文件：
```env
# 增加批处理大小（需要更多 GPU 内存）
MAX_BATCH_SIZE=4

# 增加并发数
MAX_CONCURRENT=5
```

### 2. 使用更快的 GPU

```bash
# 在 docker-compose.yml 中指定 GPU
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: all  # 使用所有 GPU
          capabilities: [gpu]
```

### 3. 启用模型优化

在 `src/model_loader.py` 中启用 TensorRT：
```python
# 启用混合精度
with torch.cuda.amp.autocast():
    output = model(input)
```

## 故障排查

### 查看服务日志

```bash
# 实时日志
docker-compose logs -f

# 最近 100 行
docker-compose logs --tail=100

# 特定服务
docker-compose logs inference
```

### 重启服务

```bash
# 重启
docker-compose restart

# 完全重建
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### 检查 GPU 状态

```bash
# 在容器内检查
docker exec -it inference-server nvidia-smi
```

## 集成到现有系统

### FastAPI 客户端

```python
from fastapi import FastAPI, UploadFile
import requests

app = FastAPI()

SEPARATION_SERVICE_URL = "http://localhost:8004"

@app.post("/process-audio")
async def process_audio(file: UploadFile):
    # 转发到分离服务
    files = {"file": await file.read()}
    response = requests.post(
        f"{SEPARATION_SERVICE_URL}/api/v1/separate",
        files=files
    )
    
    return response.json()
```

### Docker Compose 集成

```yaml
version: '3.8'

services:
  # 你的主应用
  web-app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - inference
  
  # 音轨分离服务
  inference:
    image: ai-inference:latest
    ports:
      - "8004:8004"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

## 支持的音频格式

| 格式 | 扩展名 | 说明 |
|------|--------|------|
| MP3 | .mp3 | 最常用，压缩格式 |
| WAV | .wav | 无损格式，推荐 |
| FLAC | .flac | 无损压缩 |
| M4A | .m4a | Apple 格式 |
| OGG | .ogg | 开源格式 |

## 输出说明

分离后会生成以下文件：

- `{filename}_vocals.wav` - 人声音轨
- `{filename}_instrumental.wav` - 伴奏音轨
- `{filename}_other.wav` - 其他音轨（如果有）

所有输出文件均为 WAV 格式，44.1kHz 采样率，16 位深度。
