# 🔧 Demucs 推理失败修复指南（exit status 2）

## 🎯 问题描述

**错误信息**：
```json
{
  "type": "error",
  "error": "AI 分离失败：Demucs 推理失败: exit status 2"
}
```

**发生时间**：2026-05-09
**影响范围**：所有音频分离任务无法完成

---

## ✅ 诊断结果

### **已完成检查**

| 检查项 | 状态 | 详情 |
|--------|------|------|
| Demucs 安装 | ✅ 成功 | 版本 4.0.1 |
| Python 环境 | ✅ 正常 | Python 3.13.9 |
| PyTorch | ✅ 已安装 | 2.11.0+cpu |
| CUDA 支持 | ⚠️ CPU 模式 | `torch.cuda.is_available() = False` |
| 预训练模型 | ✅ 已下载 | htdemucs (80.2 MB) |
| OpenMP 冲突 | ✅ 已修复 | 环境变量已设置 |

### **发现的问题**

#### ❌ **问题 1：缺少 `torchcodec` 依赖包**（主要原因）

**错误详情**：
```
ImportError: TorchCodec is required for load_with_torchcodec.
Please install torchcodec to use this function.
```

**原因分析**：
- torchaudio ≥ 2.1.0 需要 `torchcodec` 来加载音频文件
- Demucs 4.x 依赖最新版 torchaudio
- pip 安装 demucs 时未自动安装此依赖

---

#### ❌ **问题 2：Go 代码参数格式错误**（次要原因）

**原始代码** ([inference.go#L48-L53](file:///d:/audio-ai-platform/services/ai-worker/internal/model/inference.go#L48-L53))：
```go
args := []string{
    "-n", "htdemucs",
    "--two-stems=vocals",      // ❌ 错误格式
    "--segment=10",            // ❌ 错误格式
    "--overlap=0.25",          // ❌ 错误格式
    fmt.Sprintf("-o%s", outputDir),  // ❌ 缺少空格
}
```

**修复后代码**：
```go
args := []string{
    "-n", "htdemucs",
    "--two-stems", "vocals",   // ✅ 正确格式
    "--segment", "10",         // ✅ 正确格式
    "--overlap", "0.25",       // ✅ 正确格式
    "-o", outputDir,           // ✅ 正确格式
}
```

**Demucs 4.x 参数规范**：
- 所有参数使用空格分隔值，不使用等号
- 例如：`--two-stems vocals` 而不是 `--two-stems=vocals`

---

## 🛠️ 完整修复步骤

### **步骤 1：安装缺失的依赖包** ⭐ 最重要

在 PowerShell 中执行：

```bash
# 1. 安装 torchcodec
pip install torchcodec

# 2. 验证安装成功
pip show torchcodec

# 预期输出：
# Name: torchcodec
# Version: x.x.x
# Summary: ...
# ...
```

**如果网络慢，使用国内镜像源**：
```bash
pip install torchcodec -i https://pypi.tuna.tsinghua.edu.cn/simple
```

---

### **步骤 2：验证 Demucs 能正常运行**

```bash
# 设置环境变量（解决 OpenMP 冲突）
$env:KMP_DUPLICATE_LIB_OK='TRUE'

# 测试运行 Demucs（使用之前创建的测试文件）
demucs -n htdemucs -d cpu --two-stems vocals `
    'd:\audio-ai-platform\services\ai-worker\test_audio.wav' `
    -o 'd:\audio-ai-platform\services\ai-worker\test_output'
```

**预期输出**：
```
Selected model is a bag of 1 models.
You will see that many progress bars per track.
Separated tracks will be stored in D:\...\test_output\htdemucs

Separating track test_audio.wav
100%|██████████| xxx/xxx [xx:xx<00:00, xxxMB/s]  ← 成功！
```

**预期输出文件**：
```
test_output/htdemucs/test_audio/
├── vocals.wav      # 人声轨道 (约 2-3 MB)
└── no_vocals.wav   # 伴奏轨道 (约 2-3 MB)
```

**验证文件存在**：
```bash
ls 'd:\audio-ai-platform\services\ai-worker\test_output\htdemucs\test_audio'
```

---

### **步骤 3：重启 Go 后端服务**

Go 代码已经修复并重新编译 ✅

```bash
cd d:\audio-ai-platform\services\ai-worker

# 停止旧服务（如果在运行）
# Ctrl+C 或关闭终端

# 启动新服务
.\ai-worker.exe -f etc\ai-worker.yaml
```

**启动日志应显示**：
```
[AI Service] 初始化完成
[AI Service] Demucs 模型就绪
[Server] Starting HTTP server on :8004
✅ 服务启动成功
```

---

### **步骤 4：测试完整流程**

使用 Apifox 连接 WebSocket：

**连接地址**：
```
ws://localhost:8004/api/v1/audio/separate/ws?task_id=test_fix_001&token=YOUR_TOKEN
```

**发送消息**：
```json
{
  "type": "separate",
  "data": {
    "content_id": 2
  }
}
```

**预期返回**：
```json
{
  "type": "separation_complete",
  "message": "分离成功！",
  "data": {
    "task_id": "test_fix_001",
    "content_id": 2,
    "tracks": [
      {"id": 1, "track_name": "vocals", "track_url": "/static/tracks/test_fix_001_vocals.wav"},
      {"id": 2, "track_name": "no_vocals", "track_url": "/static/tracks/test_fix_001_no_vocals.wav"}
    ]
  }
}
```

**服务端日志应显示**：
```
[WebSocket] 🎵 开始执行音频分离任务
[WebSocket] URL 已清理: 原始='\r\nhttps://...' → 清理后='https://...'
[WebSocket] 创建临时目录: ./temp/task_test_fix_001_xxx
[WebSocket] 📥 开始下载音频文件
[WebSocket] ✅ 文件下载完成: ./temp/task_test_fix_001_xxx/input.mp3 (xxxx KB)
[WebSocket] ✅ 音频文件已准备
[Demucs] 执行命令: demucs [-n htdemucs --two-stems vocals --segment 10 ...]
[Demucs] 推理完成: task_id=test_fix_001, tracks=2, time=xxs
[WebSocket] ✅ AI 分离完成
[WebSocket] 🎉 任务完成！task_id=test_fix_001
[WebSocket] 🧹 任务结束，清理临时目录...
[WebSocket] ✅ 临时目录已清理
```

---

## 📊 修改清单

### **Python 依赖**
| 包名 | 操作 | 原因 |
|------|------|------|
| `torchcodec` | **新增安装** | torchaudio 加载音频必需 |

### **Go 代码修改**
| 文件 | 函数 | 修改内容 |
|------|------|---------|
| [inference.go](file:///d:/audio-ai-platform/services/ai-worker/internal/model/inference.go) | `runDemucsInference()` | 修复 Demucs 参数格式 |

**具体改动**（[第 47-54 行](file:///d:/audio-ai-platform/services/ai-worker/internal/model/inference.go#L47-L54)）：
```diff
  args := []string{
      "-n", "htdemucs",
-     "--two-stems=vocals",
-     "--segment=10",
-     "--overlap=0.25",
-     fmt.Sprintf("-o%s", outputDir),
+     "--two-stems", "vocals",
+     "--segment", "10",
+     "--overlap", "0.25",
+     "-o", outputDir,
  }
```

---

## 🔍 故障排查

### **如果仍然报错 exit status 2**

#### **情况 1：torchcodec 未正确安装**

**症状**：
```
ImportError: No module named 'torchcodec'
```

**解决方法**：
```bash
# 确认 Python 路径
where python
which python  # Linux/macOS

# 使用正确的 pip 安装
python -m pip install torchcodec

# 或者指定完整路径
C:\Users\Lenovo\anaconda3\Scripts\pip.exe install torchcodec
```

---

#### **情况 2：多个 Python 版本冲突**

**症状**：
```
pip show torchcodec  # 显示已安装
python -c "import torchcodec"  # 报 ModuleNotFoundError
```

**原因**：pip 和 python 不是同一个 Python 解释器

**解决方法**：
```bash
# 查看当前使用的 Python
where python
where pip

# 统一使用 anaconda 的 Python
C:\Users\Lenovo\anaconda3\python.exe -m pip install torchcodec
```

---

#### **情况 3：权限问题**

**症状**：
```
PermissionError: [Errno 13] Permission denied: '...\\torchcodec\\...'
```

**解决方法**：
```bash
# 以管理员身份运行 PowerShell
# 右键点击 PowerShell → 以管理员身份运行

# 然后重新执行安装
pip install torchcodec
```

---

#### **情况 4：网络问题导致下载失败**

**症状**：
```
ERROR: Could not find a version that satisfies the requirement torchcodec
```

**解决方法**：
```bash
# 使用国内镜像源
pip install torchcodec -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host pypi.tuna.tsinghua.edu.cn

# 或者使用其他镜像
pip install torchcodec -i https://mirrors.aliyun.com/pypi/simple/
```

---

### **如果出现其他 exit code**

#### **exit status 1** - 一般性错误

可能原因：
- 输入文件不存在或损坏
- 输出目录无写入权限
- 内存不足

**排查方法**：
```bash
# 手动测试
$env:KMP_DUPLICATE_LIB_OK='TRUE'
demucs -n htdemucs -d cpu your_file.mp3 -o test_out

# 查看详细错误信息
demucs -v -n htdemucs your_file.mp3 -o test_out
```

---

#### **exit status 137** - 被 OOM Killer 杀死

**原因**：内存不足（Out of Memory）

**解决方法**：
```bash
# 减小 segment 大小（在 inference.go 中修改）
"--segment", "5",  # 从 10 改为 5

# 或者使用更小的模型
"-n", "hdemucs",  # 比 htdemucs 更小

# 重启服务后生效
```

---

## ⚠️ 注意事项

### **1. CPU vs GPU 性能差异**

当前环境配置：
- **模式**：CPU（无 CUDA）
- **处理速度**：约比 GPU 慢 5-10 倍
- **内存占用**：较低（适合测试）

**预估处理时间**（3 分钟 MP3 文件）：
- CPU 模式：2-5 分钟
- GPU 模式：10-30 秒

**如需启用 GPU**：
1. 安装 NVIDIA 显卡驱动
2. 安装 CUDA Toolkit
3. 安装 GPU 版 PyTorch：
   ```bash
   pip uninstall torch torchaudio
   pip install torch torchaudio --index-url https://download.pytorch.org/whl/cu118
   ```

---

### **2. OpenMP 环境变量**

**必须设置**（已在 Go 代码中自动添加）：
```go
cmd.Env = append(os.Environ(), "KMP_DUPLICATE_LIB_OK=TRUE")
```

**手动运行时也需要设置**：
```powershell
$env:KMP_DUPLICATE_LIB_OK='TRUE'
demucs ...
```

---

### **3. 模型缓存位置**

**Windows**：
```
C:\Users\<用户名>\.cache\torch\hub\checkpoints\
```

**模型文件**：
```
955717e8-8726e21a.th  (80.2 MB)  ← htdemucs 模型
```

**如需重新下载模型**：
```bash
# 删除旧模型
Remove-Item "$env:USERPROFILE\.cache\torch\hub\checkpoints\955717e8*"

# 下次运行时自动下载
demucs -n htdemucs test.mp3 -o out
```

---

## 📈 性能优化建议

### **1. 减少 CPU 占用**

编辑 [inference.go](file:///d:/audio-ai-platform/services/ai-worker/internal/model/inference.go)，调整参数：

```go
args := []string{
    "-n", "htdemucs",
    "--two-stems", "vocals",
    "--segment", "5",        // ↓ 从 10 减少到 5（更快但质量略降）
    "--overlap", "0.25",     // 保持不变
    "-o", outputDir,
    "-j", "2",              // 新增：限制 CPU 核心数
}
```

---

### **2. 使用更小模型（快速测试）**

```go
// 替换 htdemucs 为 hdemucs（快 2-3 倍，质量略低）
"-n", "hdemucs",  // 轻量级模型
```

---

### **3. 批量处理优化**

对于大量文件，考虑：
- 使用队列系统（RabbitMQ、Redis）
- 限制并发数（避免 OOM）
- 监控系统资源使用

---

## ✅ 验证清单

修复完成后，请逐项确认：

- [ ] `pip install torchcodec` 执行成功
- [ ] `python -c "import torchcodec"` 无报错
- [ ] 手动运行 `demucs` 命令成功生成输出文件
- [ ] Go 后端编译成功（`go build` 无错误）
- [ ] 服务启动正常（端口 8004 可访问）
- [ ] WebSocket 连接成功
- [ ] 发送分离请求后收到 `separation_complete` 消息
- [ ] 生成的音轨文件可播放
- [ ] 临时目录 `./temp/` 已自动清理

---

## 🆘 如果仍然有问题

### **收集诊断信息**

请提供以下信息以便进一步排查：

```bash
# 1. 系统信息
systeminfo | Select-Object OS*, TotalPhysicalMemory

# 2. Python 环境
python --version
pip list | Select-String "torch|demucs"

# 3. Demucs 详细日志
$env:KMP_DUPLICATE_LIB_OK='TRUE'
demucs -v -n htdemucs test_audio.wav -o debug_output 2>&1 | Out-File demucs_debug.log

# 4. 查看 demucs_debug.log 内容
Get-Content demucs_debug.log -Tail 50
```

---

## 📞 快速参考

### **常用命令速查**

```bash
# 检查 Demucs 版本
demucs --help  # 4.x 不支持 --version

# 设置环境变量
$env:KMP_DUPLICATE_LIB_OK='TRUE'

# 运行分离（CPU）
demucs -n htdemucs -d cpu --two-stems vocals input.mp3 -o output

# 运行分离（GPU）
demucs -n htdemucs -d cuda:0 --two-stems vocals input.mp3 -o output

# 查看输出文件
ls output\htdemucs\input\
```

### **关键文件路径**

| 文件 | 路径 |
|------|------|
| Go 后端主程序 | `services/ai-worker/ai-worker.exe` |
| 配置文件 | `services/ai-worker/etc/ai-worker.yaml` |
| 推理逻辑 | `services/ai-worker/internal/model/inference.go` |
| WebSocket 处理 | `services/ai-worker/internal/handler/audio_separate_ws_handler.go` |
| 模型缓存 | `C:\Users\Lenovo\.cache\torch\hub\checkpoints\` |
| 临时文件 | `services/ai-worker/temp/` |
| 输出文件 | `services/ai-worker/output/` |

---

## 🎉 总结

### **根本原因**
1. ❌ **主要**：缺少 `torchcodec` 依赖包
2. ❌ **次要**：Go 代码中 Demucs 参数格式错误

### **修复措施**
1. ✅ 安装 `torchcodec` 包
2. ✅ 修复 Go 代码参数格式（等号 → 空格分隔）
3. ✅ 重新编译 Go 程序

### **预期效果**
- ✅ Demucs 可以正常运行
- ✅ 音频分离功能恢复正常
- ✅ 不再出现 exit status 2 错误

---

**最后更新**: 2026-05-09
**修复版本**: AI Worker v1.1
**适用环境**: Windows + Python 3.13 + Demucs 4.0.1 + PyTorch 2.11 (CPU)

🚀 **现在请按照步骤操作，问题应该可以完全解决！**
