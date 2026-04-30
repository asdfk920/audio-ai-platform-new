# go-admin 前端启动指南

## 当前状态

✅ 后端服务已启动：http://localhost:8000
✅ 前端项目已克隆：admin-ui/
⏳ npm install 正在后台运行中...

## 前端启动步骤

### 1. 等待 npm install 完成

npm install 正在后台运行，可能需要 5-10 分钟，取决于网络速度。

检查是否完成：
```bash
ls admin-ui/node_modules/
```

如果看到很多目录，说明安装完成。

### 2. 配置后端 API 地址

前端默认配置已经指向 http://localhost:8000，无需修改。

查看配置：
```bash
cat admin-ui/.env.development
```

应该显示：
```
VUE_APP_BASE_API = 'http://localhost:8000'
```

### 3. 启动前端开发服务器

```bash
cd admin-ui
npm run dev
```

### 4. 访问前端

前端会运行在：http://localhost:9527

- 地址：http://localhost:9527
- 账号：admin
- 密码：admin123

## 前端功能

go-admin-ui 是基于 Vue 2 + Element UI 的前端框架，提供：

### 核心功能
- 🎨 现代化的 UI 界面
- 📊 数据可视化大屏
- 📝 表单设计器
- 🔐 完整的权限控制
- 📱 响应式布局

### 业务模块
- 用户管理
- 角色管理
- 菜单管理
- 部门管理
- 岗位管理
- 字典管理
- 参数配置
- 通知公告
- 操作日志
- 登录日志
- 在线用户
- 定时任务
- 代码生成
- 系统接口
- 服务监控

## 手动安装（如果后台安装失败）

如果 npm install 失败或太慢，可以手动执行：

```bash
# 停止后台任务
ps aux | grep "npm install" | grep -v grep | awk '{print $2}' | xargs kill

# 使用国内镜像
cd admin-ui
npm config set registry https://registry.npmmirror.com
npm install

# 或使用 yarn
npm install -g yarn
yarn config set registry https://registry.npmmirror.com
yarn install
```

## 启动命令

```bash
# 开发模式
npm run dev

# 构建生产版本
npm run build:prod

# 预览生产版本
npm run preview

# 代码检查
npm run lint
```

## 常见问题

### 1. npm install 太慢

使用国内镜像：
```bash
npm config set registry https://registry.npmmirror.com
```

### 2. 端口被占用

修改 `vue.config.js` 中的端口：
```javascript
devServer: {
  port: 9527, // 改成其他端口
}
```

### 3. 无法连接后端

检查后端是否运行：
```bash
curl http://localhost:8000/api/v1/health
```

### 4. 登录失败

确保：
- 后端服务正常运行
- 数据库已初始化
- 使用正确的账号密码（admin/admin123）

## 项目结构

```
admin-ui/
├── public/              # 静态资源
├── src/
│   ├── api/            # API 接口
│   ├── assets/         # 资源文件
│   ├── components/     # 组件
│   ├── layout/         # 布局
│   ├── router/         # 路由
│   ├── store/          # Vuex 状态管理
│   ├── styles/         # 样式
│   ├── utils/          # 工具函数
│   ├── views/          # 页面
│   ├── App.vue
│   └── main.js
├── .env.development    # 开发环境配置
├── .env.production     # 生产环境配置
├── package.json
└── vue.config.js       # Vue 配置
```

## 开发建议

### 1. 使用代码生成器

在后台管理中使用代码生成器，可以自动生成：
- 后端 API 代码
- 前端页面代码
- 路由配置
- API 接口

### 2. 自定义主题

修改 `src/styles/variables.scss` 自定义主题颜色。

### 3. 添加新页面

1. 在 `src/views/` 创建页面组件
2. 在 `src/router/` 添加路由
3. 在后台管理中添加菜单

## 下一步

1. 等待 npm install 完成（可能需要几分钟）
2. 运行 `cd admin-ui && npm run dev`
3. 访问 http://localhost:9527
4. 使用 admin/admin123 登录
5. 开始使用完整的前后端分离系统

## 参考资料

- 官方文档：https://doc.go-admin.pro
- GitHub：https://github.com/go-admin-team/go-admin-ui
- Element UI：https://element.eleme.cn
- Vue 2：https://v2.cn.vuejs.org
