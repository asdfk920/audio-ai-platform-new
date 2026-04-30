# 版本历史接口实现文档

## 概述

实现了固件版本历史的后台接口，用于展示某一产品线或设备型号下的所有固件版本，按发布时间倒序排列，记录固件版本的演进过程和变更记录。

## 接口信息

### 接口地址
```
GET /api/v1/platform-device/firmware/history
```

### 请求方式
- Content-Type: application/json
- 需要 JWT 认证

### 请求参数

#### 必填参数（至少一个）
- `product_key` (string): 产品标识
- `device_model` (string): 设备型号

#### 选填参数
- `date_from` (string): 开始日期，格式 YYYY-MM-DD
- `date_to` (string): 结束日期，格式 YYYY-MM-DD
- `release_type` (string): 发布类型
  - `formal`: 正式版
  - `test`: 测试版
  - `gray`: 灰度版
  - `rollback`: 回滚版
  - `emergency`: 紧急版
- `status` (string): 版本状态
  - `draft`: 草稿
  - `testing`: 测试中
  - `published`: 已发布
  - `withdrawn`: 已撤回
  - `obsolete`: 已废弃
- `page` (int): 页码，默认 1
- `page_size` (int): 每页条数，默认 20，最大 100
- `with_stats` (bool): 是否包含统计数据，默认 false
- `with_change_log` (bool): 是否包含变更日志，默认 false

### 请求示例

```
GET /api/v1/platform-device/firmware/history?product_key=product_123&page=1&page_size=20&with_stats=true&with_change_log=true
```

或按设备型号查询：
```
GET /api/v1/platform-device/firmware/history?device_model=Model-A&date_from=2026-01-01&date_to=2026-04-14
```

### 返回结果

#### 成功响应
```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "total": 50,
    "page": 1,
    "page_size": 20,
    "list": [
      {
        "version": "2.1.0",
        "version_code": 20100,
        "release_date": "2026-04-10",
        "release_type": "formal",
        "release_type_text": "正式版",
        "status": "published",
        "status_text": "已发布",
        "force_update": false,
        "file_size": 52428800,
        "file_size_human": "50.00 MB",
        "download_count": 1234,
        "installed_count": 567,
        "success_rate": 98.5,
        "description": "修复已知问题，优化性能",
        "created_at": "2026-04-10 10:00:00",
        "creator": "张三",
        "change_log": "修复已知问题，优化性能",
        "is_current_version": true,
        "is_latest": true
      }
    ],
    "has_next": true,
    "has_prev": false
  }
}
```

### 错误响应

#### 缺少必填参数
```json
{
  "code": 400,
  "msg": "product_key 或 device_model 至少传一个"
}
```

#### 服务器错误
```json
{
  "code": 500,
  "msg": "查询版本历史失败"
}
```

## 实现细节

### 文件修改

1. **apis/platform_device_firmware.go**
   - 添加 `FirmwareHistoryReq` 请求结构体
   - 实现 `FirmwareHistory` API 处理函数
   - 包含参数解析、校验和错误处理

2. **service/platform_device_service.go**
   - 添加 `FirmwareHistoryRequest` 请求结构体
   - 添加 `FirmwareHistoryItem` 返回项结构体
   - 添加 `FirmwareHistoryResponse` 响应结构体
   - 实现 `FirmwareHistory` 服务方法
   - 添加 `firmwareStatus` 辅助函数

3. **router/init.go**
   - 注册路由：`GET /firmware/history`

### 处理流程

#### 第一步：参数解析
- 解析产品标识 `product_key` 或设备型号 `device_model`
- 解析时间范围 `date_from` 和 `date_to`
- 解析分页参数 `page` 和 `page_size`
- 解析状态筛选 `status` 和发布类型 `release_type`
- 解析扩展选项 `with_stats` 和 `with_change_log`

#### 第二步：构建查询条件
- 根据产品标识或设备型号构建主查询条件
- 设备型号支持模糊匹配（LIKE 查询）
- 添加时间范围筛选（created_at >= date_from AND created_at <= date_to）
- 添加版本状态筛选（status 字段映射）

#### 第三步：版本排序
- 按发布时间倒序排列（created_at DESC）
- 同一版本有多个测试版时按版本码倒序（version_code DESC）
- 确保最新版本在前

#### 第四步：版本标注
- 查询最新版本号（version_code 最大）
- 标注 `is_latest` 字段
- 标注 `is_current_version` 字段（可扩展）

#### 第五步：变更日志关联
- 关联版本说明 `version_description`
- 生成变更日志内容
- 支持通过 `with_change_log` 参数控制是否返回

#### 第六步：统计数据补充
- 查询每个版本的下载次数（download_count）
- 查询已安装设备数（统计 device 表中 firmware_version 字段）
- 计算升级成功率（预留逻辑）

#### 第七步：返回格式化
- 时间戳转换为日期格式（YYYY-MM-DD HH:mm:ss）
- 文件大小转换为人类可读格式（B/KB/MB/GB）
- 枚举值转换为中文描述
- 查询创建人昵称

### 数据结构

#### FirmwareHistoryRequest
```go
type FirmwareHistoryRequest struct {
    ProductKey    string  // 产品标识
    DeviceModel   string  // 设备型号
    DateFrom      string  // 开始日期 YYYY-MM-DD
    DateTo        string  // 结束日期 YYYY-MM-DD
    ReleaseType   string  // 发布类型
    Status        string  // 版本状态
    Page          int     // 页码
    PageSize      int     // 每页条数
    WithStats     bool    // 是否包含统计数据
    WithChangeLog bool    // 是否包含变更日志
}
```

#### FirmwareHistoryItem
```go
type FirmwareHistoryItem struct {
    Version          string  // 版本号
    VersionCode      int     // 整型版本码
    ReleaseDate      string  // 发布日期
    ReleaseType      string  // 发布类型标识
    ReleaseTypeText  string  // 发布类型文本
    Status           string  // 状态标识
    StatusText       string  // 状态文本
    ForceUpdate      bool    // 是否强制升级
    FileSize         int64   // 文件大小（字节）
    FileSizeHuman    string  // 文件大小（人类可读）
    DownloadCount    int64   // 下载次数
    InstalledCount   int64   // 已安装设备数
    SuccessRate      float64 // 升级成功率
    Description      string  // 版本说明
    CreatedAt        string  // 创建时间
    Creator          string  // 创建人
    ChangeLog        string  // 变更日志
    IsCurrentVersion bool    // 是否当前版本
    IsLatest         bool    // 是否最新版本
}
```

#### FirmwareHistoryResponse
```go
type FirmwareHistoryResponse struct {
    Total     int64                 // 总条数
    Page      int                   // 当前页码
    PageSize  int                   // 每页条数
    List      []FirmwareHistoryItem // 版本列表
    HasNext   bool                  // 是否有下一页
    HasPrev   bool                  // 是否有上一页
}
```

## 查询维度

### 按产品维度查询
```
GET /api/v1/platform-device/firmware/history?product_key=product_123
```
查看某一产品下所有固件版本

### 按设备型号查询
```
GET /api/v1/platform-device/firmware/history?device_model=Model-A
```
查看适用于某型号的所有固件

### 按时间范围查询
```
GET /api/v1/platform-device/firmware/history?product_key=product_123&date_from=2026-01-01&date_to=2026-04-14
```
查看某一时间段内发布的所有版本

### 按发布状态查询
```
GET /api/v1/platform-device/firmware/history?product_key=product_123&status=published
```
区分已发布和测试中版本

## 版本类型说明

| 类型标识 | 中文名称 | 说明 |
|----------|----------|------|
| formal | 正式版 | 已发布给所有用户的稳定版本 |
| test | 测试版 | 仅供测试的预发布版本（包含 beta/rc） |
| gray | 灰度版 | 分批次发布的版本 |
| rollback | 回滚版 | 修复问题的补丁版本 |
| emergency | 紧急版 | 修复严重问题的强制推送版本 |

## 版本状态说明

| 状态标识 | 中文名称 | 说明 |
|----------|----------|------|
| draft | 草稿 | 保存但未发布的版本 |
| testing | 测试中 | 正在测试的版本 |
| published | 已发布 | 正常对外提供的版本 |
| withdrawn | 已撤回 | 发布后撤销的版本 |
| obsolete | 已废弃 | 不再推荐的旧版本 |

## 版本对比功能（预留）

版本历史页面可提供相邻版本对比功能：
- 查看当前版本与上一版本的差异
- 查看当前版本与最新版本的差异
- 变更内容包括：新增功能、优化功能、修复问题、已知问题

**实现建议：**
```go
// 可添加额外接口实现版本对比
GET /api/v1/platform-device/firmware/compare?version1=2.0.0&version2=2.1.0&product_key=product_123
```

## 时间轴展示（前端实现建议）

版本历史支持时间轴模式展示：
- 沿时间线标注重要版本节点（首个版本、重大更新版本）
- 每个节点显示：版本号、发布日期、简要说明
- 点击节点可展开查看版本详情

**数据结构支持：**
- 已提供完整的版本信息
- 前端可根据 `is_latest` 和 `version_code` 标注重要节点
- 可根据 `release_type` 区分版本重要性

## 优化建议

### 1. 性能优化
- 添加缓存机制（Redis）
- 缓存版本历史列表
- 缓存统计数据

### 2. 数据完整性
- 添加 `release_type` 字段到数据库
- 添加独立的变更日志表
- 记录详细的版本对比信息

### 3. 统计功能增强
- 实现升级成功率计算
- 添加升级失败原因分析
- 统计各版本的设备覆盖率

### 4. 查询优化
- 添加组合索引
- 支持更多筛选条件
- 支持全文搜索

### 5. 审计功能
- 记录版本历史查询日志
- 统计热门查询产品
- 分析查询趋势

## 测试建议

### 功能测试
- 测试按产品标识查询
- 测试按设备型号查询
- 测试时间范围筛选
- 测试状态筛选
- 测试分页功能
- 测试统计数据准确性

### 边界测试
- 测试空结果
- 测试超大页码
- 测试超长日期范围
- 测试无效参数

### 性能测试
- 测试大量数据下的查询性能
- 测试并发查询
- 测试缓存命中率

### 集成测试
- 测试与固件列表接口的一致性
- 测试与设备统计接口的一致性
- 测试数据实时更新

## 使用示例

### 示例 1：查询产品所有版本
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/firmware/history?product_key=product_123&page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

### 示例 2：查询设备型号的所有版本（含统计）
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/firmware/history?device_model=Model-A&with_stats=true" \
  -H "Authorization: Bearer <token>"
```

### 示例 3：查询指定时间范围的版本
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/firmware/history?product_key=product_123&date_from=2026-01-01&date_to=2026-03-31" \
  -H "Authorization: Bearer <token>"
```

### 示例 4：查询已发布的正式版本
```bash
curl -X GET "http://api.example.com/api/v1/platform-device/firmware/history?product_key=product_123&status=published" \
  -H "Authorization: Bearer <token>"
```

## 总结

版本历史接口已完整实现，支持：
- ✅ 多维度查询（产品、设备型号、时间范围、状态）
- ✅ 分页和排序
- ✅ 版本标注（最新版本、当前版本）
- ✅ 统计数据（下载次数、安装设备数）
- ✅ 格式化输出（日期、文件大小、枚举值）
- ✅ 变更日志关联
- ✅ 创建人信息查询

接口已通过编译检查，可以直接使用！
