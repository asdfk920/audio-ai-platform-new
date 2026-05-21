# 服务端ACK确认机制实现指南

## 📋 概述

为了配合设备端的可靠传输机制，服务端需要在收到消息后返回**ACK（Acknowledgement）确认包**，确保设备端能够正确删除本地缓存数据。

## 🔧 服务端实现要点

### **1. ACK响应格式**

```json
{
  "type": "ack",
  "message_id": "uuid-xxxx-xxxx-xxxx",
  "success": true,
  "timestamp": 1700000000000,
  "error": null
}
```

**字段说明：**
- `type`: 固定值 `"ack"`，标识这是确认消息
- `message_id`: 设备发送的消息中的 `_meta.message_id` 字段
- `success`: 是否处理成功 (`true`/`false`)
- `timestamp`: 服务端处理时间戳
- `error`: 失败时的错误信息（可选）

---

### **2. 消息去重机制（Redis实现）**

```go
// 在 device_ws_logic.go 中添加

// ProcessDeviceMessageWithAck - 处理设备消息并返回ACK
func (l *DeviceWsLogic) ProcessDeviceMessageWithAck(deviceID int64, message map[string]interface{}) (map[string]interface{}, error) {
    // 1. 提取消息元数据
    meta, _ := message["_meta"].(map[string]interface{})
    if meta == nil {
        return nil, fmt.Errorf("缺少_meta字段")
    }
    
    messageID, _ := meta["message_id"].(string)
    if messageID == "" {
        return nil, fmt.Errorf("缺少message_id")
    }

    // 2. 去重检查（使用Redis Set）
    dedupKey := fmt.Sprintf("device:processed_messages:%s", deviceID)
    
    // 检查是否已处理过
    isDuplicate, err := l.svcCtx.Redis.SIsMember(l.ctx, dedupKey, messageID).Result()
    if err != nil {
        logx.Errorf("[WS] Redis去重检查失败: %v", err)
        // 继续处理（宁可重复也不能丢失）
    } else if isDuplicate {
        // 已处理过，直接返回成功ACK（幂等性）
        return l.buildAckResponse(messageID, true, nil)
    }

    // 3. 标记为已处理（TTL=24小时）
    err = l.svcCtx.Redis.SAdd(l.ctx, dedupKey, messageID).Err()
    if err == nil {
        l.svcCtx.Redis.Expire(l.ctx, dedupKey, 24*time.Hour)
    }

    // 4. 根据消息类型分发处理
    msgType, _ := message["type"].(string)
    switch msgType {
    case "status_report":
        err = l.handleStatusReport(deviceID, message)
    case "heartbeat":
        err = l.handleHeartbeat(deviceID, message)
    case "cmd_response":
        err = l.handleCommandResponse(deviceID, message)
    default:
        err = l.handleGenericMessage(deviceID, message)
    }

    if err != nil {
        logx.Errorf("[WS] 消息处理失败: type=%s, id=%s, error=%v", msgType, messageID, err)
        return l.buildAckResponse(messageID, false, err.Error())
    }

    // 5. 返回成功ACK
    return l.buildAckResponse(messageID, true, nil)
}

// buildAckResponse - 构造ACK响应
func (l *DeviceWsLogic) buildAckResponse(messageID string, success bool, errMsg string) map[string]interface{} {
    ack := map[string]interface{}{
        "type":      "ack",
        "message_id": messageID,
        "success":   success,
        "timestamp": time.Now().UnixMilli(),
    }
    
    if !success && errMsg != "" {
        ack["error"] = errMsg
    }
    
    return ack
}
```

---

### **3. WebSocket消息处理流程**

在现有的WebSocket Handler中集成ACK逻辑：

```go
// 在 device_ws_handler.go 或相关文件中修改 HandleMessage 方法

func (h *DeviceWsHandler) HandleMessage(conn *websocket.Conn, deviceID int64, data []byte) error {
    var message map[string]interface{}
    if err := json.Unmarshal(data, &message); err != nil {
        return fmt.Errorf("JSON解析失败: %w", err)
    }

    // 跳过心跳消息（不需要ACK）
    msgType, _ := message["type"].(string)
    if msgType == "ping" || msgType == "pong" {
        return nil
    }

    // 处理消息并生成ACK
    ack, err := h.logic.ProcessDeviceMessageWithAck(deviceID, message)
    if err != nil {
        logx.Errorf("[WS] 消息处理异常: %v", err)
        
        // 返回错误ACK
        ack = map[string]interface{}{
            "type":      "ack",
            "message_id": extractMessageID(message),
            "success":   false,
            "timestamp": time.Now().UnixMilli(),
            "error":     err.Error(),
        }
    }

    // 发送ACK给设备
    ackData, _ := json.Marshal(ack)
    if err := conn.WriteMessage(websocket.TextMessage, ackData); err != nil {
        logx.Errorf("[WS] 发送ACK失败: %v", err)
        return err
    }

    logx.Infof("[WS] ✅ ACK已发送: message_id=%s, success=%v", 
        ack["message_id"], ack["success"])

    return nil
}

// 辅助函数：提取消息ID
func extractMessageID(message map[string]interface{}) string {
    if meta, ok := message["_meta"].(map[string]interface{}); ok {
        if id, ok := meta["message_id"].(string); ok {
            return id
        }
    }
    return ""
}
```

---

### **4. 性能优化建议**

#### **批量ACK（减少网络开销）**
```go
// 对于高频消息，可以批量收集后统一ACK
type AckBatch struct {
    mu       sync.Mutex
    messages []map[string]interface{}
    timer    *time.Timer
}

func (b *AckBatch) Add(ack map[string]interface{}) {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.messages = append(b.messages, ack)

    // 累积到10条或100ms后发送
    if len(b.messages) >= 10 {
        b.flush()
    } else if b.timer == nil {
        b.timer = time.AfterFunc(100 * time.Millisecond, b.flush)
    }
}
```

#### **异步处理（不阻塞WebSocket）**
```go
// 使用goroutine异步处理消息
go func() {
    ack, err := logic.ProcessDeviceMessageWithAck(deviceID, message)
    // 通过channel或回调发送ACK
    ackChannel <- ack
}()
```

---

### **5. 监控和日志**

```go
// 记录关键指标
type MessageProcessingMetrics struct {
    TotalReceived   int64
    SuccessCount    int64
    FailedCount     int64
    DuplicateCount  int64
    AvgProcessTime  float64
}

// 定期输出统计
func (l *DeviceWsLogic) LogMessageStats() {
    stats := l.metrics
    logx.Infof("[WS] 📊 消息处理统计:" +
        " 总计=%d | 成功=%d | 失败=%d | 重复=%d | 平均耗时=%.2fms",
        stats.TotalReceived,
        stats.SuccessCount,
        stats.FailedCount,
        stats.DuplicateCount,
        stats.AvgProcessTime,
    )
}
```

---

### **6. 错误处理策略**

| 场景 | 服务端行为 | 设备端预期 |
|------|-----------|-----------|
| 消息格式错误 | 返回 `success: false`, `error: "格式无效"` | 重试该消息 |
| 消息ID缺失 | 返回 `success: false`, `error: "缺少message_id"` | 重新构造并发送 |
| 业务逻辑错误 | 返回 `success: false`, `error: 具体原因` | 根据错误类型决定是否重试 |
| 重复消息 | 返回 `success: true` （幂等） | 删除本地缓存 |
| 系统过载 | 返回 `success: false`, `error: "系统繁忙"` | 指数退避重试 |

---

### **7. 测试验证方法**

#### **使用Postman/curl测试**
```bash
# 发送测试消息（模拟设备）
curl -X POST http://localhost:8002/api/device/test-message \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "status_report",
    "payload": {"battery_level": 85},
    "_meta": {"message_id": "test-uuid-123"}
  }'
```

#### **查看Redis去重记录**
```bash
redis-cli SMEMBERS device:processed_messages:<device_id>
```

#### **监控ACK延迟**
```bash
# 在服务端日志中搜索
grep "ACK已发送" /var/log/device-ws.log
```

---

## ⚠️ 注意事项

1. **必须快速返回ACK**：不要在业务逻辑中做耗时操作后再返回ACK
2. **保证幂等性**：相同消息ID多次处理结果应一致
3. **合理设置TTL**：去重记录不要永久保存，定期清理
4. **监控队列积压**：如果ACK延迟过高，说明服务端处理能力不足
5. **优雅降级**：即使Redis不可用，也应继续处理消息（宁可重复）

---

## 📚 相关文件

- [reliable-websocket-client.js](./reliable-websocket-client.js) - 客户端实现
- [persistent-message-queue.js](./persistent-message-queue.js) - 持久化队列
- [reliable-transmission-demo.js](./reliable-transmission-demo.js) - 完整演示

---

## 🎯 下一步行动

1. ✅ 集成上述Go代码到你的WebSocket Handler
2. ✅ 配置Redis用于消息去重
3. ✅ 添加单元测试验证ACK逻辑
4. ✅ 使用演示脚本进行端到端测试
5. ✅ 监控生产环境中的ACK成功率
