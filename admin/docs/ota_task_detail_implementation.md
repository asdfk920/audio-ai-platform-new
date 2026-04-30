# OTA 任务详情接口实现文档

## 概述

实现了 OTA 任务详情的后台接口，用于查看某一次设备升级任务的完整信息，包括任务配置、目标设备、执行进度、结果统计等。

## 接口信息

### 接口地址
```
GET /api/v1/platform-device/ota-task/detail
```

### 请求方式
- Content-Type: application/json
- 需要 JWT 认证

### 请求参数

#### 必填参数
- `task_id` (int64): 任务 ID，必填且为正整数

#### 选填参数
- `with_devices` (bool): 是否包含设备列表，默认 false
- `with_logs` (bool): 是否包含操作日志，默认 false

### 请求示例

```
GET /api/v1/platform-device/ota-task/detail?task_id=123&with_devices=true&with_logs=true
```

### 返回结果

#### 成功响应
```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "task_id": 123,
    "task_name": "2026 年春季固件升级",
    "task_type": "manual",
    "task_type_text": "手动创建",
    "status": "running",
    "status_text": "执行中",
    "created_at": "2026-04-10 09:00:00",
    "creator": "张三",
    "start_time": "2026-04-10 09:05:00",
    "end_time": "",
    "total_duration_seconds": 7200,
    
    "product_key": "product_123",
    "product_name": "智能音箱 Pro",
    "target_version": "2.1.0",
    "target_version_code": 20100,
    "file_url": "/uploads/firmware/product_123/v2.1.0.bin",
    "force_update": false,
    "max_retry": 3,
    "timeout_seconds": 300,
    
    "total_devices": 1000,
    "device_filter": {
      "device_model": "Model-A",
      "min_version": "1.5.0"
    },
    "exclude_devices": [1001, 1002],
    
    "pending": 200,
    "downloading": 50,
    "downloaded": 100,
    "upgrading": 80,
    "success": 500,
    "failed": 50,
    "timeout": 15,
    "cancelled": 5,
    
    "progress": 57.0,
    "success_rate": 89.29,
    "avg_duration_seconds": 180.5,
    "estimated_remaining_seconds": 36100,
    
    "device_list": [
      {
        "device_id": 1001,
        "device_name": "设备 001",
        "device_sn": "SN001234567",
        "device_model": "Model-A",
        "current_version": "1.8.0",
        "target_version": "2.1.0",
        "status": "success",
        "status_text": "成功",
        "start_time": "2026-04-10 09:10:00",
        "end_time": "2026-04-10 09:13:00",
        "duration_seconds": 180,
        "retry_count": 0,
        "is_online": true
      }
    ],
    
    "logs": [
      {
        "log_id": 1,
        "task_id": 123,
        "operator": "张三",
        "operation": "create",
        "content": "创建 OTA 升级任务",
        "ip_address": "192.168.1.100",
        "created_at": "2026-04-10 09:00:00"
      }
    ]
  }
}
```

### 错误响应

#### 缺少必填参数
```json
{
  "code": 400,
  "msg": "task_id 必填且为正整数"
}
```

#### 任务不存在
```json
{
  "code": 404,
  "msg": "任务不存在"
}
```

## 实现细节

### 文件修改

1. **apis/platform_device_firmware.go**
   - 添加 `OTATaskDetail` API 处理函数
   - 包含参数解析、校验和错误处理

2. **service/platform_device_service.go**
   - 添加错误定义：`ErrOTATaskNotFound`
   - 添加 `OTATaskDetailRequest` 请求结构体
   - 添加 `OTATaskDeviceInfo` 设备信息结构体
   - 添加 `OTATaskLog` 日志结构体
   - 添加 `OTATaskDetailResponse` 响应结构体
   - 实现 `OTATaskDetail` 服务方法（7 步完整处理流程）
   - 添加辅助函数：
     - `otaTaskTypeText`: 任务类型文本转换
     - `otaTaskStatus`: 任务状态标识转换
     - `otaTaskStatusText`: 任务状态文本转换
     - `otaDeviceStatusText`: 设备状态文本转换

3. **router/init.go**
   - 注册路由：`GET /ota-task/detail`

### 处理流程

#### 第一步：任务定位
- 根据任务 ID 从 `ota_upgrade_task` 表查询任务主记录
- 查询字段包括：任务名称、类型、状态、产品标识、目标版本、升级策略等
- 如果任务不存在，返回 `ErrOTATaskNotFound` 错误

#### 第二步：基础信息组装
- 读取任务创建时保存的配置信息
- 包括目标版本、升级策略、目标范围等
- 查询创建人昵称（关联 sys_user 表）
- 查询产品名称（关联 ota_product 表）
- 解析设备筛选条件（JSON 格式）
- 解析排除设备列表（JSON 格式）
- 计算任务总耗时（结束时间 - 开始时间）

#### 第三步：执行数据汇总
- 实时统计各状态的设备数量
- 统计字段包括：
  - `pending`: 待下发数
  - `downloading`: 下载中数
  - `downloaded`: 已下载数
  - `upgrading`: 升级中数
  - `success`: 成功数
  - `failed`: 失败数
  - `timeout`: 超时数
  - `cancelled`: 已取消数
- 从 `ota_upgrade_device` 表按状态分组统计

#### 第四步：目标设备列表获取
- 计算目标设备总数（各状态数量之和）
- 如果请求包含设备列表（`with_devices=true`）
- 从 `ota_upgrade_device` 表查询设备详情
- 限制返回数量（最多 100 条，避免过多）
- 检查设备在线状态（关联 device 表）

#### 第五步：结果详情获取
- 读取每台设备的升级结果记录
- 包括：
  - 设备基本信息（ID、名称、SN、型号）
  - 当前版本和目标版本
  - 升级状态
  - 开始时间和结束时间
  - 升级耗时
  - 错误码和错误信息（如果失败）
  - 重试次数
  - 在线状态

#### 第六步：日志信息关联
- 如果请求包含操作日志（`with_logs=true`）
- 从 `ota_upgrade_task_log` 表查询日志记录
- 按时间正序排列
- 限制返回数量（最多 50 条）
- 包括：
  - 操作人
  - 操作类型
  - 操作内容
  - IP 地址
  - 操作时间

#### 第七步：汇总返回
- 计算进度信息：
  - `progress`: 完成百分比
  - `success_rate`: 成功率
  - `avg_duration`: 平均升级耗时
  - `estimated_remaining`: 预计剩余时间
- 组装完整的任务详情响应
- 返回给调用方

## 数据结构

### OTATaskDetailRequest
```go
type OTATaskDetailRequest struct {
    TaskID      int64   // 任务 ID
    WithDevices bool    // 是否包含设备列表
    WithLogs    bool    // 是否包含操作日志
}
```

### OTATaskDeviceInfo
```go
type OTATaskDeviceInfo struct {
    DeviceID     int64   `json:"device_id"`
    DeviceName   string  `json:"device_name"`
    DeviceSn     string  `json:"device_sn"`
    DeviceModel  string  `json:"device_model"`
    CurrentVer   string  `json:"current_version"`
    TargetVer    string  `json:"target_version"`
    Status       string  `json:"status"`
    StatusText   string  `json:"status_text"`
    StartTime    string  `json:"start_time"`
    EndTime      string  `json:"end_time"`
    Duration     int64   `json:"duration_seconds"`
    ErrorCode    string  `json:"error_code,omitempty"`
    ErrorMsg     string  `json:"error_msg,omitempty"`
    RetryCount   int     `json:"retry_count"`
    IsOnline     bool    `json:"is_online"`
}
```

### OTATaskLog
```go
type OTATaskLog struct {
    LogID     int64  `json:"log_id"`
    TaskID    int64  `json:"task_id"`
    Operator  string `json:"operator"`
    Operation string `json:"operation"`
    Content   string `json:"content"`
    IpAddress string `json:"ip_address"`
    CreatedAt string `json:"created_at"`
}
```

### OTATaskDetailResponse
```go
type OTATaskDetailResponse struct {
    // 基础信息
    TaskID        int64   `json:"task_id"`
    TaskName      string  `json:"task_name"`
    TaskType      string  `json:"task_type"`
    TaskTypeText  string  `json:"task_type_text"`
    Status        string  `json:"status"`
    StatusText    string  `json:"status_text"`
    CreatedAt     string  `json:"created_at"`
    Creator       string  `json:"creator"`
    StartTime     string  `json:"start_time"`
    EndTime       string  `json:"end_time"`
    TotalDuration int64   `json:"total_duration_seconds"`
    
    // 任务配置
    ProductKey     string `json:"product_key"`
    ProductName    string `json:"product_name"`
    TargetVersion  string `json:"target_version"`
    TargetCode     int    `json:"target_version_code"`
    FileURL        string `json:"file_url"`
    ForceUpdate    bool   `json:"force_update"`
    MaxRetry       int    `json:"max_retry"`
    TimeoutSeconds int    `json:"timeout_seconds"`
    
    // 目标范围
    TotalDevices   int64                  `json:"total_devices"`
    DeviceList     []OTATaskDeviceInfo    `json:"device_list,omitempty"`
    DeviceFilter   map[string]interface{} `json:"device_filter,omitempty"`
    ExcludeDevices []int64                `json:"exclude_devices,omitempty"`
    
    // 执行统计
    Pending     int64 `json:"pending"`
    Downloading int64 `json:"downloading"`
    Downloaded  int64 `json:"downloaded"`
    Upgrading   int64 `json:"upgrading"`
    Success     int64 `json:"success"`
    Failed      int64 `json:"failed"`
    Timeout     int64 `json:"timeout"`
    Cancelled   int64 `json:"cancelled"`
    
    // 进度信息
    Progress           float64 `json:"progress"`
    SuccessRate        float64 `json:"success_rate"`
    AvgDuration        float64 `json:"avg_duration_seconds"`
    EstimatedRemaining int64   `json:"estimated_remaining_seconds"`
    
    // 操作日志
    Logs []OTATaskLog `json:"logs,omitempty"`
}
```

## 任务类型说明

| 类型标识 | 中文名称 | 说明 |
|----------|----------|------|
| manual | 手动创建 | 由管理员手动发起的升级任务 |
| scheduled | 定时创建 | 到达指定时间自动执行的升级任务 |
| rule | 规则触发 | 满足条件自动触发的升级任务 |

## 任务状态说明

| 状态标识 | 中文名称 | 说明 |
|----------|----------|------|
| waiting | 等待中 | 任务已创建，等待到达执行时间 |
| running | 执行中 | 任务正在下发升级指令 |
| paused | 执行暂停 | 任务被暂停，可恢复执行 |
| completed | 已完成 | 所有设备处理完毕 |
| failed | 执行失败 | 任务执行异常被终止 |
| cancelled | 已取消 | 管理员取消任务 |

## 设备执行状态说明

| 状态标识 | 中文名称 | 说明 |
|----------|----------|------|
| pending | 待下发 | 等待 MQTT 下发升级指令 |
| downloading | 下载中 | 设备正在下载固件包 |
| downloaded | 已下载 | 固件下载完成，等待升级 |
| upgrading | 升级中 | 设备正在安装固件 |
| success | 成功 | 升级完成，设备运行正常 |
| failed | 失败 | 升级失败，包含错误码和错误信息 |
| timeout | 超时 | 升级超时未完成 |
| cancelled | 已取消 | 被管理员取消升级 |

## 异常信息记录

对于 `failed` 状态的设备，记录以下异常信息：

### 错误码（error_code）
- `DOWNLOAD_FAILED`: 下载失败
- `INSTALL_FAILED`: 安装失败
- `SIGNATURE_FAILED`: 签名验证失败
- `NETWORK_ERROR`: 网络错误
- `STORAGE_FULL`: 存储空间不足
- `INCOMPATIBLE`: 固件不兼容

### 错误描述（error_msg）
- 网络超时
- 文件损坏
- 验证失败
- 安装过程异常
- 设备响应超时

### 设备日志片段（device_log）
- 用于问题定位
- 记录升级过程的关键日志
- 包含错误发生时的上下文信息

### 重试次数（retry_count）
- 记录已重试次数
- 不超过最大重试次数配置

## 关联信息

### 关联固件信息
- 可查看目标固件版本详情
- 通过 `target_version` 和 `product_key` 关联 `ota_firmware` 表

### 关联产品信息
- 可查看所属产品配置
- 通过 `product_key` 关联 `ota_product` 表

### 关联操作日志
- 可查看完整的操作记录
- 通过 `task_id` 关联 `ota_upgrade_task_log` 表

### 关联设备分组
- 可查看目标分组配置
- 通过 `device_filter` 解析筛选条件

## 进度计算逻辑

### 完成百分比（progress）
```
progress = (success + failed + timeout + cancelled) / total_devices * 100
```

### 成功率（success_rate）
```
success_rate = success / (success + failed + timeout + cancelled) * 100
```

### 平均升级耗时（avg_duration）
- 仅统计成功升级的设备
- 计算所有成功设备的平均耗时

### 预计剩余时间（estimated_remaining）
```
estimated_remaining = avg_duration * pending_count
```

## 优化建议

### 1. 性能优化
- 添加缓存机制（Redis）
- 缓存任务详情和统计数据
- 定期更新缓存

### 2. 数据完整性
- 添加设备升级详细日志表
- 记录每个阶段的详细状态
- 保存错误堆栈信息

### 3. 统计功能增强
- 按时间段统计升级趋势
- 分析失败原因分布
- 统计各地区/型号升级情况

### 4. 查询优化
- 添加组合索引
- 支持更多筛选条件
- 支持导出功能

### 5. 实时监控
- WebSocket 推送实时进度
- 实时统计图表展示
- 异常情况实时告警

## 测试建议

### 功能测试
- 测试查询不同状态的任务
- 测试包含/不包含设备列表
- 测试包含/不包含操作日志
- 测试统计数据准确性

### 边界测试
- 测试不存在的任务 ID
- 测试无效的任务 ID
- 测试空结果
- 测试大量设备数据

### 性能测试
- 测试大量设备下的查询性能
- 测试并发查询
- 测试缓存命中率

### 集成测试
- 测试与任务列表接口的一致性
- 测试与设备统计接口的一致性
- 测试数据实时更新

## 使用示例

### 示例 1：查询任务基本信息
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/detail?task_id=123" \
  -H "Authorization: Bearer <token>"
```

### 示例 2：查询任务详情（含设备列表）
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/detail?task_id=123&with_devices=true" \
  -H "Authorization: Bearer <token>"
```

### 示例 3：查询任务详情（含日志）
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/detail?task_id=123&with_logs=true" \
  -H "Authorization: Bearer <token>"
```

### 示例 4：查询完整任务详情
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/detail?task_id=123&with_devices=true&with_logs=true" \
  -H "Authorization: Bearer <token>"
```

## 总结

OTA 任务详情接口已完整实现，支持：
- ✅ 任务基础信息查询（配置、状态、时间）
- ✅ 任务配置信息（目标版本、升级策略）
- ✅ 目标范围信息（设备总数、筛选条件）
- ✅ 执行统计信息（各状态设备数量）
- ✅ 进度信息（完成率、成功率、预计剩余时间）
- ✅ 设备列表详情（可选）
- ✅ 操作日志关联（可选）
- ✅ 异常信息记录（错误码、错误信息、重试次数）
- ✅ 关联信息查询（产品、固件、日志）

接口已通过编译检查，可以直接使用！🎉
