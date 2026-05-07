"""
音轨分离服务完整流程测试
"""

import requests
import time
import sys


BASE_URL = "http://localhost:8004"


def print_section(title):
    """打印章节标题"""
    print("\n" + "=" * 60)
    print(title.center(60))
    print("=" * 60 + "\n")


def test_health():
    """测试健康检查"""
    print_section("测试 1: 健康检查")
    
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=10)
        response.raise_for_status()
        
        data = response.json()
        print(f"✓ 服务状态：{data['status']}")
        print(f"✓ 模型名称：{data['model_name']}")
        print(f"✓ 设备：{data['device']}")
        print(f"✓ GPU 可用：{data['gpu_available']}")
        print(f"✓ 任务统计：{data.get('tasks', {})}")
        
        return True
    except Exception as e:
        print(f"✗ 健康检查失败：{e}")
        return False


def test_start_separation():
    """测试开始分离"""
    print_section("测试 2: 开始音轨分离")
    
    # 使用测试音频 URL（这里使用一个示例 URL）
    test_audio_url = "https://www2.cs.uic.edu/~i101/SoundFiles/BabyElephantWalk60.wav"
    
    try:
        payload = {
            "audio_url": test_audio_url,
            "user_id": "test_user",
            "device_id": "test_device"
        }
        
        print(f"请求 URL: {BASE_URL}/api/v1/separate/start")
        print(f"音频链接：{test_audio_url}")
        
        response = requests.post(
            f"{BASE_URL}/api/v1/separate/start",
            json=payload,
            timeout=30
        )
        response.raise_for_status()
        
        result = response.json()
        print(f"\n✓ 响应状态码：{response.status_code}")
        print(f"✓ 成功：{result['success']}")
        print(f"✓ 任务 ID: {result['task_id']}")
        print(f"✓ 消息：{result['message']}")
        
        if result.get('data'):
            data = result['data']
            print(f"✓ 任务状态：{data['status']}")
            print(f"✓ 进度：{data['progress']}")
        
        return result.get('task_id')
        
    except Exception as e:
        print(f"✗ 开始分离失败：{e}")
        return None


def test_get_status(task_id):
    """测试查询任务状态"""
    print_section(f"测试 3: 查询任务状态 ({task_id})")
    
    try:
        max_attempts = 30
        poll_interval = 2
        
        for i in range(max_attempts):
            response = requests.get(
                f"{BASE_URL}/api/v1/separate/status/{task_id}",
                timeout=30
            )
            response.raise_for_status()
            
            status_data = response.json()
            status = status_data['status']
            progress = status_data.get('progress', {})
            
            print(f"[{i+1}/{max_attempts}] 状态：{status:15s} | "
                  f"进度：{progress.get('percentage', 0):5.1f}% | "
                  f"阶段：{progress.get('stage', 'unknown')}")
            
            if status == "completed":
                print(f"\n✓ 任务完成！")
                print(f"✓ 处理时间：{status_data.get('processing_time', 0):.2f}秒")
                
                if status_data.get('output_files'):
                    print(f"✓ 输出文件:")
                    for stem, path in status_data['output_files'].items():
                        print(f"  - {stem}: {path}")
                
                return status_data
            
            elif status == "failed":
                print(f"\n✗ 任务失败")
                print(f"✗ 错误信息：{status_data.get('error_message', '未知')}")
                return status_data
            
            elif status == "timeout":
                print(f"\n✗ 任务超时")
                return status_data
            
            time.sleep(poll_interval)
        
        print(f"\n✗ 查询超时，等待了 {max_attempts * poll_interval} 秒")
        return None
        
    except Exception as e:
        print(f"✗ 查询失败：{e}")
        return None


def test_download(task_id, status_data):
    """测试下载结果"""
    print_section("测试 4: 下载分离结果")
    
    if not status_data or status_data.get('status') != 'completed':
        print("⊘ 任务未完成，跳过下载测试")
        return
    
    try:
        output_files = status_data.get('output_files', {})
        
        if not output_files:
            print("⊘ 没有输出文件")
            return
        
        for stem in output_files.keys():
            print(f"\n下载 {stem}...")
            
            response = requests.get(
                f"{BASE_URL}/api/v1/separate/download/{task_id}/{stem}",
                timeout=60
            )
            response.raise_for_status()
            
            filename = f"test_{task_id}_{stem}.wav"
            with open(filename, 'wb') as f:
                f.write(response.content)
            
            file_size = len(response.content)
            print(f"✓ 下载成功：{filename} ({file_size:,} 字节)")
        
        return True
        
    except Exception as e:
        print(f"✗ 下载失败：{e}")
        return False


def test_list_tasks():
    """测试列出任务"""
    print_section("测试 5: 列出所有任务")
    
    try:
        response = requests.get(
            f"{BASE_URL}/api/v1/tasks",
            timeout=30
        )
        response.raise_for_status()
        
        data = response.json()
        print(f"✓ 任务总数：{data['count']}")
        
        if data['tasks']:
            print(f"\n任务列表:")
            for task in data['tasks'][:5]:  # 只显示前 5 个
                print(f"  - {task['task_id']}: {task['status']}")
        
        return True
        
    except Exception as e:
        print(f"✗ 列出任务失败：{e}")
        return False


def main():
    """主测试函数"""
    print("\n")
    print("╔" + "═" * 58 + "╗")
    print("║".center(60))
    print("║  BSRoformer SCNet 音轨分离服务 - 完整流程测试".center(60))
    print("║".center(60))
    print("╚" + "═" * 58 + "╝")
    
    # 测试 1: 健康检查
    if not test_health():
        print("\n✗ 服务未启动，请先启动服务")
        print("  python server.py")
        return 1
    
    # 测试 2: 开始分离
    task_id = test_start_separation()
    
    if not task_id:
        print("\n✗ 创建任务失败")
        return 1
    
    # 测试 3: 查询状态并等待完成
    status_data = test_get_status(task_id)
    
    # 测试 4: 下载结果
    test_download(task_id, status_data)
    
    # 测试 5: 列出任务
    test_list_tasks()
    
    # 总结
    print_section("测试总结")
    print("✓ 所有测试完成！")
    print("\n服务信息:")
    print(f"  - API 地址：{BASE_URL}")
    print(f"  - 文档地址：{BASE_URL}/docs")
    print(f"  - 健康检查：{BASE_URL}/health")
    
    return 0


if __name__ == "__main__":
    sys.exit(main())
