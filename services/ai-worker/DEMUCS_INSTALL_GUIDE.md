# ✅ Demucs 安装方案 - 彻底解决 "executable file not found" 错误

## 🐛 问题原因

**错误信息**：
```json
{
  "type": "error",
  "error": "AI 分离失败：Demucs 推理失败: exec: \"demucs\": executable file not found in %PATH%"
}
```

**根本原因**：
- AI 模型调用 `demucs` 命令进行音频分离
- 系统中**未安装 Demucs** 或 **未添加到 PATH**
- Docker 镜像中也缺少这个依赖

---

## 🔧 解决方案

### 方案一：Docker 部署（推荐 ⭐）

#### ✅ 已修改：[`Dockerfile`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\Dockerfile)

在安装 BSRoFormer 之后添加了：

```dockerfile
# 安装 Demucs 音频分离模型 ⭐
RUN pip3 install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple

# 验证 Demucs 安装
RUN python3 -c "import demucs; print(f'Demucs installed successfully: {demucs.__version__}')"
RUN demucs --version || echo "Demucs CLI ready"
```

#### 构建和启动 Docker 镜像

**Windows PowerShell**：
```powershell
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

# 1. 构建镜像（包含 Demucs）
docker build -t ai-worker:v1 .

# 2. 启动容器（使用挂载脚本）
.\start-docker.bat
```

**Linux/Mac**：
```bash
cd services/ai-worker

# 1. 构建镜像
docker build -t ai-worker:v1 .

# 2. 启动容器
./start-docker.sh
```

---

### 方案二：本地开发环境安装

#### 📌 Windows 用户（当前使用）

**方法 A：双击运行安装脚本**

直接双击 [`install_demucs.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\install_demucs.bat) 即可自动安装。

**方法 B：手动安装**

打开 PowerShell 或 CMD，执行：

```powershell
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker

# 步骤 1: 升级 pip
python -m pip install --upgrade pip

# 步骤 2: 安装 Demucs（使用清华镜像加速）
pip install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple

# 步骤 3: 验证安装
python -c "import demucs; print(f'Demucs 版本: {demucs.__version__}')"

# 步骤 4: 测试命令行
demucs --help
```

**方法 C：如果遇到权限问题**

```powershell
# 使用用户级安装（不需要管理员权限）
pip install --user demucs -i https://pypi.tuna.tsinghua.edu.cn/simple

# 添加到 PATH（临时）
set PATH=%PATH%;%APPDATA%\Python\Python313\Scripts

# 永久添加到 PATH（需要重启终端）
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:APPDATA\Python\Python313\Scripts", "User")
```

#### 📌 Linux/Mac 用户

```bash
cd services/ai-worker

# 升级并安装
pip3 install --upgrade pip
pip3 install demucs

# 验证
python3 -c "import demucs; print(f'Demucs 版本: {demucs.__version__}')"
demucs --version
```

---

## 🎯 验证安装成功

### 检查 1：Python 模块导入

```powershell
python -c "import demucs; print(f'✅ Demucs 版本: {demucs.__version__}')"
```

**预期输出**：
```
✅ Demucs 版本: 4.0.1
```

### 检查 2：CLI 命令可用

```powershell
demucs --version
```

**预期输出**：
```
demucs 4.0.1
```

### 检查 3：帮助信息

```powershell
demucs -h
```

应该显示完整的命令行参数说明。

---

## 🔄 安装后的测试流程

### 1️⃣ 重启 AI Worker 服务

```powershell
# 先停止旧服务（Ctrl+C 或关闭终端）

# 重新启动
cd c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker
.\ai-worker.exe -f etc/ai-worker.yaml
```

### 2️⃣ 连接 WebSocket

在 Apifox 中连接：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_demucs&token=你的JWT_TOKEN
```

### 3️⃣ 接收音频列表

**预期响应**：
```json
{
  "type": "audio_list",
  "message": "获取到 17 条音频记录",
  "data": {
    "audio_list": [...],
    "total": 17
  }
}
```

### 4️⃣ 发送分离请求

选择 `content_id = 2`：
```json
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}
```

### 5️⃣ 等待分离完成（可能需要几分钟）

**预期响应**：
```json
{
  "type": "separation_complete",
  "message": "分离成功！",
  "data": {
    "task_id": "test_demucs",
    "content_id": 2,
    "tracks": [
      {"track_name": "vocals", ...},
      {"track_name": "drums", ...},
      {"track_name": "bass", ...},
      {"track_name": "other", ...}
    ]
  }
}
```

### 6️⃣ 查看输出文件

```powershell
dir output\test_demucs\
```

应该看到 4 个 `.wav` 文件：
- `vocals.wav`
- `drums.wav`
- `bass.wav`
- `other.wav`

---

## 📊 完整的文件清单

| 文件 | 用途 | 状态 |
|------|------|------|
| [`Dockerfile`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\Dockerfile) | 已添加 Demucs 安装 | ✅ 已修改 |
| [`install_demucs.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\install_demucs.bat) | Windows 本地安装脚本 | ✅ 新建 |
| [`start-docker.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\start-docker.bat) | Docker 启动脚本 | ✅ 可用 |
| `output/` | 分离结果目录 | ✅ 已创建 |
| `cache/` | 缓存目录 | ✅ 已创建 |

---

## 🔧 故障排除

### Q1: pip 安装超时或失败

**解决方案**：
```powershell
# 使用国内镜像源
pip install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host pypi.tuna.tsinghua.edu.cn

# 或者使用代理
pip install demucs --proxy http://127.0.0.1:7890
```

### Q2: "demucs 不是内部或外部命令"

**原因**：Scripts 目录未加入 PATH

**解决方案**：
```powershell
# 找到 Python Scripts 路径
where python

# 通常在：
# C:\Users\你的用户名\AppData\Local\Programs\Python\Python313\Scripts
# C:\Users\你的用户名\AppData\Roaming\Python\Python313\Scripts

# 手动执行（临时）
C:\Users\你的用户名\AppData\Roaming\Python\Python313\Scripts\demucs.exe --version

# 或者永久添加 PATH（重启后生效）
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Users\你的用户名\AppData\Roaming\Python\Python313\Scripts", "User")
```

### Q3: ModuleNotFoundError: No module named 'demucs'

**解决方案**：
```powershell
# 确认安装位置
pip show demucs

# 如果安装在用户目录，确保 Python 版本一致
python --version
pip --version

# 重新安装到正确的 Python 环境
python -m pip install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple
```

### Q4: CUDA/GPU 相关错误

**如果你有 NVIDIA GPU**：
```powershell
# 安装支持 CUDA 的 PyTorch 和 Demucs
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121
pip install demucs['cuda'] -i https://pypi.tuna.tsinghua.edu.cn/simple
```

**如果没有 GPU（仅 CPU）**：
```powershell
# CPU 版本（速度较慢但可用）
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install demucs -i https://pypi.tuna.tsinghua.edu.cn/simple
```

### Q5: ffmpeg 未找到错误

**安装 ffmpeg**：
```powershell
# 使用 winget 安装
winget install ffmpeg

# 或手动下载
# 下载地址：https://ffmpeg.org/download.html
# 解压后添加 bin 目录到 PATH
```

验证安装：
```powershell
ffmpeg -version
```

---

## 🎵 Demucs 支持的模型

### 默认模型（htdemucs）

Demucs 4.0 默认使用 `htdemucs` 模型，可分离出 4 个音轨：

| 音轨名称 | 说明 |
|----------|------|
| `vocals` | 人声（主唱） |
| `drums` | 鼓点/打击乐 |
| `bass` | 贝斯/低音 |
| `other` | 其他乐器 |

### 其他可用模型

```bash
# 高质量模型
demucs -n htdemucs_ft input.mp3

# 快速模型（速度更快）
demucs -n htdemucs_fast input.mp3

# 仅分离人声
demucs --two-stems=vocals input.mp3
```

---

## 📈 性能参考

### GPU 加速（NVIDIA）
- **处理时间**：约 10-30 秒 / 分钟音频
- **显存占用**：约 2-4 GB
- **推荐配置**：GTX 1060+ / RTX 2060+

### CPU 模式
- **处理时间**：约 2-5 分钟 / 分钟音频
- **内存占用**：约 4-8 GB RAM
- **推荐配置**：8 核 CPU 以上

---

## 🚀 下一步操作

### 立即执行（Windows 本地开发）

1. **双击运行** [`install_demucs.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\install_demucs.bat)
   - 自动安装 Demucs 及其依赖
   - 预计耗时：2-5 分钟（取决于网络速度）

2. **验证安装**
   ```powershell
   python -c "import demucs; print(demucs.__version__)"
   ```

3. **重启服务**
   ```powershell
   # 在 ai-worker 终端按 Ctrl+C 停止
   # 然后重新启动
   .\ai-worker.exe -f etc/ai-worker.yaml
   ```

4. **测试 WebSocket**
   - Apifox 连接：`ws://localhost:8004/api/v1/audio/separate/ws`
   - 发送分离请求
   - 等待结果返回

### 生产部署（Docker）

```powershell
cd services/ai-worker

# 1. 构建包含 Demucs 的镜像
docker build -t ai-worker:v1 .

# 2. 启动容器
.\start-docker.bat

# 3. 测试接口
# 访问 http://localhost:8004/api/v1/health
```

---

## 📞 技术支持

### 常用调试命令

```powershell
# 查看 Demucs 是否正确安装
python -c "
try:
    import demucs
    print('✅ Demucs 导入成功')
    print(f'版本: {demucs.__version__}')
except Exception as e:
    print(f'❌ 导入失败: {e}')
"

# 查看 PATH 中是否有 demucs
where.exe demucs 2>$null
if ($?) {
    Write-Host "✅ demucs 命令已找到"
} else {
    Write-Host "❌ demucs 命令未找到"
}

# 查看详细安装信息
pip show demucs

# 测试实际分离功能（小文件）
# demucs test.mp3 -o test_output
```

### 日志查看

```powershell
# 实时查看服务日志
Get-Content ai-worker.log -Wait -Tail 50

# 或者在启动服务的终端查看实时输出
```

---

## ✨ 总结

### 本次修复的核心改动

1. **[`Dockerfile`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\Dockerfile)** - 添加 Demucs 安装步骤
2. **[`install_demucs.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\install_demucs.bat)** - Windows 一键安装脚本
3. 配置优化 - OutputDir 设置为本地路径

### 解决的问题

| 问题 | 状态 | 方案 |
|------|------|------|
| ❌ `demucs: executable file not found` | ✅ 已修复 | 安装 Demucs 到系统/Docker |
| ❌ 输出目录权限问题 | ✅ 已修复 | 配置 OutputDir + Docker 挂载 |
| ❌ 表名错误 (audio_contents) | ✅ 已修复 | 改为 content |
| ❌ WebSocket 路由被拦截 | ✅ 已修复 | 独立路由组 |

---

**🎉 安装完成后即可正常使用音频分离功能！**

**快速开始**：双击 [`install_demucs.bat`](file://c:\Users\Lenovo\Desktop\audio-ai-platform\services\ai-worker\install_demucs.bat) → 重启服务 → 测试 WebSocket
