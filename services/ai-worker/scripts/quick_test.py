#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
BS-RoFormer 快速测试脚本
用于验证主备模型是否正常工作

使用方法：
    python quick_test.py
"""

import os
import sys
import subprocess
import json

def run_command(cmd, description):
    """运行命令并返回结果"""
    print(f"\n{'='*60}")
    print(f"🔍 {description}")
    print(f"{'='*60}")
    print(f"命令: {' '.join(cmd)}")

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=60
        )

        if result.returncode == 0:
            print(f"✅ 成功")
            if result.stdout.strip():
                print(f"输出:\n{result.stdout}")
            return True, result.stdout
        else:
            print(f"❌ 失败 (退出码: {result.returncode})")
            if result.stderr.strip():
                print(f"错误:\n{result.stderr}")
            return False, result.stderr
    except Exception as e:
        print(f"❌ 异常: {e}")
        return False, str(e)

def main():
    print("🚀 BS-RoFormer 快速测试")
    print("=" * 60)

    results = []

    # 1. 检查 Python 环境
    success, _ = run_command(
        [sys.executable, "--version"],
        "检查 Python 版本"
    )
    results.append(("Python 环境", success))

    # 2. 检查 PyTorch
    success, output = run_command(
        [sys.executable, "-c", "import torch; print('PyTorch:', torch.__version__); print('CUDA:', torch.cuda.is_available())"],
        "检查 PyTorch"
    )
    results.append(("PyTorch", success))

    # 3. 检查 BS-RoFormer
    success, output = run_command(
        [sys.executable, "-c", "from bs_roformer import BSRoformer, MelBandRoformer; print('✅ BS-RoFormer 已安装')"],
        "检查 BS-RoFormer（主模型）"
    )
    results.append(("BS-RoFormer", success))

    # 4. 检查 Demucs
    success, output = run_command(
        [sys.executable, "-c", "import demucs; print('Demucs:', demucs.__version__)"],
        "检查 Demucs（备用模型）"
    )
    results.append(("Demucs", success))

    # 5. 检查推理脚本
    script_path = os.path.join(os.path.dirname(__file__), 'bsroformer_inference.py')
    if not os.path.exists(script_path):
        script_path = os.path.join(os.getcwd(), 'scripts', 'bsroformer_inference.py')

    if os.path.exists(script_path):
        success, _ = run_command(
            [sys.executable, script_path, "--help"],
            "检查推理脚本"
        )
        results.append(("推理脚本", success))
    else:
        print(f"\n⚠️ 推理脚本未找到: {script_path}")
        results.append(("推理脚本", False))

    # 6. 测试帮助信息
    if os.path.exists(script_path):
        success, output = run_command(
            [sys.executable, script_path],
            "显示帮助信息"
        )
        results.append(("帮助信息", success))

    # 输出总结
    print("\n" + "=" * 60)
    print("📊 测试结果总结")
    print("=" * 60)

    passed = sum(1 for _, s in results if s)
    total = len(results)

    for name, success in results:
        status = "✅ 通过" if success else "❌ 失败"
        print(f"{status} - {name}")

    print("\n" + "-" * 60)
    print(f"总计: {passed}/{total} 通过")

    if passed == total:
        print("\n🎉 所有检查通过！系统已就绪。")
        print("\n下一步:")
        print("1. 启动 Go 服务: .\\ai-worker.exe -f etc\\ai-worker.yaml")
        print("2. 使用 Apifox 测试 WebSocket 连接")
        print("3. 查看 BSROFORMER_INTEGRATION_GUIDE.md 了解详情")
        return 0
    else:
        failed_items = [name for name, s in results if not s]
        print(f"\n⚠️ 有 {total - passed} 项未通过:")
        for item in failed_items:
            print(f"  ❌ {item}")

        print("\n请查看上方日志，修复问题后重试。")
        print("\n常见问题解决方案:")
        print("- BS-RoFormer 未安装: pip install BS-RoFormer")
        print("- Demucs 未安装: pip install demucs")
        print("- PyTorch 问题: pip install torch torchaudio")

        return 1

if __name__ == "__main__":
    sys.exit(main())
