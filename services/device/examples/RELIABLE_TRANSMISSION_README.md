# 🛡️ 设备数据零丢失保障系统

## 📖 系统概述

本系统实现了**设备端数据的完整可靠性保障**，确保在各种异常情况下（断电、网络波动、服务重启等）数据**零丢失、不重复、有序传输**。

---

## 🎯 核心能力

| 能力 | 描述 | 重要性 |
|------|------|--------|
| **本地持久化** | 数据写入JSON文件，断电不丢、重启不丢 | ⭐⭐⭐⭐⭐ |
| **ACK确认机制** | 服务端确认后才删除本地数据 | ⭐⭐⭐⭐⭐ |
| **断连自动缓存** | 不管什么原因断连，自动缓存不丢失 | ⭐⭐⭐⭐⭐ |
| **重连自动补发** | 恢复连接后立即按顺序补发所有待发数据 | ⭐⭐⭐⭐⭐ |
| **消息顺序保证** | 按时间顺序补发，严格FIFO，不乱序 | ⭐⭐⭐⭐ |
| **重复发送判断** | 服务端根据消息ID去重，避免重复执行 | ⭐⭐⭐⭐ |

---

## 📦 文件清单

```
examples/
├── persistent-message-queue.js        # JS版持久化消息队列
├── reliable-websocket-client.js       # JS版可靠传输WebSocket客户端
├── reliable-transmission-demo.js      # JS版完整演示脚本
├── persistent_message_queue.py        # Python版持久化消息队列
└── SERVER_ACK_IMPLEMENTATION.md       # 服务端ACK实现指南
```

---

## 🚀 快速开始（3种方式）

---

### **方式1: JavaScript (Node.js / 浏览器)**

#### 安装依赖
```bash
npm install uuid ws
```

#### 基础使用示例
```javascript
const ReliableWebSocketClient = require('./reliable-websocket-client');

// 1. 创建客户端实例
const client = new ReliableWebSocketClient({
    url: 'ws://localhost:8002/ws/device',
    token: 'your_jwt_token_here',       // 从 /api/device/auth 获取
    deviceId: 'DEVICE-SN-123456',

    // 可靠传输配置（核心！）
    enablePersistence: true,            // 开启持久化存储
    storagePath: './data/my_device_queue.json',  // 存储路径
    ackTimeout: 10000,                  // ACK超时10秒
    maxRetryCount: 5,                   // 最大重试5次
    maxQueueSize: 1000,                 // 队列最大1000条

    // 回调函数
    onOpen: (event) => {
        console.log('✅ 连接已建立');
    },

    onMessage: (data) => {
        console.log('📥 收到消息:', data);
        
        // 处理指令并反馈
        if (data.type === 'instruction') {
            handleCommand(data);
            
            // 使用可靠发送返回结果
            client.sendReliable({
                type: 'cmd_response',
                payload: {
                    cmd: data.cmd,
                    success: true,
                    timestamp: Date.now()
                }
            }).then(result => {
                console.log('✅ 指令响应已确认:', result.message_id);
            });
        }
    },

    onReconnected: (info) => {
        console.log(`🎉 重连成功! 开始自动补发 ${client.getReliabilityStats().message_queue.pending} 条缓存消息...`);
    }
});

// 2. 启动连接
client.connect();

// 3. 可靠发送数据（带持久化+ACK）
async function sendDeviceStatus() {
    try {
        const result = await client.sendReliable({
            type: 'status_report',
            payload: {
                battery_level: 85,
                volume: 60,
                playing: true,
                temperature: 25.5,
                timestamp: Date.now()
            },
            metadata: {
                priority: 'normal',
                source: 'device_firmware_v2.1'
            }
        });

        console.log(`✅ 数据已确认收到! Message ID: ${result.message_id}`);
        console.log(`   确认时间: ${new Date(result.acked_at).toLocaleString()}`);

    } catch (error) {
        console.error(`❌ 发送失败: ${error.message}`);
        // 失败的消息会自动保留在本地队列中，重连后会重新发送
    }
}

// 定期发送状态报告
setInterval(sendDeviceStatus, 5000); // 每5秒发送一次
```

#### 运行演示脚本
```bash
node reliable-transmission-demo.js <your_jwt_token> [device_id]
```

**输出示例：**
```
============================================================================
[10:30:15] 🚀 可靠传输演示系统启动
============================================================================

[10:30:15] 📍 步骤1: 创建可靠传输客户端...
[MessageQueue] 📦 持久化消息队列初始化完成
[MessageQueue] 📍 存储路径: ./data/reliable_transmission_demo.json
[ReliableWS] ✅ 可靠传输模式已启用

[10:30:16] ✅ WebSocket连接已建立
[ReliableWS] 🚀 连接已建立，开始同步待发消息...

[10:30:17] 📤 [1/20] 发送消息:
   类型: status_report
   序号: #1
   ✅ 已入队并等待ACK
   Message ID: a1b2c3d4...
   队列长度: 1

[10:30:19] ✅ 收到ACK确认:
   Message ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
   Success: true
   Timestamp: 5/20/2026, 10:30:19:123 AM

[10:30:19] ✅ 消息已确认(ACK): a1b2c3d4-e5f6-7890-abcd-ef1234567890

... (继续发送剩余消息)

============================================================================
📈 最终统计报告
============================================================================

✅ 消息发送统计:
   总计发送: 20 条
   成功入队: 20 条 (100.0%)
   发送失败: 0 条

📦 队列状态:
   待发送: 0 条
   等待ACK: 0 条
   已确认: 20 条
   队列利用率: 0.0%

💾 存储信息:
   文件路径: ./data/reliable_transmission_demo.json
   文件大小: 0.25 KB

✅ 演示完成！按 Ctrl+C 退出
```

---

### **方式2: Python (嵌入式设备/IoT)**

#### 安装依赖
```bash
pip install websocket-client requests uuid
```

#### 基础使用示例
```python
#!/usr/bin/env python3
from device_websocket_client import DeviceWebSocketClient
from persistent_message_queue import PersistentMessageQueue
import time
import json
import threading

# 1. 创建持久化队列
message_queue = PersistentMessageQueue({
    'storage_path': './data/python_device_queue.json',
    'max_queue_size': 1000
})

# 2. 创建WebSocket客户端
client = DeviceWebSocketClient({
    'url': 'ws://localhost:8002/ws/device',
    'token': 'your_jwt_token',
    'device_id': 'PY-DEVICE-001',
    
    # 重连配置
    'reconnect_enabled': True,
    'initial_reconnect_delay': 2.0,
    'max_reconnect_delay': 30.0,
})

# 3. 注册回调函数
def on_open(ws):
    print('✅ 连接已建立')
    start_sync_loop()  # 启动同步循环

def on_message(data):
    if isinstance(data, dict) and data.get('type') == 'ack':
        handle_ack(data)
    else:
        process_incoming_data(data)

def handle_ack(ack_data):
    """处理服务端ACK"""
    message_id = ack_data.get('message_id')
    success = ack_data.get('success', False)
    
    if success:
        message_queue.mark_as_acked(message_id)
        print(f'✅ ACK确认成功: {message_id}')
    else:
        error = ack_data.get('error', '未知错误')
        print(f'❌ ACK失败: {message_id}, 错误: {error}')

def send_reliable(message):
    """可靠发送消息"""
    result = message_queue.enqueue(message)
    
    if result['success']:
        message_id = result['message_id']
        
        if client.state == ConnectionState.CONNECTED:
            # 在线：立即发送
            send_via_websocket(message_id)
        else:
            # 离线：保持PENDING，重连后自动发送
            print(f'💾 消息已缓存: {message_id}')

def start_sync_loop():
    """启动消息同步线程"""
    def sync_worker():
        while True:
            time.sleep(2)  # 每2秒检查一次
            
            if client.state != ConnectionState.CONNECTED:
                continue
                
            pending = message_queue.get_pending_messages()
            for msg in pending:
                send_via_websocket(msg['id'])
    
    thread = threading.Thread(target=sync_worker, daemon=True)
    thread.start()

# 4. 启动连接
print('📍 正在连接...')
client.connect()

# 5. 主循环 - 定期发送状态
try:
    count = 0
    while True:
        count += 1
        
        send_reliable({
            'type': 'status_report',
            'payload': {
                'battery_level': 80 + (count % 20),
                'temperature': 25 + (count % 10),
                'timestamp': int(time.time() * 1000),
                'sequence': count
            }
        })
        
        stats = message_queue.get_stats()
        print(f'📊 统计: 待发={stats["pending"]}, 总计={stats["total"]}')
        
        time.sleep(5)

except KeyboardInterrupt:
    print('\n👋 用户中断')
    message_queue.destroy()
    client.close()
```

运行Python测试：
```bash
python persistent_message_queue.py  # 测试队列功能
```

---

### **方式3: 浏览器可视化测试**

直接打开 `websocket-reconnect-demo.html`（在上一节已创建），该页面已经集成了可靠传输功能。

---

## 🔧 高级配置

### **生产环境推荐配置**

```javascript
const productionConfig = {
    // 可靠传输
    enablePersistence: true,
    storagePath: '/var/lib/device/message_queue.json',  // Linux持久化目录
    ackTimeout: 15000,          // 15秒ACK超时（网络较差环境）
    maxRetryCount: 10,          // 最大重试10次
    maxQueueSize: 5000,         // 较大队列容量
    
    // 重连策略
    initialReconnectDelay: 3000,  // 初始3秒
    maxReconnectDelay: 60000,     // 最大60秒
    reconnectDecay: 1.8,          // 较小的退避因子
    reconnectJitter: 0.4,         // 较大抖动（避免同时重连）
    
    // 心跳
    pingInterval: 120000,         // 2分钟心跳（省电）
    pongTimeout: 150000,          // 心跳超时2.5分钟
    
    // 自动保存
    autoSaveInterval: 3000,       // 每3秒保存一次到文件
};
```

### **嵌入式设备优化配置**

```javascript
const embeddedConfig = {
    // 低功耗模式
    enablePersistence: true,
    storagePath: '/tmp/device_queue.json',  // 内存文件系统
    ackTimeout: 20000,           // 较长超时（弱网环境）
    maxRetryCount: 3,            // 减少重试次数（节省电量）
    maxQueueSize: 100,           // 小内存设备限制队列
    
    // 节能心跳
    pingInterval: 180000,        // 3分钟心跳
    networkCheckInterval: 15000, // 15秒检测一次网络
};
```

---

## 🧪 测试场景验证

### **场景1: 断电恢复测试**

```bash
# 终端1: 运行演示脚本
node reliable-transmission-demo.js <token>

# 发送几条消息后...

# 终端2: 模拟断电（强制终止进程）
kill -9 <pid>

# 终端1: 重新启动
node reliable-transmission-demo.js <token>

# 观察：之前未确认的消息是否自动补发？
```

**预期结果：**
- ✅ 未收到ACK的消息全部保留在JSON文件中
- ✅ 重启后自动读取并发送所有PENDING消息
- ✅ 无任何数据丢失

---

### **场景2: 网络波动测试**

```
1. 设备正常连接，持续发送数据
2. 拔掉网线/关闭WiFi → 观察日志
   - 应显示："当前离线，消息将在重连后发送"
   - 所有新数据写入本地文件
   
3. 等待30秒后插回网线
   - 应显示："重连成功! 开始同步..."
   - 缓存的消息逐条发送
   - 收到ACK后删除本地记录
```

**预期结果：**
- ✅ 网络断开期间数据全部缓存
- ✅ 重连后按时间顺序补发（先进先出）
- ✅ 全部ACK后队列为空

---

### **场景3: 服务重启测试**

```
1. 设备正常连接
2. 重启服务端 (Ctrl+C → go run device.go)
3. 设备检测到断开，进入RECONNECTING状态
4. 服务重启完成后
5. 设备自动重连成功
6. 补发期间的所有缓存数据
```

**预期结果：**
- ✅ 服务重启不影响数据完整性
- ✅ 重连后自动补发
- ✅ 消息顺序正确（按原始时间排序）

---

## 📊 监控和诊断

### **查看实时状态**

```javascript
// 获取完整的可靠性统计
const stats = client.getReliabilityStats();

console.table({
    '连接状态': stats.state,
    '待发消息': stats.message_queue?.pending || 0,
    '等待ACK': stats.pending_acks || 0,
    '缓存总数': stats.message_queue?.total || 0,
    '队列利用率': stats.message_queue?.utilization || 'N/A',
    '重连次数': stats.reconnectAttempts || 0,
    '网络状态': stats.networkAvailable ? '在线' : '离线'
});
```

### **检查本地存储文件**

```bash
# 查看队列内容
cat ./data/reliable_transmission_demo.json | jq '.messages[] | {id, type, status, created_at}'

# 统计各状态数量
cat ./data/reliable_transmission_demo.json | jq '[.messages[].status] | group_by(.) | map({status: .[0], count: length})'

# 查看最旧的消息
cat ./data/reliable_transmission_demo.json | jq '.messages | sort_by(.created_at) | .[0]'
```

### **常见问题排查**

| 问题现象 | 可能原因 | 解决方案 |
|---------|---------|---------|
| 队列不断增长 | 服务端未返回ACK | 检查服务端ACK逻辑 |
| 大量SENT状态消息 | ACK超时或丢失 | 增加`ackTimeout`值 |
| 消息重复执行 | 服务端未做去重 | 实现`SERVER_ACK_IMPLEMENTATION.md`中的去重逻辑 |
| 文件过大 | 清理不及时 | 减小`maxQueueSize`或定期调用`clearAll()` |
| 内存占用高 | 队列积压严重 | 检查网络连接和服务端处理速度 |

---

## 🎯 最佳实践

### **1. 合理设置队列容量**
```javascript
// 根据设备能力和业务需求调整
maxQueueSize: isEmbeddedDevice ? 100 : 10000;  // 嵌入式设备用小队列
```

### **2. 重要消息优先级标记**
```javascript
await client.sendReliable({
    type: 'critical_alert',
    payload: { ... },
    metadata: { priority: 'high' }  // 可用于后续优先发送
});
```

### **3. 定期清理历史数据**
```javascript
// 每天清理一次已确认的旧消息（如果保留了历史）
setInterval(() => {
    const queue = client.messageQueue;
    // 只保留最近24小时的ACKED记录（可选）
}, 24 * 60 * 60 * 1000);
```

### **4. 错误恢复策略**
```javascript
try {
    await client.sendReliable(criticalData);
} catch (error) {
    if (error.message.includes('达到最大重试次数')) {
        // 记录到本地日志，人工介入
        logToLocalStorage('FAILED', criticalData);
        alertUser('关键数据发送失败，请检查网络');
    } else {
        // 自动重试中，无需特殊处理
        console.warn('临时失败，等待重试...');
    }
}
```

---

## 🔄 与现有代码集成

如果你已经有使用 `DeviceWebSocketClient` 的代码，升级非常简单：

```javascript
// ❌ 旧代码（普通模式）
const client = new DeviceWebSocketClient({ ... });
client.send({ type: 'status', data: {...} });

// ✅ 新代码（可靠模式）- 只需改2处！
const ReliableWS = require('./reliable-websocket-client');

const client = new ReliableWS({ 
    enablePersistence: true,  // 开启持久化
    ...其他原有配置 
});

// send() 方法现在自动使用可靠传输！
await client.send({ type: 'status', data: {...} });  // 带ACK确认

// 或者显式使用 sendReliable()
await client.sendReliable({ type: 'critical', data: {...} });
```

**完全向后兼容！** 不修改任何现有代码即可获得可靠性保障。

---

## 📞 技术支持

如遇到问题，请按以下顺序排查：

1. ✅ 查看控制台日志（详细级别）
2. ✅ 检查本地JSON文件内容
3. ✅ 使用演示脚本复现问题
4. ✅ 参考服务端实现指南 (`SERVER_ACK_IMPLEMENTATION.md`)
5. ✅ 查看本文档的"常见问题排查"章节

---

## 🎉 总结

通过本系统，你的设备现在具备了：

- ✅ **工业级可靠性**：媲美MQTT/CoAP的QoS 1级别保障
- ✅ **零配置成本**：无需额外的消息中间件
- ✅ **即插即用**：3行代码即可启用
- ✅ **完美兼容**：与现有代码无缝集成

**立即开始保护你的设备数据吧！** 🚀
