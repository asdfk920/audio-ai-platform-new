# go-admin 启动指南

## 当前状态

✅ go-admin 已成功集成到项目中
✅ 配置文件已修改为使用 PostgreSQL
❌ 需要启动 Docker 和数据库

## 启动步骤

### 1. 启动 Docker Desktop

请先启动 Docker Desktop 应用

### 2. 启动数据库

```bash
cd /Users/jacklau/Documents/Programs/Go/audio-ai-platform
make docker-up
```

或者：

```bash
docker-compose up -d
```

### 3. 启动 go-admin

```bash
cd admin
go run main.go server -c config/settings.yml
```

### 4. 访问后台

- 后台地址: http://localhost:8000
- 默认账号: admin
- 默认密码: admin123

## go-admin 功能特性

### 内置功能模块

1. **用户管理**
   - 用户列表
   - 用户新增/编辑/删除
   - 用户角色分配
   - 用户状态管理

2. **角色管理**
   - 角色列表
   - 角色权限配置
   - 菜单权限分配
   - 数据权限配置

3. **菜单管理**
   - 菜单树形结构
   - 菜单新增/编辑/删除
   - 菜单图标配置
   - 菜单排序

4. **部门管理**
   - 部门树形结构
   - 部门新增/编辑/删除
   - 部门负责人配置

5. **岗位管理**
   - 岗位列表
   - 岗位新增/编辑/删除
   - 岗位状态管理

6. **字典管理**
   - 字典类型管理
   - 字典数据管理
   - 字典缓存刷新

7. **参数配置**
   - 系统参数配置
   - 参数新增/编辑/删除

8. **通知公告**
   - 公告列表
   - 公告发布
   - 公告状态管理

9. **操作日志**
   - 操作日志查询
   - 日志详情查看
   - 日志导出

10. **登录日志**
    - 登录日志查询
    - 登录统计
    - 异常登录监控

11. **在线用户**
    - 在线用户列表
    - 强制下线
    - 会话管理

12. **定时任务**
    - 任务列表
    - 任务新增/编辑/删除
    - 任务执行日志

13. **代码生成**
    - 数据库表导入
    - 代码自动生成
    - 前后端代码生成

14. **系统接口**
    - API 接口管理
    - 接口文档
    - 接口测试

15. **服务监控**
    - 服务器信息
    - CPU 使用率
    - 内存使用率
    - 磁盘使用率

## 配置说明

### 数据库配置

已配置为使用项目的 PostgreSQL 数据库：

```yaml
database:
  driver: postgres
  source: host=localhost port=5432 user=admin password=admin123 dbname=audio_platform sslmode=disable TimeZone=Asia/Shanghai
```

### JWT 配置

```yaml
jwt:
  secret: go-admin
  timeout: 3600  # 1小时
```

### 日志配置

```yaml
logger:
  path: temp/logs
  level: trace
  enableddb: false
```

## 自定义业务模块

### 添加设备管理模块

1. **创建数据模型**

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

2. **使用代码生成器**

```bash
cd admin
go run main.go gen -t devices -m Device
```

这会自动生成：
- API 接口
- Service 层
- DTO 对象
- 路由配置
- 前端页面（如果配置了前端路径）

3. **访问新模块**

启动后在菜单管理中添加设备管理菜单，即可访问。

## API 接口

### 登录接口

```bash
POST http://localhost:8000/api/v1/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

### 获取用户列表

```bash
GET http://localhost:8000/api/v1/sys-user?pageSize=10&pageIndex=1
Authorization: Bearer {token}
```

## 前端集成（可选）

如果需要前端界面，可以克隆 go-admin-ui：

```bash
cd /Users/jacklau/Documents/Programs/Go/audio-ai-platform
git clone https://github.com/go-admin-team/go-admin-ui.git admin-ui
cd admin-ui
npm install
npm run dev
```

前端会运行在 http://localhost:9527

## 常见问题

### 1. 数据库连接失败

确保 PostgreSQL 已启动：
```bash
docker ps | grep postgres
```

### 2. 端口被占用

修改 config/settings.yml 中的端口号

### 3. 日志目录不存在

```bash
mkdir -p admin/temp/logs
```

## 下一步

1. 启动 Docker Desktop
2. 运行 `make docker-up` 启动数据库
3. 运行 `cd admin && go run main.go server -c config/settings.yml`
4. 访问 http://localhost:8000
5. 使用 admin/admin123 登录

## 参考资料

- 官方文档: https://doc.go-admin.pro
- GitHub: https://github.com/go-admin-team/go-admin
- 视频教程: https://space.bilibili.com/565616721
