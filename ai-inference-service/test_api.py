"""
BSRoformer SCNet 音轨分离服务测试脚本
"""

import requests
import time
from pathlib import Path


BASE_URL = "http://localhost:8004"


def test_health():
    """测试健康检查"""
    print("=" * 50)
    print("测试 1: 健康检查")
    print("=" * 50)
    
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=10)
        response.raise_for_status()
        
        data = response.json()
        print(f"✓ 状态：{data['status']}")
        print(f"✓ 模型：{data['model_name']}")
        print(f"✓ 设备：{data['device']}")
        print(f"✓ GPU 可用：{data['gpu_available']}")
        print()
        
        return True
    except Exception as e:
        print(f"✗ 健康检查失败：{e}")
        return False


def test_separate(audio_file: str):
    """测试音轨分离"""
    print("=" * 50)
    print("测试 2: 音轨分离")
    print("=" * 50)
    
    if not Path(audio_file).exists():
        print(f"✗ 文件不存在：{audio_file}")
        return False
    
    try:
        print(f"上传文件：{audio_file}")
        
        with open(audio_file, "rb") as f:
            files = {"file": f}
            response = requests.post(
                f"{BASE_URL}/api/v1/separate",
                files=files,
                timeout=300
            )
        
        response.raise_for_status()
        data = response.json()
        
        print(f"✓ 分离成功")
        print(f"✓ 耗时：{data['processing_time']:.2f}秒")
        print(f"✓ 输出文件:")
        for name, path in data['output_files'].items():
            print(f"  - {name}: {path}")
        print()
        
        return data
    except Exception as e:
        print(f"✗ 分离失败：{e}")
        return False


def test_download(file_path: str, output_path: str):
    """测试下载分离结果"""
    print("=" * 50)
    print("测试 3: 下载文件")
    print("=" * 50)
    
    try:
        # 提取相对路径
        if "output/" in file_path:
            rel_path = file_path.split("output/")[1]
        else:
            rel_path = file_path
        
        response = requests.get(
            f"{BASE_URL}/api/v1/download/{rel_path}",
            timeout=60
        )
        
        response.raise_for_status()
        
        with open(output_path, "wb") as f:
            f.write(response.content)
        
        print(f"✓ 下载成功：{output_path}")
        print(f"✓ 文件大小：{Path(output_path).stat().st_size / 1024:.2f} KB")
        print()
        
        return True
    except Exception as e:
        print(f"✗ 下载失败：{e}")
        return False


def test_root():
    """测试根路径"""
    print("=" * 50)
    print("测试 4: 根路径")
    print("=" * 50)
    
    try:
        response = requests.get(f"{BASE_URL}/", timeout=10)
        response.raise_for_status()
        
        data = response.json()
        print(f"✓ 服务：{data['service']}")
        print(f"✓ 版本：{data['version']}")
        print(f"✓ 状态：{data['status']}")
        print()
        
        return True
    except Exception as e:
        print(f"✗ 根路径测试失败：{e}")
        return False


def main():
    """主测试函数"""
    print("\n")
    print("╔" + "=" * 48 + "╗")
    print("║  BSRoformer SCNet 音轨分离服务测试        ║")
    print("╚" + "=" * 48 + "╝")
    print()
    
    # 测试 1: 健康检查
    if not test_health():
        print("\n✗ 服务未启动，请先启动服务")
        print("  docker-compose up -d")
        return
    
    # 测试 2: 根路径
    test_root()
    
    # 测试 3: 音轨分离（需要提供测试文件）
    test_file = "data/input/test.mp3"
    if Path(test_file).exists():
        result = test_separate(test_file)
        
        if result and result.get("success"):
            # 测试 4: 下载（可选）
            vocals_path = result["output_files"].get("vocals")
            if vocals_path:
                test_download(vocals_path, "vocals_output.wav")
    else:
        print(f"\n提示：将测试音频文件放到 {test_file}")
        print("支持的格式：mp3, wav, flac, m4a, ogg")
    
    print("=" * 50)
    print("测试完成")
    print("=" * 50)


if __name__ == "__main__":
    main()
