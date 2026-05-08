# BSRoFormer Python 环境安装脚本
# 使用官方 pip 方式安装 BSRoFormer 模型

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "BSRoFormer Python 环境安装脚本" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# 检查 Python 是否安装
Write-Host "[步骤 1/4] 检查 Python 环境..." -ForegroundColor Yellow
try {
    $pythonVersion = python --version 2>&1
    Write-Host "✓ Python 已安装：$pythonVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Python 未安装或未添加到 PATH" -ForegroundColor Red
    Write-Host "请先安装 Python 3.10+ : https://www.python.org/downloads/" -ForegroundColor Yellow
    exit 1
}

# 检查 Python 版本
$versionInfo = python --version 2>&1
if ($versionInfo -match "Python 3\.(\d+)") {
    $minorVersion = [int]$Matches[1]
    if ($minorVersion -lt 10) {
        Write-Host "✗ Python 版本过低，需要 3.10+，当前：$versionInfo" -ForegroundColor Red
        exit 1
    }
    Write-Host "✓ Python 版本满足要求：$versionInfo" -ForegroundColor Green
}

Write-Host ""

# 升级 pip
Write-Host "[步骤 2/4] 升级 pip..." -ForegroundColor Yellow
python -m pip install --upgrade pip
Write-Host "✓ pip 已升级" -ForegroundColor Green
Write-Host ""

# 安装 PyTorch（CUDA 版本）
Write-Host "[步骤 3/4] 安装 PyTorch（CUDA 12.0 版本）..." -ForegroundColor Yellow
Write-Host "如果安装慢，可以手动从 https://pytorch.org/ 获取安装命令" -ForegroundColor Gray
python -m pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu120
Write-Host "✓ PyTorch 已安装" -ForegroundColor Green
Write-Host ""

# 安装 BSRoFormer
Write-Host "[步骤 4/4] 安装 BSRoFormer..." -ForegroundColor Yellow
python -m pip install BS-RoFormer --upgrade
Write-Host "✓ BSRoFormer 已安装" -ForegroundColor Green
Write-Host ""

# 验证安装
Write-Host "验证安装..." -ForegroundColor Yellow
Write-Host ""

Write-Host "1. 检查 torch:" -ForegroundColor Cyan
python -c "import torch; print(f'  Torch 版本：{torch.__version__}'); print(f'  CUDA 可用：{torch.cuda.is_available()}'); print(f'  CUDA 版本：{torch.version.cuda}' if torch.cuda.is_available() else '')"

Write-Host ""
Write-Host "2. 检查 bs_roformer:" -ForegroundColor Cyan
python -c "import bs_roformer; print('  BSRoFormer 已安装 ✓')"

Write-Host ""
Write-Host "3. 测试模型加载（会自动下载预训练权重）:" -ForegroundColor Cyan
$testScript = @'
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
'@

python -c $testScript

Write-Host ""
Write-Host "=========================================" -ForegroundColor Green
Write-Host "安装完成！" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green
Write-Host ""
Write-Host "下一步:" -ForegroundColor Yellow
Write-Host "1. 启动 ai-worker 服务" -ForegroundColor White
Write-Host "2. 服务会自动使用已安装的 BSRoFormer 模型" -ForegroundColor White
Write-Host "3. 查看文档：docs/BSROFORMER_DEPLOYMENT.md" -ForegroundColor White
Write-Host ""
