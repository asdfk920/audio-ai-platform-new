# BSRoFormer 快速开始指南

## 5 分钟快速安装

### 方式 1：一键安装脚本（推荐）

#### Windows

```powershell
# 打开 PowerShell，进入 ai-worker 目录
cd services/ai-worker

# 运行安装脚本
.\install_bsroformer.ps1
```

#### Linux/Mac

```bash
# 进入 ai-worker 目录
cd services/ai-worker

# 运行安装脚本
chmod +x install_bsroformer.sh
./install_bsroformer.sh
```

### 方式 2：手动安装

```bash
# 1. 安装 Python 3.10+
# 访问 https://www.python.org/downloads/ 下载安装

# 2. 打开终端/命令提示符，执行以下命令
pip install --upgrade pip
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu120
pip install BS-RoFormer
```

## 验证安装

```bash
# 执行快速测试
python -c "from bs_roformer import BSRoFormer; print('✓ BSRoFormer 已安装')"
```

## 测试推理

创建一个测试脚本 `test_bsroformer.py`：

```python
import torch
from bs_roformer import BSRoFormer
import time

print("BSRoFormer 推理测试")
print("=" * 50)

# 1. 加载模型
print("正在加载模型...")
model = BSRoFormer.from_pretrained("lucidrains/bs-roformer-mel")
model.eval()

# 2. 移动到设备
device = "cuda" if torch.cuda.is_available() else "cpu"
model = model.to(device)
print(f"✓ 模型已加载到：{device}")

# 3. 创建测试音频（10 秒立体声）
print("\n创建测试音频...")
audio = torch.randn(1, 2, 44100 * 10).to(device)
print(f"✓ 音频形状：{audio.shape}")

# 4. 执行推理
print("\n执行推理...")
start = time.time()
with torch.no_grad():
    separated = model(audio)
elapsed = time.time() - start

# 5. 显示结果
print("\n" + "=" * 50)
print("推理结果:")
print(f"  推理耗时：{elapsed:.3f}秒")
print(f"  平均延迟：{elapsed*1000:.1f}ms")
print(f"  输出音轨数：{len(separated)}")
print(f"  目标延迟：<100ms")

if elapsed * 1000 < 100:
    print("\n✓ 性能达标！延迟 < 100ms")
else:
    print(f"\n⚠ 延迟稍高，可以尝试:")
    print("   - 启用 FP16: model.half()")
    print("   - 减小音频长度")
    print("   - 使用更短的 segment")

print("=" * 50)
```

运行测试：

```bash
python test_bsroformer.py
```

## 启动 ai-worker 服务

安装完成后，启动 ai-worker 服务：

```bash
# 编译服务
cd services/ai-worker
go build -o ai-worker.exe .

# 启动服务
./ai-worker.exe -f etc/ai-worker.yaml
```

服务会：
1. 自动检测 Python 环境
2. 自动加载已安装的 BSRoFormer 模型
3. 开始处理推理请求

## 常见问题

### Q: Python 版本过低

**错误**: `Python 3.10+ is required`

**解决**: 
- 从 https://www.python.org/downloads/ 下载 Python 3.10 或更高版本
- 安装时勾选 "Add Python to PATH"

### Q: CUDA 不可用

**错误**: `CUDA is not available`

**解决**:
- 确保已安装 NVIDIA 驱动
- 确保安装了 CUDA 版本的 PyTorch
- 检查命令：`nvidia-smi`

### Q: 模型下载慢

**解决**: 使用国内镜像

```bash
pip install BS-RoFormer -i https://pypi.tuna.tsinghua.edu.cn/simple
```

### Q: 延迟超过 100ms

**优化建议**:
1. 启用 FP16（半精度）:
   ```python
   model = model.half()  # 转换为 FP16
   audio = audio.half()
   ```

2. 减小音频长度:
   ```python
   audio = torch.randn(1, 2, 44100 * 5).to(device)  # 5 秒而不是 10 秒
   ```

3. 使用 GPU:
   ```python
   device = "cuda"  # 确保使用 GPU
   ```

## 下一步

- 查看完整文档：[`docs/BSROFORMER_DEPLOYMENT.md`](docs/BSROFORMER_DEPLOYMENT.md)
- 查看 API 文档：[`../../ai-inference-service/INFERENCE_API_DOCS.md`](../../ai-inference-service/INFERENCE_API_DOCS.md)
- 查看性能优化指南：[`docs/PERFORMANCE_OPTIMIZATION.md`](docs/PERFORMANCE_OPTIMIZATION.md)

## 技术支持

遇到问题？
1. 检查日志：`ai-worker.log`
2. 查看常见问题：[`docs/BSROFORMER_DEPLOYMENT.md`](docs/BSROFORMER_DEPLOYMENT.md#常见问题)
3. 提交 Issue
