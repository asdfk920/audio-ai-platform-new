# 平台用户列表 API 文档

## 接口说明

实现管理后台用户列表查询功能，支持分页、多条件筛选、会员信息查询、设备绑定统计。

## 接口地址

```
GET /api/v1/platform-user/list
```

## 请求参数

### Header

```
Authorization: Bearer <admin_access_token>
```

### Query 参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| page | int | 否 | 页码，默认 1 | 1 |
| pageSize | int | 否 | 每页条数，默认 10，最大 100 | 10 |
| mobile | string | 否 | 手机号（模糊搜索） | 13800138000 |
| nickname | string | 否 | 昵称（模糊搜索） | 张三 |
| email | string | 否 | 邮箱（模糊搜索） | zhangsan@example.com |
| status | int | 否 | 账号状态 0-禁用 1-正常 | 1 |
| memberLevel | int | 否 | 会员等级 | 2 |
| realNameStatus | int | 否 | 实名状态 | 1 |
| registerTimeStart | string | 否 | 注册时间开始（Unix 时间戳） | 1672531200 |
| registerTimeEnd | string | 否 | 注册时间结束（Unix 时间戳） | 1704067199 |

### 请求示例

```bash
curl -X GET 'http://localhost:8000/api/v1/platform-user/list?page=1&pageSize=10&status=1&memberLevel=2' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
```

## 成功响应

### 响应数据结构

```json
{
  "code": 200,
  "msg": "查询成功",
  "data": {
    "list": [
      {
        "userId": 1001,
        "mobile": "13800138000",
        "email": "zhangsan@example.com",
        "nickname": "张三",
        "avatar": "https://example.com/avatar/1001.jpg",
        "memberLevel": 2,
        "memberLevelName": "SVIP 会员",
        "memberExpireAt": 1735689600,
        "status": 1,
        "realNameStatus": 1,
        "bindDeviceCount": 3,
        "registerTime": 1672531200,
        "lastLoginTime": 1704067200,
        "createdAt": "2023-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 10
  }
}
```

### 字段说明

#### 用户列表项（list[]）

| 字段名 | 类型 | 说明 |
|--------|------|------|
| userId | int64 | 用户 ID |
| mobile | string | 手机号 |
| email | string | 邮箱 |
| nickname | string | 昵称 |
| avatar | string | 头像 URL |
| memberLevel | int32 | 会员等级（0-普通 1-VIP 2-SVIP 3-终身） |
| memberLevelName | string | 会员等级名称 |
| memberExpireAt | int64 | 会员过期时间戳（0 表示永久） |
| status | int32 | 账号状态（0-禁用 1-正常） |
| realNameStatus | int32 | 实名状态 |
| bindDeviceCount | int64 | 绑定设备数量 |
| registerTime | int64 | 注册时间戳 |
| lastLoginTime | int64 | 最后登录时间戳 |
| createdAt | datetime | 创建时间 |
| updatedAt | datetime | 更新时间 |

#### 分页信息

| 字段名 | 类型 | 说明 |
|--------|------|------|
| total | int64 | 总条数 |
| page | int | 当前页码 |
| pageSize | int | 每页条数 |

## 失败响应

### 401 未授权

```json
{
  "code": 401,
  "msg": "未授权，请先登录",
  "data": null
}
```

### 403 权限不足

```json
{
  "code": 403,
  "msg": "权限不足，无法访问用户列表",
  "data": null
}
```

### 400 参数错误

```json
{
  "code": 400,
  "msg": "参数解析失败：page 必须为正整数",
  "data": null
}
```

### 500 服务器内部错误

```json
{
  "code": 500,
  "msg": "查询失败：数据库连接异常",
  "data": null
}
```

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 参数错误 |
| 401 | 未授权，请先登录 |
| 403 | 权限不足 |
| 500 | 服务器内部错误 |

## 业务规则

### 1. 权限控制

- 仅管理员可访问
- 需要有效的 JWT Token
- 需要用户列表查看权限

### 2. 分页限制

- 默认每页 10 条
- 最大每页 100 条
- 禁止全表查询

### 3. 查询规则

- 支持模糊搜索：手机号、昵称、邮箱
- 支持精确筛选：状态、会员等级、实名状态
- 支持时间范围：注册时间段
- 多个条件可组合使用

### 4. 数据脱敏

- 不返回密码、密钥等敏感信息
- 手机号可根据需要脱敏（如：138****8000）
- 邮箱可根据需要脱敏（如：zhang***@example.com）

### 5. 性能优化

- 必须分页查询
- 仅查询必要字段
- 关联查询设备绑定数（避免 N+1 查询）
- 高频查询可加缓存

## 使用示例

### JavaScript (Axios)

```javascript
const axios = require('axios');

async function getUserList(page = 1, pageSize = 10, filters = {}) {
  try {
    const response = await axios.get('http://localhost:8000/api/v1/platform-user/list', {
      params: {
        page,
        pageSize,
        ...filters
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });
    
    if (response.data.code === 200) {
      return response.data.data;
    } else {
      throw new Error(response.data.msg);
    }
  } catch (error) {
    console.error('查询用户列表失败:', error);
    throw error;
  }
}

// 使用示例
getUserList(1, 10, { status: 1, memberLevel: 2 })
  .then(data => {
    console.log('用户列表:', data.list);
    console.log('总数:', data.total);
  })
  .catch(err => {
    console.error('错误:', err.message);
  });
```

### Vue 3

```vue
<template>
  <div class="user-list">
    <el-table :data="userList" v-loading="loading">
      <el-table-column prop="userId" label="用户 ID" width="80" />
      <el-table-column prop="mobile" label="手机号" />
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="memberLevelName" label="会员等级" />
      <el-table-column prop="bindDeviceCount" label="绑定设备数" />
      <el-table-column prop="status" label="状态">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '正常' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>
    
    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { getUserListApi } from '@/api/user';

const loading = ref(false);
const userList = ref([]);
const page = ref(1);
const pageSize = ref(10);
const total = ref(0);

const fetchUserList = async () => {
  loading.value = true;
  try {
    const data = await getUserListApi({
      page: page.value,
      pageSize: pageSize.value,
      status: 1
    });
    userList.value = data.list;
    total.value = data.total;
  } catch (error) {
    console.error('查询失败:', error);
  } finally {
    loading.value = false;
  }
};

const handlePageChange = (newPage) => {
  page.value = newPage;
  fetchUserList();
};

onMounted(() => {
  fetchUserList();
});
</script>
```

## 注意事项

1. **权限验证**：确保管理员已登录且拥有用户列表查看权限
2. **参数校验**：分页参数必须在合法范围内
3. **性能考虑**：避免复杂查询条件，必要时添加数据库索引
4. **数据安全**：不返回敏感字段，做好数据脱敏
5. **错误处理**：前端需友好展示错误信息

## 相关接口

- `GET /api/v1/platform-user/:userId` - 获取用户详情
- `POST /api/v1/platform-user` - 创建用户
- `PUT /api/v1/platform-user/:userId` - 更新用户
- `DELETE /api/v1/platform-user/:userId` - 删除用户
