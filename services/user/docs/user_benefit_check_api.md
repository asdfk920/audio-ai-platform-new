# 用户会员权益校验接口文档

## 接口说明

- **接口地址**: `/api/v1/user/benefit/check`
- **请求方式**: `POST`
- **功能**: 校验用户是否拥有指定会员权益，用于业务接口权限控制
- **权限要求**: 需要 JWT 登录认证
- **使用场景**: 内容播放、高音质、专属功能、设备限制等业务场景

---

## 请求参数

### Header
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

### Body
```json
{
  "benefit_code": "content_vip"
}
```

### 参数说明
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| benefit_code | string | 是 | 权益标识，如 `content_vip`、`high_quality`、`unlimited_device` |

### 常见权益标识
| 权益标识 | 说明 | 使用场景 |
|----------|------|----------|
| content_vip | 内容会员 | VIP 内容播放、付费音频 |
| high_quality | 高音质 | 无损音质、Hi-Res 播放 |
| unlimited_device | 无限设备 | 不限制绑定设备数量 |
| offline_download | 离线下载 | 下载音频到本地 |
| ad_free | 免广告 | 跳过片头广告 |
| priority_download | 优先下载 | 高速下载通道 |

---

## 后端处理全流程

### 1. 用户请求权益校验
- 业务接口（内容播放、高音质等）触发权益校验
- 后端从用户 Token 解析出 user_id
- 传入待校验的权益标识（如 content_vip）

### 2. 校验用户合法性
- 查询用户信息，校验用户存在
- 校验用户账号状态正常（status=1）
- 用户不存在或异常则拒绝

### 3. 查询用户会员信息
- 查询用户会员档案：会员等级、有效期、状态
- 查询是否有效会员（未过期、未冻结）
- 查询是否为永久会员

### 4. 查询会员权益列表
- 根据会员等级查询该等级所拥有的全部权益列表
- 从 `member_level_benefit` 表关联 `member_benefit` 表

### 5. 判断是否包含请求权益
- 遍历用户权益列表，判断是否包含请求的 benefit_code
- 支持精确匹配

### 6. 判断会员有效期
- 永久会员：直接通过
- 有效期会员：检查是否在有效期内
- 已过期会员：拒绝

### 7. 判断特殊权益
- 检查是否有单独赠送的权益（体验卡、活动赠送）
- 检查是否在体验期/活动期内
- 特殊权益优先级高于会员等级

### 8. 生成校验结果
- 校验通过：has_permission=true，reason="校验通过"
- 校验不通过：has_permission=false，返回具体原因

### 9. 返回结果给调用方
- 业务接口根据结果放行或拒绝
- 返回权限不足时携带具体原因

---

## 返回结果

### 校验通过（200）
```json
{
  "code": 0,
  "msg": "校验成功",
  "data": {
    "user_id": 123,
    "benefit_code": "content_vip",
    "has_permission": true,
    "reason": "校验通过",
    "level_code": "vip_annual",
    "expire_at": 1775692800,
    "is_permanent": false,
    "subscription_active": true
  }
}
```

### 校验不通过 - 非会员（200）
```json
{
  "code": 0,
  "msg": "校验成功",
  "data": {
    "user_id": 123,
    "benefit_code": "content_vip",
    "has_permission": false,
    "reason": "非会员用户，无此权益"
  }
}
```

### 校验不通过 - 会员已过期（200）
```json
{
  "code": 0,
  "msg": "校验成功",
  "data": {
    "user_id": 123,
    "benefit_code": "content_vip",
    "has_permission": false,
    "reason": "会员已过期",
    "level_code": "vip_monthly",
    "expire_at": 1775606400,
    "is_permanent": false
  }
}
```

### 校验不通过 - 等级不足（200）
```json
{
  "code": 0,
  "msg": "校验成功",
  "data": {
    "user_id": 123,
    "benefit_code": "high_quality",
    "has_permission": false,
    "reason": "当前会员等级不支持此权益",
    "level_code": "vip_basic",
    "is_permanent": false
  }
}
```

### 校验不通过 - 查询失败（200）
```json
{
  "code": 0,
  "msg": "校验成功",
  "data": {
    "user_id": 123,
    "benefit_code": "content_vip",
    "has_permission": false,
    "reason": "查询会员信息失败"
  }
}
```

---

## 字段说明

### CheckUserBenefitResp 响应字段
| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | int64 | 用户 ID |
| benefit_code | string | 请求校验的权益标识 |
| has_permission | bool | **是否拥有该权益**（true=通过，false=拒绝） |
| reason | string | **校验结果说明**（非会员、等级不足、已过期等） |
| level_code | string | 会员等级代码（可选，校验通过时返回） |
| expire_at | int64 | 会员过期时间戳（可选，Unix 时间戳） |
| is_permanent | bool | 是否永久会员（可选） |
| subscription_active | bool | 订阅是否有效（可选） |

---

## 校验通过条件（满足其一即可）

✅ **条件 1**: 用户是有效会员，且会员等级包含该权益
- 会员在有效期内（或永久会员）
- 会员等级对应的权益列表包含请求的 benefit_code

✅ **条件 2**: 用户拥有单独赠送的该权益且在有效期
- 体验卡、活动赠送等特殊权益
- 权益在有效期内

✅ **条件 3**: 用户在体验期/活动期内享有该权益
- 新用户体验期
- 特殊活动期

---

## 不通过场景

❌ **场景 1**: 用户不是会员
- reason: "非会员用户，无此权益"

❌ **场景 2**: 会员已过期
- reason: "会员已过期"

❌ **场景 3**: 会员被冻结/禁用
- reason: "会员账号异常"

❌ **场景 4**: 当前会员等级不支持该权益
- reason: "当前会员等级不支持此权益"

❌ **场景 5**: 权益使用次数已达上限
- reason: "权益使用次数已达上限"

❌ **场景 6**: 设备数量超出会员允许绑定上限
- reason: "设备数量超出会员限制"

---

## 数据库查询

### 1. 查询用户会员档案
```sql
SELECT level_code, expire_at, is_permanent, status
FROM user_member
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
```

### 2. 查询会员等级权益列表
```sql
SELECT b.benefit_code, b.benefit_name, b.description
FROM member_level_benefit mlb
JOIN member_benefit b ON b.benefit_code = mlb.benefit_code
WHERE mlb.level_code = $1
  AND mlb.status = 1
  AND b.status = 1;
```

### 3. 查询用户特殊权益（可选）
```sql
SELECT benefit_code, expire_at
FROM user_benefit_grant
WHERE user_id = $1
  AND benefit_code = $2
  AND status = 1
  AND (expire_at IS NULL OR expire_at > NOW());
```

---

## 业务联动

### 内容服务
```go
// 播放 VIP 内容前校验
resp, err := benefitClient.CheckUserBenefit(ctx, &types.CheckUserBenefitReq{
    BenefitCode: "content_vip",
})
if err != nil || !resp.HasPermission {
    return nil, errors.New("无权播放 VIP 内容：" + resp.Reason)
}
// 继续播放逻辑
```

### 设备服务
```go
// 绑定设备前校验设备数量限制
if deviceCount >= limit {
    resp, err := benefitClient.CheckUserBenefit(ctx, &types.CheckUserBenefitReq{
        BenefitCode: "unlimited_device",
    })
    if err != nil || !resp.HasPermission {
        return nil, errors.New("设备数量超出限制")
    }
}
```

### 用户服务
```go
// 高音质播放校验
resp, err := l.CheckUserBenefit(ctx, "high_quality")
if !resp.HasPermission {
    return nil, errors.New("非会员无法使用高音质")
}
```

---

## 调用示例

### cURL
```bash
curl -X POST http://localhost:8888/api/v1/user/benefit/check \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "benefit_code": "content_vip"
  }'
```

### JavaScript
```javascript
async checkUserBenefit(benefitCode) {
  const res = await fetch('/api/v1/user/benefit/check', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      benefit_code: benefitCode
    })
  });
  
  const result = await res.json();
  if (result.code === 0) {
    if (result.data.has_permission) {
      console.log('拥有权益', result.data);
      return true;
    } else {
      console.error('无权访问', result.data.reason);
      return false;
    }
  }
  return false;
}

// 使用示例
const hasVip = await checkUserBenefit('content_vip');
if (hasVip) {
  // 播放 VIP 内容
} else {
  // 提示开通会员
}
```

### Go 微服务调用
```go
// 在服务间调用
type BenefitService struct {
    httpClient *http.Client
}

func (s *BenefitService) CheckBenefit(ctx context.Context, userID int64, benefitCode string) (bool, string, error) {
    req := &types.CheckUserBenefitReq{
        BenefitCode: benefitCode,
    }
    
    // 调用用户微服务
    resp, err := s.userClient.CheckUserBenefit(ctx, req)
    if err != nil {
        return false, "系统繁忙", err
    }
    
    return resp.HasPermission, resp.Reason, nil
}
```

---

## 实现文件清单

```
services/user/
├── internal/
│   ├── types/
│   │   └── types.go                          # DTO 定义（已更新）
│   ├── logic/
│   │   └── check_user_benefit_logic.go       # 业务逻辑层（新建）
│   ├── handler/
│   │   └── check_user_benefit_handler.go     # API 处理器（新建）
│   └── handler/
│       └── routes.go                         # 路由注册（已更新）
└── docs/
    └── user_benefit_check_api.md             # API 文档（新建）
```

---

## 注意事项

1. **权限校验**: 必须校验用户身份，防止越权查询
2. **缓存优化**: 可缓存用户权益信息（Redis），减少数据库查询
3. **性能考虑**: 高频调用场景建议使用本地缓存 + 异步刷新
4. **日志审计**: 记录权益校验日志，用于分析和审计
5. **错误处理**: 查询失败时返回友好提示，不暴露系统错误
6. **特殊权益**: 支持单独赠送权益，优先级高于会员等级

---

## 性能优化建议

### 1. 缓存策略
```go
// Redis 缓存用户权益（5 分钟过期）
cacheKey := fmt.Sprintf("user:benefit:%d:%s", userId, benefitCode)
cached, err := redis.Get(ctx, cacheKey).Bool()
if err == nil {
    return cached, nil
}

// 查询数据库后写入缓存
redis.Set(ctx, cacheKey, hasPermission, 5*time.Minute)
```

### 2. 批量查询
```go
// 一次性查询用户所有权益
allBenefits := userRepo.ListAllUserBenefits(ctx, userId)
// 内存判断，避免多次数据库查询
```

### 3. 本地缓存
```go
// 使用 go-zero 缓存组件
cache := cache.NewCache(redisCli)
cache.GetWithLock(key, func() (interface{}, error) {
    // 查询数据库
})
```

---

## 与其他接口的关系

| 接口 | 功能 | 调用时机 |
|------|------|----------|
| `/user/member/info` | 查询会员信息 | 会员中心展示 |
| `/user/member/benefits` | 查询会员权益列表 | 会员中心展示 |
| `/user/benefit/check` | **权益校验** | **业务接口权限控制** |

---

## 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 校验成功（包含 has_permission 字段） | - |
| 1004 | Token 无效或过期 | 重新登录获取新 token |
| 1008 | 参数错误 | 检查 benefit_code 是否正确 |

---

## 监控与统计

### 1. 关键指标
- 权益校验 QPS
- 校验通过率
- 平均响应时间
- 缓存命中率

### 2. 日志记录
```go
l.Logger.Infof("权益校验：user_id=%d, benefit_code=%s, has_permission=%v, reason=%s",
    userId, req.BenefitCode, resp.HasPermission, resp.Reason)
```

### 3. 告警规则
- 校验失败率 > 10% 触发告警
- 响应时间 > 500ms 触发告警
- 缓存失效率 > 50% 触发告警

---

**版本**: v1.0.0  
**更新时间**: 2026-04-08  
**状态**: ✅ 已完成并编译通过
