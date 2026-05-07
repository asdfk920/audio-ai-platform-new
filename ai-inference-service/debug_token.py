"""
Token 认证调试工具
帮助诊断 Token 无效的问题
"""

import requests
import jwt
import json

BASE_URL = "http://localhost:8004"
JWT_SECRET_KEY = "your-secret-key-change-in-production"

print("=" * 60)
print("Token 认证调试工具")
print("=" * 60)
print()

# 请输入您要测试的 token
print("请输入要测试的 Token:")
print("(直接回车将使用测试 token)")
user_input = input("> ").strip()

if not user_input:
    # 生成一个测试 token
    print("\n生成测试 Token...")
    auth_response = requests.post(
        f"{BASE_URL}/api/v1/auth/generate-token",
        params={"user_id": "test_user", "device_id": "test_device"},
        timeout=30
    )
    token = auth_response.json()["token"]
    print(f"✓ 生成新 Token: {token[:50]}...")
else:
    token = user_input
    print(f"使用输入的 Token: {token[:50]}...")

print()
print("-" * 60)
print("Token 分析")
print("-" * 60)

# 尝试解码 token（不验证签名）
try:
    decoded = jwt.decode(token, options={"verify_signature": False})
    print(f"✓ Token 格式有效")
    print(f"  Payload: {json.dumps(decoded, indent=2)}")
    
    if "user_id" in decoded:
        print(f"✓ 包含 user_id: {decoded['user_id']}")
    else:
        print(f"✗ 缺少 user_id")
        
    if "exp" in decoded:
        from datetime import datetime
        exp_time = datetime.fromtimestamp(decoded["exp"])
        print(f"✓ 过期时间：{exp_time}")
        
except Exception as e:
    print(f"✗ Token 格式错误：{e}")
    exit(1)

print()
print("-" * 60)
print("验证签名")
print("-" * 60)

# 验证签名
try:
    decoded = jwt.decode(token, JWT_SECRET_KEY, algorithms=["HS256"])
    print(f"✓ 签名验证成功")
    print(f"  用户：{decoded.get('user_id')}")
except jwt.InvalidSignatureError:
    print(f"✗ 签名验证失败")
    print(f"  原因：Token 不是使用当前服务的密钥生成的")
    print(f"  解决：请使用本服务 /api/v1/auth/generate-token 接口生成 token")
except jwt.ExpiredSignatureError:
    print(f"✗ Token 已过期")
except Exception as e:
    print(f"✗ 验证失败：{e}")

print()
print("-" * 60)
print("实际调用 API 测试")
print("-" * 60)

headers = {
    "Authorization": f"Bearer {token}",
    "Content-Type": "application/json"
}

payload = {
    "audio_url": "https://www2.cs.uic.edu/~i101/SoundFiles/BabyElephantWalk60.wav",
    "device_id": "test_device"
}

print(f"调用 API...")
try:
    response = requests.post(
        f"{BASE_URL}/api/v1/separate/start",
        json=payload,
        headers=headers,
        timeout=30
    )
    
    print(f"状态码：{response.status_code}")
    print(f"响应：{json.dumps(response.json(), indent=2, ensure_ascii=False)}")
    
    if response.status_code == 200:
        print("\n✓ Token 有效，API 调用成功！")
    else:
        print(f"\n✗ API 调用失败")
        
except Exception as e:
    print(f"✗ 请求失败：{e}")

print()
print("=" * 60)
print("调试完成")
print("=" * 60)
print()
print("常见问题:")
print("1. Token 不是使用本服务生成的 -> 请使用 /api/v1/auth/generate-token 生成")
print("2. Token 已过期 -> 重新生成 token")
print("3. Token 格式错误 -> 检查是否完整复制")
print("4. 密钥不匹配 -> 确保服务器使用相同的 JWT_SECRET_KEY")
