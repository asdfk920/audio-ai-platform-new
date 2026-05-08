#!/bin/bash
# BSRoFormer Python 环境安装脚本
# 使用官方 pip 方式安装 BSRoFormer 模型

echo "========================================="
echo "BSRoFormer Python 环境安装脚本"
echo "========================================="
echo ""

# 检查 Python 是否安装
echo "[步骤 1/4] 检查 Python 环境..."
if command -v python3 &> /dev/null; then
    PYTHON_CMD="python3"
elif command -v python &> /dev/null; then
    PYTHON_CMD="python"
else
    echo "✗ Python 未安装或未添加到 PATH"
    echo "请先安装 Python 3.10+"
    exit 1
fi

PYTHON_VERSION=$($PYTHON_CMD --version 2>&1)
echo "✓ Python 已安装：$PYTHON_VERSION"

# 检查 Python 版本
if [[ $PYTHON_VERSION =~ "Python 3."([0-9]+) ]]; then
    MINOR_VERSION=${BASH_REMATCH[1]}
    if [ $MINOR_VERSION -lt 10 ]; then
        echo "✗ Python 版本过低，需要 3.10+，当前：$PYTHON_VERSION"
        exit 1
    fi
    echo "✓ Python 版本满足要求：$PYTHON_VERSION"
fi

echo ""

# 升级 pip
echo "[步骤 2/4] 升级 pip..."
$PYTHON_CMD -m pip install --upgrade pip
echo "✓ pip 已升级"
echo ""

# 安装 PyTorch（CUDA 版本）
echo "[步骤 3/4] 安装 PyTorch（CUDA 12.0 版本）..."
echo "如果安装慢，可以手动从 https://pytorch.org/ 获取安装命令"
$PYTHON_CMD -m pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu120
echo "✓ PyTorch 已安装"
echo ""

# 安装 BSRoFormer
echo "[步骤 4/4] 安装 BSRoFormer..."
$PYTHON_CMD -m pip install BS-RoFormer --upgrade
echo "✓ BSRoFormer 已安装"
echo ""

# 验证安装
echo "验证安装..."
echo ""

echo "1. 检查 torch:"
$PYTHON_CMD -c "import torch; print(f'  Torch 版本：{torch.__version__}'); print(f'  CUDA 可用：{torch.cuda.is_available()}'); print(f'  CUDA 版本：{torch.version.cuda}' if torch.cuda.is_available() else '')"

echo ""
echo "2. 检查 bs_roformer:"
$PYTHON_CMD -c "import bs_roformer; print('  BSRoFormer 已安装 ✓')"

echo ""
echo "3. 测试模型加载（会自动下载预训练权重）:"
$PYTHON_CMD << 'EOF'
import torch
from bs_roformer import BSRoFormer
import time

print("  正在加载模型...")
model = BSRoFormer.from_pretrained("lucidrains/bs-roformer-mel")
model.eval()

# 移动到设备
device = "cuda" if torch.cuda.is_available() else "cpu"
model = model.to(device)
print(f"  模型已加载到：{device}")

# 测试推理
print("  执行测试推理...")
audio = torch.randn(1, 2, 44100 * 5).to(device)  # 5 秒音频
start = time.time()
with torch.no_grad():
    separated = model(audio)
elapsed = time.time() - start

print(f"  ✓ 测试推理完成")
print(f"  推理耗时：{elapsed:.3f}秒")
print(f"  输出音轨数：{len(separated)}")
print(f"  平均延迟：{elapsed*1000:.1f}ms")
EOF

echo ""
echo "========================================="
echo "安装完成！"
echo "========================================="
echo ""
echo "下一步:"
echo "1. 启动 ai-worker 服务"
echo "2. 服务会自动使用已安装的 BSRoFormer 模型"
echo "3. 查看文档：docs/BSROFORMER_DEPLOYMENT.md"
echo ""
