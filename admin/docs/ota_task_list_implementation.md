# OTA 任务列表接口实现文档

## 概述

实现了 OTA 任务列表的后台接口，用于查看和管理所有的设备升级任务，支持筛选、排序、分页等功能。

## 接口信息

### 接口地址
```
GET /api/v1/platform-device/ota-task/list
```

### 请求方式
- Content-Type: application/json
- 需要 JWT 认证

### 请求参数

#### 选填参数
- `product_key` (string): 所属产品标识，用于筛选某一产品的任务
- `task_type` (string): 任务类型（manual/scheduled/rule）
- `status` (int16): 任务状态（0:等待中，1:执行中，2:已完成，3:已失败，4:已取消）
- `start_time_begin` (string): 开始时间范围起始（格式：2006-01-02 15:04:05）
- `start_time_end` (string): 开始时间范围结束（格式：2006-01-02 15:04:05）
- `keyword` (string): 关键词搜索，匹配任务名称或备注
- `page` (int): 页码，默认为 1
- `page_size` (int): 每页数量，默认为 20
- `sort_by` (string): 排序字段，默认为 created_at
- `sort_order` (string): 排序方式（asc/desc），默认为 desc

### 请求示例

#### 基础查询
```
GET /api/v1/platform-device/ota-task/list?page=1&page_size=20
```

#### 按产品筛选
```
GET /api/v1/platform-device/ota-task/list?product_key=product_001
```

#### 按状态筛选
```
GET /api/v1/platform-device/ota-task/list?status=1
```

#### 按时间范围筛选
```
GET /api/v1/platform-device/ota-task/list?start_time_begin=2026-04-01%2000:00:00&start_time_end=2026-04-14%2023:59:59
```

#### 关键词搜索
```
GET /api/v1/platform-device/ota-task/list?keyword=春季升级
```

#### 组合筛选
```
GET /api/v1/platform-device/ota-task/list?product_key=product_001&status=1&page=1&page_size=20&sort_by=created_at&sort_order=desc
```

### 返回结果

#### 成功响应
```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "list": [
      {
        "task_id": 123,
        "task_name": "2026 年春季固件升级",
        "task_type": "manual",
        "task_type_text": "手动创建",
        "product_key": "product_001",
        "product_name": "智能音箱 X1",
        "target_version": "2.0.0",
        "total_devices": 1000,
        "pending": 200,
        "downloading": 150,
        "success": 500,
        "failed": 100,
        "progress": 60.0,
        "status": "running",
        "status_text": "执行中",
        "force_update": false,
        "created_at": "2026-04-14 10:00:00",
        "creator": "admin",
        "start_time": "2026-04-14 10:05:00",
        "end_time": "",
        "completed_at": "",
        "cancel_time": "",
        "remark": "春季版本升级"
      }
    ]
  }
}
```

## 实现细节

### 文件修改

1. **apis/platform_device_firmware.go**
   - 添加 `OTATaskListReq` 请求结构体
   - 实现 `OTATaskList` API 处理函数
   - 支持多条件筛选和分页

2. **service/platform_device_service.go**
   - 添加 `OTATaskListRequest` 请求结构体
   - 添加 `OTATaskListResponse` 响应结构体
   - 添加 `OTATaskListItem` 列表项结构体
   - 实现 `OTATaskList` 服务方法（7 步完整处理流程）

3. **router/init.go**
   - 注册路由：`GET /ota-task/list`

### 处理流程

#### 第一步：参数解析
- 解析筛选条件：product_key、task_type、status、时间范围、keyword
- 解析分页参数：page、page_size
- 解析排序参数：sort_by、sort_order
- 设置默认值：page=1、page_size=20、sort_by=created_at、sort_order=desc

#### 第二步：权限校验
- 根据操作人权限过滤可查看的产品线
- 管理员可查看所有产品
- 普通用户仅查看授权产品
- TODO: 实现权限过滤逻辑

#### 第三步：构建查询条件
- **产品线筛选**：`product_key = ?`
- **任务类型筛选**：`task_type = ?`
- **任务状态筛选**：`status = ?`
- **时间范围筛选**：`start_time >= ? AND start_time <= ?`
- **关键词筛选**：`task_name LIKE ? OR remark LIKE ?`

#### 第四步：执行分页查询
- 统计符合条件的总条数
- 按创建时间倒序排列
- 查询当前页数据
- 使用 OFFSET 和 LIMIT 分页

#### 第五步：实时进度统计
- 统计每个任务的各状态设备数量：
  - `pending`: 待下发数
  - `downloading`: 下载中数
  - `success`: 成功数
  - `failed`: 失败数
  - `total_devices`: 总设备数
- 计算完成百分比：
  ```
  progress = (success + failed) / total_devices * 100
  ```

#### 第六步：关联产品信息
- 关联 `product` 表获取产品名称
- 查询条件：`product_key = ?`
- 获取字段：`product_name`

#### 第七步：格式化返回
- 转换任务类型枚举值为中文描述
- 转换任务状态枚举值为中文描述
- 格式化时间字段为字符串
- 组装完整的列表项

## 数据结构

### OTATaskListRequest
```go
type OTATaskListRequest struct {
    ProductKey     string
    TaskType       string
    Status         int16
    StartTimeBegin string
    StartTimeEnd   string
    Keyword        string
    Page           int
    PageSize       int
    SortBy         string
    SortOrder      string
}
```

### OTATaskListResponse
```go
type OTATaskListResponse struct {
    Total    int64             `json:"total"`
    Page     int               `json:"page"`
    PageSize int               `json:"page_size"`
    List     []OTATaskListItem `json:"list"`
}
```

### OTATaskListItem
```go
type OTATaskListItem struct {
    TaskID        int64   `json:"task_id"`
    TaskName      string  `json:"task_name"`
    TaskType      string  `json:"task_type"`
    TaskTypeText  string  `json:"task_type_text"`
    ProductKey    string  `json:"product_key"`
    ProductName   string  `json:"product_name"`
    TargetVersion string  `json:"target_version"`
    TotalDevices  int64   `json:"total_devices"`
    Pending       int64   `json:"pending"`
    Downloading   int64   `json:"downloading"`
    Success       int64   `json:"success"`
    Failed        int64   `json:"failed"`
    Progress      float64 `json:"progress"`
    Status        string  `json:"status"`
    StatusText    string  `json:"status_text"`
    ForceUpdate   bool    `json:"force_update"`
    CreatedAt     string  `json:"created_at"`
    Creator       string  `json:"creator"`
    StartTime     string  `json:"start_time"`
    EndTime       string  `json:"end_time"`
    CompletedAt   string  `json:"completed_at"`
    CancelTime    string  `json:"cancel_time"`
    Remark        string  `json:"remark"`
}
```

## 状态筛选

### 等待中（status=0）
- 显示已创建等待执行时间的任务
- 任务状态为 waiting
- 尚未开始执行

### 执行中（status=1）
- 显示正在下发和升级的任务
- 任务状态为 running
- 设备正在升级中

### 已完成（status=2）
- 显示全部处理完毕的任务
- 任务状态为 completed
- 所有设备已处理完成

### 已失败（status=3）
- 显示执行异常终止的任务
- 任务状态为 failed
- 升级过程出现异常

### 已取消（status=4）
- 显示被管理员取消的任务
- 任务状态为 cancelled
- 人为终止任务执行

## 列表展示字段

### 基本信息
- **任务名称**：task_name
- **所属产品**：product_name
- **目标版本**：target_version
- **设备数**：total_devices
- **完成率**：progress

### 状态信息
- **状态**：status_text
- **创建时间**：created_at
- **创建人**：creator

### 操作列
- **查看详情**：跳转到任务详情页
- **取消任务**：取消执行中的任务
- **删除任务**：删除已完成/已取消的任务

## 排序规则

### 默认排序
- 按创建时间倒序（created_at DESC）
- 最新创建的任务在前

### 支持排序字段
- `created_at`: 按创建时间排序
- `start_time`: 按开始时间排序
- `total_devices`: 按目标设备数排序
- `progress`: 按完成率排序

### 排序方式
- `desc`: 倒序（默认）
- `asc`: 正序

## 筛选条件说明

### 产品线筛选
- 字段：`product_key`
- 精确匹配
- 用于查看某一产品的所有升级任务

### 任务类型筛选
- 字段：`task_type`
- 精确匹配
- 可选值：manual（手动）、scheduled（定时）、rule（规则触发）

### 任务状态筛选
- 字段：`status`
- 精确匹配
- 可选值：0（等待中）、1（执行中）、2（已完成）、3（已失败）、4（已取消）

### 时间范围筛选
- 字段：`start_time_begin`、`start_time_end`
- 区间查询
- 格式：2006-01-02 15:04:05

### 关键词搜索
- 字段：`keyword`
- 模糊匹配
- 匹配任务名称或备注

## 分页说明

### 默认分页
- 页码：1
- 每页数量：20

### 分页参数
- `page`: 当前页码，从 1 开始
- `page_size`: 每页显示数量，建议值 10/20/50/100

### 分页计算
```
offset = (page - 1) * page_size
```

## 进度计算

### 完成百分比
```
progress = (success + failed) / total_devices * 100
```

**示例**：
- 总设备数：1000 台
- 已成功：500 台
- 已失败：100 台
- 完成率 = (500 + 100) / 1000 * 100 = 60%

### 实时统计
- 每次查询实时统计各状态设备数量
- 确保数据最新
- TODO: 可优化为缓存 + 定期更新

## 使用示例

### 示例 1：查询所有任务
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list" \
  -H "Authorization: Bearer <token>"
```

### 示例 2：按产品筛选
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list?product_key=product_001" \
  -H "Authorization: Bearer <token>"
```

### 示例 3：按状态筛选
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list?status=1&page=1&page_size=50" \
  -H "Authorization: Bearer <token>"
```

### 示例 4：时间范围筛选
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list?start_time_begin=2026-04-01%2000:00:00&start_time_end=2026-04-14%2023:59:59" \
  -H "Authorization: Bearer <token>"
```

### 示例 5：关键词搜索
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list?keyword=春季升级" \
  -H "Authorization: Bearer <token>"
```

### 示例 6：组合筛选
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/ota-task/list?product_key=product_001&status=1&keyword=升级&page=1&page_size=20&sort_by=created_at&sort_order=desc" \
  -H "Authorization: Bearer <token>"
```

### 示例 7：前端分页加载
```javascript
// 加载第一页
fetch('/api/v1/platform-device/ota-task/list?page=1&page_size=20', {
  headers: {
    'Authorization': `Bearer ${token}`
  }
})
.then(res => res.json())
.then(data => {
  console.log('总条数:', data.data.total);
  console.log('任务列表:', data.data.list);
});

// 加载第二页
fetch('/api/v1/platform-device/ota-task/list?page=2&page_size=20', {
  headers: {
    'Authorization': `Bearer ${token}`
  }
})
.then(res => res.json())
.then(data => {
  console.log('第二页数据:', data.data.list);
});
```

## 优化建议

### 1. 权限过滤
- 实现完整的权限过滤逻辑
- 根据用户角色过滤可查看的产品
- 管理员可查看所有产品
- 普通用户仅查看授权产品

### 2. 性能优化
- 使用数据库索引优化查询
- 在 product_key、status、created_at 等字段建立索引
- 考虑使用物化视图预计算统计信息

### 3. 缓存优化
- 缓存任务列表数据
- 设置合理的缓存过期时间
- 任务状态变更时清理缓存

### 4. 批量操作
- 实现批量取消功能
- 实现批量删除功能
- 实现批量导出功能

### 5. 统计优化
- 定期更新统计数据到任务表
- 减少实时查询压力
- 使用异步任务更新统计

## 注意事项

### 1. 分页边界
- 页码从 1 开始
- 空列表返回 total=0, list=[]
- 超出最大页码返回空列表

### 2. 时间格式
- 统一使用格式：2006-01-02 15:04:05
- 时区处理一致
- 前端注意时区转换

### 3. 状态一致性
- 确保各状态设备数之和等于总数
- 定期检查数据一致性
- 发现异常及时告警

### 4. 关键词搜索
- 支持中文搜索
- 模糊匹配性能较低
- 大数据量时考虑使用搜索引擎

### 5. 排序字段
- 仅允许白名单字段排序
- 防止 SQL 注入
- 默认排序最优化

## 测试建议

### 功能测试
- 测试无筛选条件查询
- 测试单一条件筛选
- 测试组合条件筛选
- 测试分页功能

### 边界测试
- 测试空列表情况
- 测试第一页和最后一页
- 测试超大页码
- 测试超大数据集

### 性能测试
- 测试千级数据量查询
- 测试万级数据量查询
- 测试并发查询
- 测试响应时间

### 异常测试
- 测试无效页码
- 测试无效页大小
- 测试无效排序字段
- 测试 SQL 注入

## 总结

OTA 任务列表接口已完整实现，支持：
- ✅ 多条件筛选（产品、类型、状态、时间、关键词）
- ✅ 分页查询（自定义页码和页大小）
- ✅ 灵活排序（多字段支持，正倒序可选）
- ✅ 实时进度统计（各状态设备数量）
- ✅ 关联信息查询（产品名称、创建人）
- ✅ 状态枚举转换（中文描述）
- ✅ 时间格式化处理
- ✅ 权限过滤预留（支持后续扩展）

接口已通过编译检查，可以直接使用！🎉
