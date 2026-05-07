"""
音轨分离服务 - Token 认证测试
"""

import requests
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = "http://localhost:8004"

print("=" * 60)
print("BSRoformer SCNet 音轨分离服务 - Token 认证测试")
print("=" * 60)
print()

# 步骤 1: 生成 token
print("[步骤 1] 生成 JWT Token")
print("-" * 60)

auth_response = requests.post(
    f"{BASE_URL}/api/v1/auth/generate-token",
    params={"user_id": "user_123", "device_id": "device_456"},
    timeout=30
)

auth_data = auth_response.json()
print(f"[OK] Token 生成成功")
print(f"  Token: {auth_data['token'][:50]}...")
print(f"  类型：{auth_data['token_type']}")
print(f"  过期时间：{auth_data['expires_in']} 秒")
print()

token = auth_data['token']

# 步骤 2: 使用 token 调用 API（不带 user_id）
print("[步骤 2] 使用 Token 调用音轨分离 API")
print("-" * 60)

headers = {
    "Authorization": f"Bearer {token}",
    "Content-Type": "application/json"
}

# 注意：现在不需要在 body 中传 user_id 了
payload = {
    "audio_url": "https://www2.cs.uic.edu/~i101/SoundFiles/BabyElephantWalk60.wav",
    "device_id": "device_456"
}

print(f"请求 Header: Authorization: Bearer {token[:30]}...")
print(f"请求 Body: {json.dumps(payload, indent=2)}")
print()

try:
    response = requests.post(
        f"{BASE_URL}/api/v1/separate/start",
        json=payload,
        headers=headers,
        timeout=30
    )
    
    result = response.json()
    
    if response.status_code == 200:
        print(f"[OK] 任务创建成功")
        print(f"  任务 ID: {result['task_id']}")
        print(f"  用户 ID: {result['data']['user_id']} (从 token 中提取)")
        print(f"  状态：{result['data']['status']}")
        
        task_id = result['task_id']
    else:
        print(f"[FAIL] 失败：{result.get('detail', '未知错误')}")
        exit(1)
        
except Exception as e:
    print(f"[FAIL] 失败：{e}")
    exit(1)

print()

# 步骤 3: 测试无 token 的情况
print("[步骤 3] 测试无 Token 的情况")
print("-" * 60)

try:
    response = requests.post(
        f"{BASE_URL}/api/v1/separate/start",
        json=payload,
        timeout=30
    )
    
    result = response.json()
    print(f"[OK] 正确拒绝请求")
    print(f"  状态码：{response.status_code}")
    print(f"  错误信息：{result.get('detail', '未知')}")
    
except Exception as e:
    print(f"[FAIL] 失败：{e}")

print()

# 步骤 4: 测试无效 token
print("[步骤 4] 测试无效 Token")
print("-" * 60)

invalid_headers = {
    "Authorization": "Bearer invalid_token_xyz",
    "Content-Type": "application/json"
}

try:
    response = requests.post(
        f"{BASE_URL}/api/v1/separate/start",
        json=payload,
        headers=invalid_headers,
        timeout=30
    )
    
    result = response.json()
    print(f"[OK] 正确拒绝无效 Token")
    print(f"  状态码：{response.status_code}")
    print(f"  错误信息：{result.get('detail', '未知')}")
    
except Exception as e:
    print(f"[FAIL] 失败：{e}")

print()
print("=" * 60)
print("测试完成！")
print("=" * 60)
print()
print("使用说明:")
print("1. 在 Apifox 中，先在 Header 添加 Authorization: Bearer <token>")
print("2. Body 中只需要传 audio_url 和 device_id，不需要 user_id")
print("3. user_id 会自动从 token 中提取")
print()
print("生成 Token 接口:")
print(f"  POST {BASE_URL}/api/v1/auth/generate-token?user_id=user_123")
print()
