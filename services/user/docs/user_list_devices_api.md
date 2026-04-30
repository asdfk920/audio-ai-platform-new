# 用户查看绑定设备列表接口文档

## 接口说明

- **接口地址**: `/api/v1/user/device/list`
- **请求方式**: `GET`
- **功能**: 用户查看自己可访问的设备列表，支持分页、条件筛选
- **权限要求**: 需要 JWT 登录认证，可同时查询本人 owner 设备与已接受的 shared 设备

---

## 请求参数

### Header
```
Authorization: Bearer <access_token>
```

### Query Parameters
```
GET /api/v1/user/device/list?page=1&page_size=20&device_name=我的设备&device_sn=SN123&device_model=X1
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1，最小 1 |
| page_size | int | 否 | 每页数量，默认 20，范围 1-100 |
| device_name | string | 否 | 设备名称（模糊搜索，不区分大小写） |
| device_sn | string | 否 | 设备序列号（模糊搜索，不区分大小写） |
| device_model | string | 否 | 设备型号（模糊搜索，不区分大小写） |

### 参数说明
- **分页**: 默认返回第 1 页，每页 20 条，最大支持 100 条/页
- **筛选条件**: 多个条件同时满足（AND 关系）
- **模糊搜索**: 支持子串匹配，不区分大小写（ASCII 字符）
- **排序**: 按绑定时间倒序排列（最新的在前）

---

## 后端处理全流程

### 1. 用户请求获取设备列表
- 用户端发起请求，携带用户 Token
- 后端从 Token 中解析出 user_id

### 2. 后端校验登录态
- 校验 JWT token 有效性
- 校验用户是否存在、账号状态正常
- 未登录或 token 无效则返回未授权

### 3. 查询 owner 与 shared 设备
- 根据 `user_id` 查询 `user_device_bind` 中本人仍绑定的 owner 设备
- 同时查询 `user_device_share` 中本人已接受且未过期的 shared 设备
- 支持按设备名称、SN、型号条件筛选
- 最终统一分页返回

### 4. 关联设备信息
- 根据查询到的 device_sn 批量查询设备表
- 获取设备基础信息：SN、型号、固件版本、硬件版本等

### 5. 关联设备影子（在线状态）
- 从设备表获取设备实时状态：在线/离线、电量、运行状态等
- 设备影子数据与设备表一体化存储

### 6. 数据合并组装
- 将 owner 绑定关系、shared 授权关系、设备信息、设备状态进行合并
- 格式化时间字段
- 为 shared 设备补充 `access_mode`、`role`、`permission_level`

### 7. 返回设备列表
- 按绑定时间倒序排列
- 分页返回结果
- 返回总记录数，便于前端分页展示

### 8. 记录接口访问日志
- 记录用户查询行为
- 用于审计和性能分析

---

## 返回结果

### 成功响应（200）
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 5,
    "list": [
      {
        "device_sn": "SN1234567890",
        "device_name": "我的设备",
        "device_model": "X1 Pro",
        "system_version": "1.0.0",
        "firmware_version": "FW_1.2.3",
        "hardware_version": "HW_2.0",
        "online_status": 1,
        "device_status": 1,
        "bind_time": "2026-04-08 15:30:45",
        "last_active_at": "2026-04-08 18:20:10"
      },
      {
        "device_sn": "SN0987654321",
        "device_name": "客厅音箱",
        "device_model": "X2 Mini",
        "system_version": "1.1.0",
        "firmware_version": "FW_1.3.0",
        "hardware_version": "HW_1.5",
        "online_status": 0,
        "device_status": 1,
        "bind_time": "2026-04-07 10:15:30",
        "last_active_at": "2026-04-08 12:00:00"
      }
    ]
  }
}
```

### 字段说明

#### 分页字段
| 字段 | 类型 | 说明 |
|------|------|------|
| total | int | 总记录数 |
| list | array | 设备列表 |

#### 设备列表项
| 字段 | 类型 | 说明 |
|------|------|------|
| device_sn | string | 设备唯一序列号 |
| device_name | string | 设备别名/昵称 |
| device_model | string | 设备型号 |
| system_version | string | 系统版本 |
| firmware_version | string | 固件版本 |
| hardware_version | string | 硬件版本 |
| online_status | int | 在线状态：0=离线，1=在线 |
| device_status | int | 设备状态：1=启用，2=禁用，3=未激活，4=已报废 |
| bind_time | string | owner 为绑定时间，shared 为授权生效时间 |
| last_active_at | string | 最后活跃时间（可选） |
| access_mode | string | `owner` 或 `shared` |
| role | string | 家庭角色：`owner` / `super_admin` / `member` |
| permission_level | string | `full_control` / `partial_control` / `view_only` |
| permission | string | JSON 字符串，承载动作白名单或只读标记 |
| owner_user_id | int | 设备主人 ID |
| share_id | int | 共享记录 ID，仅 shared 设备返回 |
| family_id | int | 家庭 ID，仅 shared 设备返回 |

### 空列表响应
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 0,
    "list": []
  }
}
```

### 失败响应

#### 401 - 用户未登录
```json
{
  "code": 1004,
  "msg": "登录已过期或无效，请重新登录"
}
```

#### 500 - 系统内部错误
```json
{
  "code": 9001,
  "msg": "查询设备列表失败"
}
```

---

## 数据库查询

### 1. 查询绑定关系（带条件筛选）
```sql
SELECT ub.id, ub.user_id, ub.device_id, ub.sn, COALESCE(ub.alias,'') AS alias,
       ub.status, ub.bound_at, ub.unbound_at,
       d.sn, d.product_key, d.mac, d.firmware_version, d.hardware_version,
       d.ip, d.status, d.online_status, d.last_active_at
FROM public.user_device_bind ub
JOIN public.device d ON ub.device_id = d.id
WHERE ub.user_id = $1 
  AND ub.status = 1
  AND d.sn ILIKE '%SN123%'           -- 可选条件
  AND ub.alias ILIKE '%我的设备%'     -- 可选条件
  AND d.product_key ILIKE '%X1%'     -- 可选条件
ORDER BY ub.bound_at DESC
LIMIT 20 OFFSET 0;
```

### 2. 统计总数
```sql
SELECT COUNT(*)
FROM public.user_device_bind ub
JOIN public.device d ON ub.device_id = d.id
WHERE ub.user_id = $1 
  AND ub.status = 1
  AND d.sn ILIKE '%SN123%'
  AND ub.alias ILIKE '%我的设备%'
  AND d.product_key ILIKE '%X1%';
```

### 3. 批量查询设备信息
```sql
SELECT id, sn, product_key, mac, firmware_version, hardware_version,
       ip, status, online_status, last_active_at
FROM public.device
WHERE sn IN ('SN1234567890', 'SN0987654321', ...);
```

---

## 安全与规则约束

### 1. 权限隔离
- 仅返回当前用户自己绑定的设备
- 不允许越权查看他人设备
- 通过 user_id 强制隔离数据

### 2. 登录态校验
- 用户未登录则直接返回未授权
- Token 过期或无效时拒绝访问

### 3. 空列表处理
- 无绑定设备时返回空列表，不报错
- total=0，list=[]

### 4. 设备影子容错
- 设备影子获取失败不影响列表返回
- 状态字段标记为未知或使用默认值

### 5. 分页限制
- 支持分页查询，防止大数据量
- 每页最大 100 条，防止恶意请求

### 6. 性能优化
- 使用只读事务查询，提高并发性能
- 批量查询设备信息，减少数据库往返次数
- 使用索引优化查询速度

---

## 业务联动

### 用户端
- 刷新设备列表，展示已绑定设备
- 显示设备在线状态、设备信息
- 支持搜索、筛选、分页操作
- 点击设备可进入详情页

### 后台端
- 可查看用户设备绑定情况
- 支持按设备 SN、型号筛选
- 审计用户设备使用情况

### 设备端
- 设备状态实时同步到设备表
- 在线状态定时更新
- 最后活跃时间自动记录

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 查询成功 | - |
| 1004 | Token 无效或过期 | 重新登录获取新 token |
| 9001 | 数据库查询失败 | 联系技术支持 |

---

## 调用示例

### cURL
```bash
curl -X GET "http://localhost:8888/api/v1/user/device/list?page=1&page_size=20&device_name=我的" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### JavaScript
```javascript
async getUserDeviceList(page = 1, pageSize = 20, filters = {}) {
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: pageSize.toString(),
    ...filters
  });
  
  const res = await fetch(`/api/v1/user/device/list?${params}`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${accessToken}`
    }
  });
  
  const result = await res.json();
  if (result.code === 0) {
    console.log('设备列表', result.data);
    return result.data;
  } else {
    console.error('查询失败', result.msg);
    throw new Error(result.msg);
  }
}

// 使用示例
getUserDeviceList(1, 20, { device_name: '我的' });
```

### Vue/React 示例
```javascript
// 组件加载时获取设备列表
onMounted(async () => {
  try {
    const data = await getUserDeviceList(1, 20);
    deviceList.value = data.list;
    total.value = data.total;
  } catch (err) {
    console.error('加载设备列表失败', err);
  }
});

// 分页变化时重新加载
const handlePageChange = (newPage) => {
  getUserDeviceList(newPage, pageSize.value)
    .then(data => {
      deviceList.value = data.list;
    });
};
```

---

## 实现文件清单

```
services/user/
├── internal/
│   ├── types/
│   │   └── types.go                          # DTO 定义（已更新）
│   ├── logic/
│   │   └── list_user_devices_logic.go        # 业务逻辑层（新建）
│   ├── handler/
│   │   └── list_user_devices_handler.go      # API 处理器（已存在）
│   └── repo/dao/
│       └── device_bind_dao.go                # 查询 DAO 方法（已更新）
└── docs/
    └── user_list_devices_api.md              # API 文档（新建）
```

---

## 注意事项

1. **权限校验**: 必须校验用户身份，防止越权查询
2. **分页性能**: 大数据量时使用分页，避免全表扫描
3. **空值处理**: 设备不存在时跳过该条记录，不影响其他数据
4. **时间格式**: 统一使用 `yyyy-MM-dd HH:mm:ss` 格式
5. **索引优化**: 建议在 `user_device_bind(user_id, status)` 和 `device(sn)` 上建立索引
6. **只读事务**: 查询操作使用只读事务，提高并发性能

---

## 性能优化建议

### 1. 数据库索引
```sql
-- 用户绑定关系索引
CREATE INDEX idx_user_bind_user_status ON public.user_device_bind(user_id, status);
CREATE INDEX idx_user_bind_bound_at ON public.user_device_bind(bound_at DESC);

-- 设备 SN 索引
CREATE INDEX idx_device_sn ON public.device(sn);
```

### 2. 查询优化
- 使用 JOIN 代替多次查询
- 批量查询代替循环查询
- 使用只读事务提高并发

### 3. 缓存策略（可选）
- 可缓存用户设备列表（Redis）
- 缓存过期时间：5-10 分钟
- 绑定/解绑操作时清除缓存

---

## 与绑定/解绑接口的关系

| 接口 | 功能 | 数据影响 |
|------|------|----------|
| `/user/device/bind` | 绑定设备 | 新增绑定记录 |
| `/user/device/unbind` | 解绑设备 | 更新绑定状态 |
| `/user/device/list` | 查询设备列表 | 只读查询 |

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
