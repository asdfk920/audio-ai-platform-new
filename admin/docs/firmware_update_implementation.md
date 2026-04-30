# 固件信息修改接口实现文档

## 概述

实现了固件信息修改的后台接口，用于更新固件的元数据信息，包括版本说明、升级策略、适用型号、启用状态等，不涉及固件包文件本身的更换。

## 接口信息

### 接口地址
```
POST /api/v1/platform-device/firmware/update
```

### 请求方式
- Content-Type: application/json
- 需要 JWT 认证

### 请求参数

#### 必填参数（二选一）
- `firmware_id` (int64): 固件 ID
- 或 `product_key` (string) + `version` (string): 产品标识 + 版本号

#### 选填参数
- `version_description` (string): 版本说明文字
- `device_models` ([]string): 适用设备型号列表
- `force_update` (bool): 是否强制升级
- `min_sys_version` (string): 最低系统版本要求
- `status` (int16): 固件状态 (1=启用，2=禁用)
- `tags` ([]string): 标签列表
- `confirm` (bool): 确认标识

### 请求示例

```json
{
  "firmware_id": 123,
  "version_description": "修复已知问题，优化性能",
  "device_models": ["Model-A", "Model-B"],
  "force_update": true,
  "min_sys_version": "2.0.0",
  "status": 1,
  "tags": ["hotfix", "performance"],
  "confirm": true
}
```

### 返回结果

#### 成功响应
```json
{
  "code": 200,
  "msg": "固件信息更新成功",
  "data": {
    "firmware_id": 123,
    "updated_fields": ["version_description", "force_update", "status"],
    "updated_at": "2026-04-14 10:30:00",
    "message": "固件信息更新成功"
  }
}
```

#### 错误响应

1. 固件不存在
```json
{
  "code": 404,
  "msg": "固件记录不存在"
}
```

2. 版本号被修改
```json
{
  "code": 400,
  "msg": "版本号不可修改，请重新上传固件"
}
```

3. 关联进行中任务
```json
{
  "code": 400,
  "msg": "固件关联进行中的升级任务，部分字段修改受限"
}
```

4. 权限不足
```json
{
  "code": 401,
  "msg": "请先登录"
}
```

## 实现细节

### 文件修改

1. **apis/platform_device_firmware.go**
   - 添加 `FirmwareUpdateReq` 请求结构体
   - 实现 `FirmwareUpdate` API 处理函数
   - 包含参数校验、权限校验、错误处理

2. **service/platform_device_service.go**
   - 添加错误定义：
     - `ErrFirmwareVersionProtected`: 版本号不可修改
     - `ErrFirmwareTaskConflict`: 固件关联进行中的升级任务
     - `ErrFirmwareUpdatePermission`: 没有权限修改该固件
   - 添加 `FirmwareUpdateRequest` 和 `FirmwareUpdateResponse` 结构体
   - 实现 `FirmwareUpdate` 服务方法，包含完整的 8 步处理流程

3. **router/init.go**
   - 注册路由：`POST /firmware/update`

### 处理流程

#### 第一步：参数解析
- 根据 `firmware_id` 或 `product_key + version` 定位目标记录
- 解析请求体中的修改字段

#### 第二步：权限校验
- 验证操作人是否已登录（通过 JWT）
- 后续可扩展为检查具体的固件管理权限

#### 第三步：版本号保护校验
- 检测请求中是否包含版本号修改
- 若尝试修改版本号，返回 `ErrFirmwareVersionProtected` 错误

#### 第四步：字段冲突校验
- 检查是否有禁止修改的字段（文件大小、MD5 等）
- 这些字段不在更新参数中，天然受到保护

#### 第五步：任务关联检查
- 查询 `ota_upgrade_task` 表，检查是否有关联的进行中升级任务
- 若存在进行中任务且修改了 `force_update` 或 `min_sys_version`，返回 `ErrFirmwareTaskConflict` 错误

#### 第六步：数据更新
- 构建更新字段映射
- 仅更新允许修改的字段：
  - `version_description`
  - `device_models`
  - `force_update`
  - `min_sys_version`
  - `status`
  - `tags`
- 更新 `updated_at` 时间戳

#### 第七步：缓存清理
- 预留缓存清理逻辑（TODO 注释）
- 可清理 `ota_firmware:{id}` 和 `ota_firmware_list:{product_key}` 缓存

#### 第八步：日志记录
- 预留日志记录逻辑（TODO 注释）
- 可记录到 `ota_firmware_log` 或 `system_operation_log` 表
- 包含修改人、修改时间、修改前后字段值

## 可修改字段说明

| 字段 | 类型 | 说明 | 业务影响 |
|------|------|------|----------|
| version_description | string | 版本说明文字 | 仅影响展示信息 |
| device_models | []string | 适用设备型号列表 | 影响后续升级任务的目标范围 |
| force_update | bool | 是否强制升级 | 影响已下载设备的升级策略 |
| min_sys_version | string | 最低系统版本要求 | 影响可升级设备范围 |
| status | int16 | 固件状态 (1=启用，2=禁用) | 影响固件是否可被任务引用 |
| tags | []string | 标签列表 | 用于分类管理 |

## 不可修改字段

以下字段禁止修改，如需修改需重新上传固件：
- 固件包文件本身
- 版本号 (`version`)
- 整型版本码 (`version_code`)
- 产品标识 (`product_key`)
- 文件大小 (`file_size`)
- MD5 校验值 (`file_md5`)

## 异常处理

| 异常场景 | 错误码 | 提示信息 |
|----------|--------|----------|
| 固件不存在 | 404 | 固件记录不存在 |
| 版本号被修改 | 400 | 版本号不可修改，请重新上传固件 |
| 关联进行中任务 | 400 | 固件关联进行中的升级任务，部分字段修改受限 |
| 权限不足 | 401 | 请先登录 |
| 字段值格式错误 | 400 | 请求参数格式错误 |

## 后续优化建议

1. **权限细化**
   - 实现基于角色的固件管理权限检查
   - 区分查看权限和修改权限

2. **日志记录**
   - 实现完整的操作日志记录到数据库
   - 记录修改前后的字段值对比

3. **缓存管理**
   - 实现 Redis 缓存清理逻辑
   - 确保数据一致性

4. **任务关联检查增强**
   - 检查更多可能影响任务执行的字段
   - 提供更详细的冲突提示信息

5. **审计功能**
   - 添加固件修改历史记录表
   - 支持查看固件信息变更历史

## 测试建议

1. **功能测试**
   - 测试修改各个可修改字段
   - 测试同时修改多个字段
   - 测试不修改任何字段的情况

2. **异常测试**
   - 测试修改版本号（应拒绝）
   - 测试修改进行中任务关联的固件
   - 测试未登录访问接口

3. **权限测试**
   - 测试不同角色用户的访问权限
   - 测试跨产品修改固件

4. **并发测试**
   - 测试并发修改同一固件
   - 验证数据一致性
