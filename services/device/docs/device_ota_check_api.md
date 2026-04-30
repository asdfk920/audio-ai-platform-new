# 设备OTA版本检查 API 文档

## 接口说明
- **接口地址**: `/api/v1/device/ota/check`
- **请求方式**: POST
- **功能**: 设备检查是否有可用的固件升级版本，支持灰度升级策略
- **认证**: JWT Token（设备认证后）

## 请求参数

### Header
```
Authorization: Bearer <device_access_token>
Content-Type: application/json
```

### Body
```json
{
  "device_sn": "SN1234567890",
  "firmware_version": "FW_1.0.0",
  "device_model": "X1",
  "device_type": "speaker"
}
```

### 参数说明
| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| device_sn | string | 是 | 设备序列号 | "SN1234567890" |
| firmware_version | string | 是 | 当前固件版本 | "FW_1.0.0" |
| device_model | string | 是 | 设备型号 | "X1" |
| device_type | string | 否 | 设备类型 | "speaker" |

## 成功响应（200）

### 需要升级
```json
{
  "need_upgrade": true,
  "latest_version": "FW_2.0.1",
  "upgrade_type": "optional",
  "changelog": "1. 优化音频播放效果\n2. 修复已知问题\n3. 提升系统稳定性",
  "download_url": "https://firmware.example.com/x1/FW_2.0.1.bin",
  "file_size": 1024000,
  "file_md5": "a1b2c3d4e5f6789012345678901234567",
  "upgrade_tips": "发现可选升级版本 FW_2.0.1，建议升级以获得更好的使用体验",
  "force_upgrade": false,
  "gray_scale": true,
  "gray_percent": 30,
  "check_time": 1775692800
}
```

### 无需升级
```json
{
  "need_upgrade": false,
  "check_time": 1775692800
}
```

### 响应字段说明
| 字段名 | 类型 | 说明 |
|--------|------|------|
| need_upgrade | bool | 是否需要升级 |
| latest_version | string | 最新版本号（need_upgrade=true时返回） |
| upgrade_type | string | 升级类型：force/optional（need_upgrade=true时返回） |
| changelog | string | 更新日志（need_upgrade=true时返回） |
| download_url | string | 固件下载地址（need_upgrade=true时返回） |
| file_size | int64 | 文件大小（字节）（need_upgrade=true时返回） |
| file_md5 | string | 文件MD5校验值（need_upgrade=true时返回） |
| upgrade_tips | string | 升级提示信息（need_upgrade=true时返回） |
| force_upgrade | bool | 是否强制升级（need_upgrade=true时返回） |
| gray_scale | bool | 是否灰度升级（need_upgrade=true时返回） |
| gray_percent | int32 | 灰度比例（0-100）（need_upgrade=true时返回） |
| check_time | int64 | 检查时间戳 |

## 错误响应

### 400 参数错误
```json
{
  "code": 400,
  "message": "设备 SN 不能为空"
}
```

### 401 未授权
```json
{
  "code": 401,
  "message": "设备认证失败"
}
```

### 403 设备禁用
```json
{
  "code": 403,
  "message": "设备已禁用，无法检查OTA"
}
```

### 404 设备不存在
```json
{
  "code": 404,
  "message": "设备不存在"
}
```

### 500 系统错误
```json
{
  "code": 500,
  "message": "系统内部错误"
}
```

## 业务处理流程

### 1. 设备身份验证
- 从 JWT Token 解析设备身份
- 校验设备存在且状态正常（未禁用）

### 2. 固件版本查询
- 根据设备型号查询最新可用固件
- 过滤已禁用和未发布的固件版本

### 3. 版本比较
- 比较设备当前版本与最新版本
- 判断是否需要升级（新版本 > 当前版本）

### 4. 灰度策略判断
- 检查固件是否启用灰度升级
- 基于设备SN计算哈希值判断是否在灰度范围内
- 灰度比例控制升级范围（0-100%）

### 5. 升级信息组装
- 根据升级类型生成提示信息
- 提供完整的升级包信息
- 记录检查日志用于统计分析

## 灰度升级策略

### 灰度算法
- 基于设备SN计算稳定的哈希值
- 哈希值取模100得到0-99的数值
- 判断数值是否小于灰度比例

### 灰度控制
- **0%**: 所有设备都不升级
- **100%**: 所有设备都升级
- **30%**: 约30%的设备在灰度范围内

### 灰度优势
- 降低升级风险
- 支持渐进式发布
- 便于问题排查和回滚

## 升级类型说明

### 强制升级（force）
- 设备必须升级，无法跳过
- 通常用于安全漏洞修复
- 升级提示语气较强

### 可选升级（optional）
- 设备可以选择是否升级
- 通常用于功能优化和体验提升
- 升级提示语气较温和

## 使用示例

### 检查OTA升级
```bash
curl -X POST \
  http://localhost:8888/api/v1/device/ota/check \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_sn": "SN1234567890",
    "firmware_version": "FW_1.0.0",
    "device_model": "X1",
    "device_type": "speaker"
  }'
```

### 响应示例
```json
{
  "need_upgrade": true,
  "latest_version": "FW_2.0.1",
  "upgrade_type": "optional",
  "changelog": "1. 优化音频播放效果\n2. 修复已知问题\n3. 提升系统稳定性",
  "download_url": "https://firmware.example.com/x1/FW_2.0.1.bin",
  "file_size": 1024000,
  "file_md5": "a1b2c3d4e5f6789012345678901234567",
  "upgrade_tips": "发现可选升级版本 FW_2.0.1，建议升级以获得更好的使用体验",
  "force_upgrade": false,
  "gray_scale": true,
  "gray_percent": 30,
  "check_time": 1775692800
}
```

## 注意事项

1. **版本比较**：使用语义化版本号比较，确保升级逻辑正确
2. **灰度稳定性**：灰度算法应保证同一设备在不同时间检查结果一致
3. **下载安全**：固件下载地址应使用HTTPS，确保文件完整性
4. **限流控制**：同一设备短时间频繁检查应做限流处理
5. **日志记录**：所有OTA检查都应记录日志，便于问题追踪

## 扩展功能

### 版本回滚
- 支持特定条件下的版本回滚
- 回滚包与升级包分离管理
- 回滚策略配置

### 差分升级
- 支持增量升级包
- 减少下载流量消耗
- 提升升级效率

### 多版本管理
- 支持多个历史版本并存
- 版本生命周期管理
- 版本依赖关系处理

### 升级统计
- 升级成功率统计
- 设备升级分布分析
- 升级问题反馈收集