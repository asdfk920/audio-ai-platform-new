# go-admin 集成指南

## 方案说明

go-admin 是一个完整的后台管理系统框架，包含：
- 用户管理
- 角色权限管理
- 菜单管理
- 部门管理
- 岗位管理
- 字典管理
- 操作日志
- 登录日志
- API 接口管理

## 与微服务集成

### 平台用户管理

已实现 go-admin 平台用户管理：**列表与详情**直连业务 PostgreSQL（与实名审核相同数据源，返回脱敏字段）；**创建用户**仍通过 HTTP 调用用户微服务注册接口。

**关键配置：**
- 用户微服务地址：`http://localhost:8001`（仅创建用户等写操作）
- 管理端数据库 `settings.database.source` 须指向与用户服务相同的业务库
- API 路径：`/api/v1/platform-user-test/list`（测试接口，无需认证）
- API 路径：`/api/v1/platform-user/list`（正式接口，需要认证）

**实现文件：**
- [admin/app/admin/apis/platform_user.go](../admin/app/admin/apis/platform_user.go) - API 处理器
- [admin/app/admin/router/platform_user.go](../admin/app/admin/router/platform_user.go) - 路由配置

**测试接口：**
```bash
# 获取用户列表（无需认证）
curl http://localhost:8000/api/v1/platform-user-test/list

# 返回示例（联系方式为脱敏字段）
{
  "code": 200,
  "count": 3,
  "data": [
    {
      "user_id": 1,
      "email_masked": "u***1@e***.com",
      "mobile_masked": "138****01",
      "nickname": "测试用户1",
      "status": 1,
      "real_name_status": 0
    }
  ],
  "msg": "查询成功"
}
```

## 集成方式

### 方式一：使用 go-admin 完整项目（推荐）

```bash
# 1. 克隆 go-admin 项目到 admin 目录
cd /Users/jacklau/Documents/Programs/Go/audio-ai-platform
rm -rf admin
git clone https://github.com/go-admin-team/go-admin.git admin

# 2. 进入 admin 目录
cd admin

# 3. 安装依赖
go mod tidy

# 4. 配置数据库
# 编辑 config/settings.yml
# 修改数据库连接信息为我们的 PostgreSQL

# 5. 初始化数据库
# go-admin 会自动创建所需的表

# 6. 启动服务
go run main.go server -c config/settings.yml

# 7. 访问后台
# 前端: http://localhost:8000
# 后端 API: http://localhost:8000/api
# 默认账号: admin / admin123
```

### 方式二：使用 go-admin 前后端分离

**后端：**
```bash
# 克隆后端
git clone https://github.com/go-admin-team/go-admin.git admin-backend
cd admin-backend
go mod tidy
go run main.go server -c config/settings.yml
```

**前端：**
```bash
# 克隆前端
git clone https://github.com/go-admin-team/go-admin-ui.git admin-frontend
cd admin-frontend
npm install
npm run dev
```

## 配置文件示例

### config/settings.yml

```yaml
settings:
  application:
    # 应用名称
    name: audio-ai-platform-admin
    # 运行模式：dev/test/prod
    mode: dev
    # 主机
    host: 0.0.0.0
    # 端口
    port: 8000
    # 是否启用 https
    ishttps: false
    # 读取超时时间
    readtimeout: 60
    # 写入超时时间
    writertimeout: 60

  # 数据库配置
  database:
    # 数据库类型：mysql/postgres/sqlite3
    driver: postgres
    # 数据库连接
    source: host=localhost port=5432 user=admin password=admin123 dbname=audio_platform sslmode=disable TimeZone=Asia/Shanghai

  # Redis 配置
  redis:
    # Redis 地址
    addr: localhost:6379
    # Redis 密码
    password: redis123
    # Redis 数据库
    db: 0

  # JWT 配置
  jwt:
    # JWT 密钥
    secret: audio-ai-platform-secret-key
    # JWT 超时时间（秒）
    timeout: 3600

  # 日志配置
  log:
    # 日志级别：debug/info/warn/error
    level: info
    # 日志路径
    path: logs/admin.log
```

## 数据库迁移

go-admin 使用 GORM 自动迁移，会自动创建以下表：

- sys_user - 用户表
- sys_role - 角色表
- sys_menu - 菜单表
- sys_dept - 部门表
- sys_post - 岗位表
- sys_dict_type - 字典类型表
- sys_dict_data - 字典数据表
- sys_config - 配置表
- sys_login_log - 登录日志表
- sys_oper_log - 操作日志表
- sys_api - API 表

## 自定义业务模块

### 1. 创建业务模型

```go
// admin/app/audio/models/device.go
package models

import "go-admin/common/models"

type Device struct {
    models.Model
    DeviceSn        string `json:"device_sn" gorm:"size:100;comment:设备序列号"`
    Model           string `json:"dao" gorm:"size:50;comment:设备型号"`
    FirmwareVersion string `json:"firmware_version" gorm:"size:50;comment:固件版本"`
    Status          string `json:"status" gorm:"size:20;comment:状态"`
    models.ControlBy
    models.ModelTime
}

func (Device) TableName() string {
    return "devices"
}
```

### 2. 创建 API 接口

```go
// admin/app/audio/apis/device.go
package apis

import (
    "github.com/gin-gonic/gin"
    "go-admin/app/audio/models"
    "go-admin/app/audio/service"
    "go-admin/common/apis"
)

type Device struct {
    apis.Api
}

// GetPage 获取设备列表
func (e Device) GetPage(c *gin.Context) {
    s := service.Device{}
    req := dto.DeviceGetPageReq{}
    err := e.MakeContext(c).
        MakeOrm().
        Bind(&req).
        MakeService(&s.Service).
        Errors
    if err != nil {
        e.Error(500, err, err.Error())
        return
    }

    list := make([]models.Device, 0)
    var count int64

    err = s.GetPage(&req, &list, &count)
    if err != nil {
        e.Error(500, err, "查询失败")
        return
    }

    e.PageOK(list, int(count), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}
```

### 3. 注册路由

```go
// admin/app/audio/router/device.go
package router

import (
    "github.com/gin-gonic/gin"
    "go-admin/app/audio/apis"
    "go-admin/common/middleware"
)

func init() {
    routerCheckRole = append(routerCheckRole, registerDeviceRouter)
}

func registerDeviceRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
    api := apis.Device{}
    r := v1.Group("/device").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole())
    {
        r.GET("", api.GetPage)
        r.GET("/:id", api.Get)
        r.POST("", api.Insert)
        r.PUT("/:id", api.Update)
        r.DELETE("", api.Delete)
    }
}
```

## 前端页面

go-admin 前端基于 Vue3 + Element Plus，可以快速生成 CRUD 页面。

### 使用代码生成器

```bash
# 在 go-admin 后端项目中
go run main.go gen -t device -m Device

# 会自动生成：
# - 后端 API
# - 前端页面
# - 路由配置
```

## 启动步骤

1. **启动数据库**
```bash
make docker-up
```

2. **配置 go-admin**
```bash
cd admin
# 编辑 config/settings.yml
# 修改数据库连接为 PostgreSQL
```

3. **初始化数据库**
```bash
cd admin
go run main.go migrate -c config/settings.yml
```

4. **启动后端**
```bash
cd admin
go run main.go server -c config/settings.yml
```

5. **启动前端（可选）**
```bash
cd admin-ui
npm install
npm run dev
```

6. **访问系统**
- 后台地址: http://localhost:8000
- 默认账号: admin
- 默认密码: admin123

## 集成到现有项目

将 go-admin 作为独立的管理后台服务，与我们的微服务并行运行：

```
audio-ai-platform/
├── services/          # 业务微服务
│   ├── user/
│   ├── device/
│   └── content/
├── admin/            # go-admin 管理后台
│   ├── app/
│   ├── config/
│   └── main.go
└── admin-ui/         # go-admin 前端（可选）
```

## 下一步

1. 克隆 go-admin 项目到 admin 目录
2. 配置数据库连接
3. 运行数据库迁移
4. 启动服务
5. 访问后台管理系统

## 参考文档

- go-admin 官网: https://www.go-admin.pro
- GitHub: https://github.com/go-admin-team/go-admin
- 文档: https://doc.go-admin.pro
