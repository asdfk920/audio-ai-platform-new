# 用户家庭与设备共享接口文档

## 概览

- 服务前缀：`/api/v1/user`
- 认证方式：`Authorization: Bearer <access_token>`
- 目标：支持自动建家、显式建家、家庭成员邀请、设备共享邀请/接受/撤销/退出，以及设备列表合并返回 owner/shared 设备。

## 家庭接口

### 1. 创建家庭

- `POST /api/v1/user/family/create`
- body:

```json
{
  "name": "张三的家庭"
}
```

### 2. 查询当前家庭

- `GET /api/v1/user/family/current`

### 3. 邀请家庭成员

- `POST /api/v1/user/family/member/invite`
- body:

```json
{
  "target_account": "13800138000",
  "role": "member",
  "remark": "邀请加入客厅设备家庭空间"
}
```

### 4. 接受家庭邀请

- `POST /api/v1/user/family/member/accept`

```json
{
  "invite_code": "FAM12AB34CD56"
}
```

### 5. 移除家庭成员

- `POST /api/v1/user/family/member/remove`

```json
{
  "user_id": 10002
}
```

### 6. 更新成员角色

- `POST /api/v1/user/family/member/role/update`

```json
{
  "user_id": 10002,
  "role": "super_admin"
}
```

### 7. 查询家庭成员列表

- `GET /api/v1/user/family/member/list`

## 设备共享接口

### 1. 创建设备共享邀请

- `POST /api/v1/user/device/share/create`
- 规则：
  - owner 首次共享会自动建家。
  - super_admin 仅可对同家庭内已进入共享模式的设备继续分享。
  - member 不能继续分享。

```json
{
  "device_sn": "SN10000001",
  "target_account": "13800138001",
  "share_type": "temporary",
  "permission_level": "partial_control",
  "permission": "{\"allowed_actions\":[\"power\",\"volume\"]}",
  "end_at": 1777777777,
  "remark": "临时开放音量和开关权限"
}
```

### 2. 接受设备共享

- `POST /api/v1/user/device/share/accept`

```json
{
  "invite_code": "SHR12AB34CD56"
}
```

### 3. 撤销设备共享

- `POST /api/v1/user/device/share/revoke`

```json
{
  "share_id": 123
}
```

### 4. 退出设备共享

- `POST /api/v1/user/device/share/quit`

```json
{
  "share_id": 123
}
```

### 5. 查询我发出的共享

- `GET /api/v1/user/device/share/sent`

### 6. 查询我收到的共享

- `GET /api/v1/user/device/share/received`

### 7. 查询共享详情

- `GET /api/v1/user/device/share/detail?share_id=123`

## 设备列表返回扩展

`GET /api/v1/user/device/list` 现在会同时返回 owner 和 shared 设备，并补充以下字段：

- `access_mode`: `owner` / `shared`
- `role`: 家庭角色，`owner` / `super_admin` / `member`
- `permission_level`: `full_control` / `partial_control` / `view_only`
- `permission`: JSON 字符串，承载限时与动作白名单
- `owner_user_id`: 设备主人 ID
- `share_id`: 共享记录 ID（仅 shared 设备）
- `family_id`: 家庭 ID（仅 shared 设备）

示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 2,
    "list": [
      {
        "device_sn": "SN10000001",
        "device_name": "主卧音箱",
        "device_model": "PK_X1",
        "system_version": "1.0.3",
        "bind_time": "2026-04-17 15:00:00",
        "access_mode": "owner",
        "role": "owner",
        "permission_level": "full_control",
        "permission": "{\"all\":true}",
        "owner_user_id": 10001
      },
      {
        "device_sn": "SN10000002",
        "device_name": "客厅音箱",
        "device_model": "PK_X2",
        "system_version": "1.2.0",
        "bind_time": "2026-04-17 15:10:00",
        "access_mode": "shared",
        "role": "member",
        "permission_level": "partial_control",
        "permission": "{\"allowed_actions\":[\"power\"]}",
        "owner_user_id": 10001,
        "share_id": 123,
        "family_id": 10
      }
    ]
  }
}
```

## 后台清理与联动

- owner 解绑设备时，会自动把该设备全部共享关系标记为 `revoked`。
- `user` 服务内置共享过期 worker，默认每 5 分钟扫描一次 `end_at < now` 的共享并自动置为 `expired`。
