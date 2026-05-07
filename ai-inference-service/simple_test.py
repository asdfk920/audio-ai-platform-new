"""
音轨分离服务简单测试
"""

import requests
import time
import json
import sys

# 设置控制台编码
sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = "http://localhost:8004"

print("=" * 60)
print("BSRoformer SCNet 音轨分离服务 - 简单测试")
print("=" * 60)
print()

# 测试 1: 健康检查
print("[测试 1] 健康检查")
try:
    response = requests.get(f"{BASE_URL}/health", timeout=10)
    data = response.json()
    print(f"[OK] 服务状态：{data['status']}")
    print(f"[OK] 模型：{data['model_name']}")
    print(f"[OK] 任务数：{data['tasks']['total']}")
except Exception as e:
    print(f"[FAIL] 失败：{e}")
    exit(1)

print()

# 测试 2: 开始分离
print("[测试 2] 开始音轨分离")
test_audio = "https://www2.cs.uic.edu/~i101/SoundFiles/BabyElephantWalk60.wav"

payload = {
    "audio_url": test_audio,
    "user_id": "test_user",
    "device_id": "test_device"
}

try:
    response = requests.post(
        f"{BASE_URL}/api/v1/separate/start",
        json=payload,
        timeout=30
    )
    result = response.json()
    
    print(f"[OK] 任务创建成功")
    print(f"  任务 ID: {result['task_id']}")
    print(f"  状态：{result['data']['status']}")
    
    task_id = result['task_id']
    
except Exception as e:
    print(f"[FAIL] 失败：{e}")
    exit(1)

print()

# 测试 3: 查询状态
print("[测试 3] 查询任务状态")
max_attempts = 30

for i in range(max_attempts):
    try:
        response = requests.get(
            f"{BASE_URL}/api/v1/separate/status/{task_id}",
            timeout=30
        )
        status = response.json()
        
        status_text = status['status']
        pct = status['progress']['percentage']
        
        print(f"  [{i+1}/{max_attempts}] 状态：{status_text:15s} ({pct}%)")
        
        if status['status'] == 'completed':
            print(f"[OK] 任务完成！")
            print(f"  处理时间：{status.get('processing_time', 0):.2f}秒")
            
            if status.get('output_files'):
                print(f"  输出文件:")
                for stem, path in status['output_files'].items():
                    print(f"    - {stem}: {path}")
            break
            
        elif status['status'] == 'failed':
            print(f"[FAIL] 任务失败")
            print(f"  错误：{status.get('error_message', '未知')}")
            break
            
        time.sleep(2)
        
    except Exception as e:
        print(f"[FAIL] 查询失败：{e}")
        break

print()

# 测试 4: 列出任务
print("[测试 4] 列出所有任务")
try:
    response = requests.get(f"{BASE_URL}/api/v1/tasks", timeout=30)
    data = response.json()
    
    print(f"[OK] 任务总数：{data['count']}")
    
    if data['tasks']:
        print(f"  最近任务:")
        for task in data['tasks'][-3:]:
            print(f"    - {task['task_id']}: {task['status']}")
            
except Exception as e:
    print(f"[FAIL] 失败：{e}")

print()
print("=" * 60)
print("测试完成！")
print("=" * 60)
print()
print("服务信息:")
print(f"  API 地址：{BASE_URL}")
print(f"  文档地址：{BASE_URL}/docs")
print(f"  健康检查：{BASE_URL}/health")
