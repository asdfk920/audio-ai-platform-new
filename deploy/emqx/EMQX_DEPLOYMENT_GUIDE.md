# 🚀 EMQX 5.8.9 远程服务器完整部署指南

## 📋 项目信息

| 项目 | 值 |
|------|-----|
| **目标服务器** | 14.103.202.69:26120 |
| **操作系统** | Amazon Linux 2023 (amzn2023) |
| **EMQX版本** | 5.8.9 |
| **安装包** | emqx-5.8.9-amzn2023-amd64.rpm |
| **后端服务** | Device API @ http://14.103.202.69:8002 |
| **MQTT端口** | 1883 (标准) / 8883 (SSL) / 8083 (WebSocket) |
| **Dashboard** | http://14.103.202.69:18083 |

---

## 🎯 部署总览（6大步骤）

根据你的需求图片，完整流程如下：

```
步骤1: 安装依赖库 ────────────────── ✅ yum install
   ↓
步骤2: 上传并安装 EMQX RPM 包 ─────── ✅ dnf install emqx-*.rpm
   ↓
步骤3: 启动服务 + 开机自启 ───────── ✅ systemctl enable/start emqx
   ↓
步骤4: 开放防火墙/安全组端口 ──────── ✅ firewall-cmd 或 AWS Security Group
   ↓
步骤5: 配置 Dashboard 控制台 ──────── ✅ 修改默认密码
   ↓
步骤6: 配置 HTTP 认证器 ⭐ 核心功能 ── ✅ 对接 /mqtt/auth 接口
```

---

## 📦 步骤1：安装服务器上的依赖库

### 1.1 SSH 登录到服务器

```bash
# Windows PowerShell / CMD
ssh -i your_key.pem ec2-user@14.103.202.69

# 或者使用 root 账户（如果允许）
ssh root@14.103.202.69
```

### 1.2 更新系统包管理器

```bash
# Amazon Linux 2023 使用 dnf (兼容 yum)
sudo dnf update -y

# 如果提示需要升级内核，可以选择性执行:
sudo dnf upgrade -y
```

### 1.3 安装 EMQX 运行所需的依赖

```bash
# 安装基础工具和EMQX依赖
sudo dnf install -y \
    curl \
    wget \
    openssl \
    net-tools \
    lsof \
    unzip \
    jq \
    gnupg2 \
    ca-certificates

# 安装 EMQX 特定运行时库
sudo dnf install -y \
    libgcc \
    libstdc++ \
    libopenssl \
    zlib \
    ncurses-libs \
    readline
```

**验证安装成功**：
```bash
# 检查关键工具是否可用
curl --version
wget --version
openssl version
```

预期输出显示各工具的版本号即可。

---

## 📥 步骤2：上传并安装 EMQX

### 方法A：从本地上传 RPM 文件（推荐）

#### 2.1 在本地 Windows/Mac 执行 SCP 上传

```powershell
# Windows PowerShell 示例
scp D:\emqx-5.8.9-amzn2023-amd64.rpm root@14.103.202.69:/tmp/

# Mac/Linux 终端示例
scp ~/Downloads/emqx-5.8.9-amzn2023-amd64.rpm root@14.103.202.69:/tmp/
```

**如果遇到权限问题**，先确保文件可读：
```powershell
# Windows: 右键文件 → 属性 → 安全 → 确保当前用户有读取权限
```

#### 2.2 在服务器上安装 RPM

```bash
# 切换到root（如果不是）
sudo su -

# 进入tmp目录
cd /tmp

# 确认文件已上传
ls -lh emqx-5.8.9-amzn2023-amd64.rpm

# 执行安装
dnf install -y ./emqx-5.8.9-amzn2023-amd64.rpm
```

**预期输出**：
```
Last metadata expiration check: ...
Dependencies resolved.
============================================================================
 Package          Architecture Version        Repository         Size
============================================================================
Installing:
 emqx             x86_64       5.8.9           @commandline      150 M
...
Complete!
```

---

### 方法B：直接在服务器下载（备选方案）

如果无法SCP上传，可尝试直接下载：

```bash
# 创建临时目录
mkdir -p /tmp/emqx-install && cd /tmp/emqx-install

# 下载 EMQX（可能需要代理或VPN）
wget https://www.emqx.com/en/downloads/broker/v5.8.9/emqx-5.8.9-amzn2023-amd64.rpm \
     --timeout=60 \
     -O emqx.rpm

# 安装
dnf install -y ./emqx.rpm
```

> ⚠️ 注意：某些网络环境下可能无法访问 EMQX 官网，建议使用方法A

---

## 🚀 步骤3：启动 EMQX 并设置开机自启

### 3.1 启用 systemd 服务

```bash
# 设置开机自启动
systemctl enable emqx

# 启动服务
systemctl start emqx
```

### 3.2 检查服务状态

```bash
# 查看运行状态
systemctl status emqx

# 预期输出（部分）:
# ● emqx.service - EMQX Broker
#    Loaded: loaded (/usr/lib/systemd/system/emqx.service; enabled; vendor preset: disabled)
#    Active: active (running) since Mon 2026-05-12 10:00:00 CST; 10s ago
```

**关键检查点**：
- ✅ `Loaded:` 显示 `enabled`（已启用开机自启）
- ✅ `Active:` 显示 `active (running)`（正在运行）

### 3.3 查看启动日志（如果失败）

```bash
# 实时查看最新日志
journalctl -u emqx -f

# 查看最近50条日志
journalctl -u emqx -n 50 --no-pager

# 常见错误及解决:
# ❌ "Address already in use" → 端口被占用，使用 netstat -tlnp 查看
# ❌ "Permission denied" → 权限不足，确保使用 sudo/root
# ❌ "Failed to start" → 查看详细错误信息
```

### 3.4 验证端口监听

```bash
# 检查关键端口是否已监听
netstat -tlnp | grep -E 'emqx|beam'

# 或使用 ss 命令（更现代）
ss -tlnp | grep emqx

# 预期输出示例:
# tcp   0   0 0.0.0.0:1883    0.0.0.0:*   LISTEN  12345/emqx
# tcp   0   0 0.0.0.0:18083   0.0.0.0:*   LISTEN  12345/emqx
# tcp   0   0 0.0.0.0:18084   0.0.0.0:*   LISTEN  12345/emqx
```

**端口说明**：
| 端口 | 用途 | 是否必须开放 |
|------|------|-------------|
| 1883 | MQTT 协议 | ✅ 必须 |
| 8883 | MQTT over SSL | 推荐 |
| 8083 | WebSocket | 可选 |
| 18083 | Dashboard Web控制台 | ✅ 必须 |
| 18084 | Management API | 可选 |

---

## 🔥 步骤4：开放防火墙端口（⭐ 关键步骤）

> ⚠️ **如果不开放端口，设备将无法连接！**

### 4.1 检查系统防火墙状态

```bash
# 检查 firewalld 是否运行
systemctl is-active firewalld

# 输出 "active" 表示防火墙正在运行
# 输出 "unknown" 或 "inactive" 表示未运行
```

### 4.2 开放端口（firewalld 方式）

**如果 firewalld 正在运行**：

```bash
# 开放 MQTT 端口
sudo firewall-cmd --permanent --add-port=1883/tcp

# 开发 MQTTS 端口
sudo firewall-cmd --permanent --add-port=8883/tcp

# 开放 Dashboard 端口
sudo firewall-cmd --permanent --add-port=18083/tcp

# 开放 WebSocket 端口（可选）
sudo firewall-cmd --permanent --add-port=8083/tcp

# 重载使规则生效
sudo firewall-cmd --reload

# 验证规则已添加
sudo firewall-cmd --list-all
```

**预期输出片段**：
```
ports: 1883/tcp 8883/tcp 18083/tcp 8083/tcp
```

### 4.3 开放端口（iptables 方式）

**如果使用 iptables**：

```bash
# 添加规则
sudo iptables -I INPUT -p tcp --dport 1883 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8883 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 18083 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8083 -j ACCEPT

# 保存规则（防止重启丢失）
sudo iptables-save > /etc/iptables.rules

# 验证
sudo iptables -L -n | grep -E '1883|18083'
```

### 4.4 AWS 安全组配置（⭐ 必须同时配置）

即使Linux防火墙已开放，**AWS安全组也必须添加入站规则**！

详见配套文档：[AWS_SECURITY_GROUP_CONFIG.md](./AWS_SECURITY_GROUP_CONFIG.md)

**快速命令行方式**（假设你已配置好AWS CLI）：

```bash
# 替换 sg-xxxxxxxx 为你的实际安全组ID
SECURITY_GROUP_ID="sg-xxxxxxxx"

# 规则1: MQTT (对所有IP开放)
aws ec2 authorize-security-group-ingress \
    --group-id "$SECURITY_GROUP_ID" \
    --protocol tcp --port 1883 \
    --cidr "0.0.0.0/0"

# 规则2: Dashboard (仅限你的IP)
YOUR_IP=$(curl -s ifconfig.me)
aws ec2 authorize-security-group-ingress \
    --group-id "$SECURITY_GROUP_ID" \
    --protocol tcp --port 18083 \
    --cidr "${YOUR_IP}/32"
```

---

## ⚙️ 步骤5：配置 EMQX Dashboard 控制台

### 5.1 访问 Dashboard

打开浏览器访问：
```
http://14.103.202.69:18083
```

**首次登录账号密码**：
- 用户名: `admin`
- 密码: `public`

> ⚠️ **重要**：登录后立即修改密码！否则任何人都能管理你的Broker

### 5.2 修改管理员密码

**方法1：通过Web界面修改**
1. 登录 Dashboard
2. 左侧菜单 → **Settings（设置）**
3. 点击 **Users（用户）**
4. 找到 admin 用户 → 点击编辑图标
5. 输入新密码（如：`Audio_Platform_2026_Secure`）
6. 保存

**方法2：通过API修改（自动化脚本可用）**

```bash
# 使用 curl 调用 EMQX REST API
curl -X PUT "http://127.0.0.1:18084/api/v5/users/admin" \
  -u "admin:public" \
  -H "Content-Type: application/json" \
  -d '{"new_password":"your_secure_password","old_password":"public"}'
```

**验证修改成功**：
```json
{
  "code": 0,
  "message": "Update user successfully!"
}
```

### 5.3 验证 Dashboard 可访问性

修改密码后，重新登录测试：
1. 清除浏览器缓存（Ctrl+Shift+Delete）
2. 重新访问 http://14.103.202.69:18083
3. 使用新密码登录
4. 确认能看到 Overview（概览）页面

---

## 🔐 步骤6：配置 HTTP 认证器（⭐⭐⭐ 核心功能）

这是整个部署的**最关键步骤**，对接设备管理微服务的 `/mqtt/auth` 接口。

### 6.1 准备工作

**前置条件确认**：
- [ ] 设备管理微服务(device-api)已在8002端口运行
- [ ] 数据库 device 表中已有测试设备数据
- [ ] `/mqtt/auth` 接口可通过 Postman/Apifox 测试通过

**快速验证后端接口可用性**：
```bash
# 在服务器上测试（或在本地浏览器测试）
curl -X POST http://14.103.202.69:8002/mqtt/auth \
  -H "Content-Type: application/json" \
  -d '{"clientid":"SN-HP-002","username":"SN-HP-002","password":"$2a$10$mockhash013xxx..."}'
```

**预期返回**：
```json
{"result":"allow"}
```

### 6.2 通过 Dashboard 配置认证器（推荐新手使用）

#### 6.2.1 打开认证配置页面

1. 登录 EMQX Dashboard (http://14.103.202.69:18083)
2. 左侧菜单点击 **Extensions（扩展）**
3. 选择 **Authentication（认证）**
4. 点击 **Create（创建）** 按钮

#### 6.2.2 选择认证类型

在下拉列表中选择：
```
Authentication Type: HTTP Server
```

#### 6.2.3 填写认证参数

按照以下表格填写表单：

| 字段 | 值 | 说明 |
|------|-----|------|
| **Method** | `post` | HTTP请求方法 |
| **URL** | `http://14.103.202.69:8002/mqtt/auth` | 你的设备认证接口地址 |
| **Headers** | `Content-Type: application/json` | 请求头 |
| **Body Template** | 见下方 | 请求体模板 |

**Body Template（请求体模板）**：
```json
{
  "clientid": "${clientid}",
  "username": "${username}",
  "password": "${password}"
}
```

> 💡 **变量说明**：
> - `${clientid}` → EMQX 自动替换为设备的 ClientID（即设备SN）
> - `${username}` → EMQX 自动替换为 Username（即设备SN）
> - `${password}` → EMQX 自动替换为 Password（即设备密钥）

#### 6.2.4 高级配置（可选但推荐）

展开 **Advanced Settings（高级设置）**：

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| Connect Timeout | `5s` | 连接超时时间（防止后端慢响应阻塞）|
| Pool Size | `32` | 连接池大小（支持高并发设备接入）|
| Retry Times | `3` | 失败重试次数 |
| SSL Enable | `false` | 后端是HTTP非HTTPS |

#### 6.2.5 保存并激活

1. 点击右下角 **Create（创建）** 按钮
2. 等待几秒钟让配置生效
3. 页面应显示绿色的 **Running（运行中）** 状态

**✅ 成功标志**：
- 状态显示为 `Connected` 或 `Running`
- 无红色错误提示
- 可以看到请求数量统计（初始为0）

---

### 6.3 通过配置文件方式配置（推荐运维使用）

如果你更喜欢配置文件方式（适合批量部署、版本管理等场景），可以使用我们准备好的配置文件：

#### 6.3.1 上传配置文件

```bash
# 在本地执行（Windows PowerShell）
scp deploy/emqx/emqx_auth_config.conf root@14.103.202.69:/tmp/
```

#### 6.3.2 应用配置

```bash
# 在服务器上执行
sudo cp /tmp/emqx_auth_config.conf /etc/emqx/emqx.conf.d/auth.conf

# 重启 EMQX 使配置生效
sudo systemctl restart emqx

# 等待启动完成
sleep 10

# 检查状态
sudo systemctl status emqx
```

#### 6.3.3 验证配置生效

```bash
# 查看日志确认HTTP认证器加载成功
sudo journalctl -u emqx | grep -i "authentication\|http.*auth"

# 预期输出:
# ... authentication.http started successfully
# ... http auth backend connected to http://14.103.202.69:8002/mqtt/auth
```

---

## 🧪 步骤7：端到端测试（验证全部配置）

### 7.1 测试 MQTT 连接认证

**使用 Mosquitto 客户端测试**（需预先安装）：

```bash
# 安装 mosquitto clients（如未安装）
sudo dnf install -y mosquitto

# 测试认证成功场景（使用数据库中已有的设备SN和密钥）
mosquitto_sub \
  -h 14.103.202.69 \
  -p 1883 \
  -i "SN-HP-002" \
  -u "SN-HP-002" \
  -P "\$2a\$10\$mockhash013xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -t "test/topic" \
  -v

# 预期输出（连接成功）:
# Connected
# （等待消息... Ctrl+C 退出）
```

**测试认证失败场景**（故意使用错误密码）：

```bash
mosquitto_sub \
  -h 14.103.202.69 \
  -p 1883 \
  -i "FAKE-DEVICE" \
  -u "FAKE-DEVICE" \
  -P "wrong_password" \
  -t "test/topic"

# 预期输出（连接被拒绝）:
# Error: Connection Refused: not authorised.
```

### 7.2 使用 Apifox/Postman 测试 HTTP 认证接口

参考之前的截图，发送 POST 请求到：
```
http://14.103.202.69:8002/mqtt/auth
```

**请求体**：
```json
{
  "clientid": "SN-HP-002",
  "username": "SN-HP-002",
  "password": "$2a$10$mockhash013xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

**预期响应**：
```json
{"result":"allow"}
```

### 7.3 检查认证日志

**查询数据库**（在你的开发环境或服务器上）：

```sql
-- 查看最近的认证记录
SELECT 
    id,
    device_sn,
    auth_result,
    error_msg,
    created_at
FROM device_auth_log 
ORDER BY created_at DESC 
LIMIT 10;
```

**预期结果**：应该能看到刚才测试产生的记录（auth_result = success 或 failed）

---

## 🎉 部署完成清单

逐项确认以下所有项目：

### 服务层
- [ ] EMQX 5.8.9 已成功安装
- [ ] EMQX 服务已启动且开机自启 (`systemctl status emqx`)
- [ ] 关键端口已监听 (1883, 18083)

### 网络层
- [ ] Linux 防火墙已开放端口 (firewalld/iptables)
- [ ] AWS 安全组已添加入站规则
- [ ] 本地能 Telnet 通 14.103.202.69:1883 和 :18083

### 配置层
- [ ] Dashboard 密码已修改（非默认 public）
- [ ] HTTP 认证器已配置并运行
- [ ] 认证 URL 指向正确的后端地址

### 功能验证
- [ ] 能打开 Dashboard Web界面
- [ ] MQTT 客户端能成功连接（使用正确凭据）
- [ ] 错误凭据会被拒绝（deny）
- [ ] 数据库 device_auth_log 表有记录生成

---

## 📚 附录：常用运维命令

### EMQX 服务管理
```bash
# 启动/停止/重启
sudo systemctl start emqx
sudo systemctl stop emqx
sudo systemctl restart emqx

# 查看状态
sudo systemctl status emqx

# 查看实时日志
sudo journalctl -u emqx -f

# 查看版本
emqx version
```

### EMQX CLI 工具
```bash
# 查看节点状态
emqx ctl status

# 查看客户端连接数
emqx ctl clients list

# 查看监听器
emqx ctl listeners list
```

### 故障排查
```bash
# 检查端口占用
sudo netstat -tlnp | grep :1883
sudo lsof -i :1883

# 检查进程是否存在
ps aux | grep emqx

# 测试本地连接
mosquitto_pub -h localhost -p 1883 -t test -m hello
```

---

## 🔗 相关文档链接

- **EMQX 配置文件**: [deploy/emqx/emqx_auth_config.conf](./emqx_auth_config.conf)
- **自动部署脚本**: [deploy/emqx/deploy_emqx_server.sh](./deploy_emqx_server.sh)
- **AWS 安全组配置**: [deploy/emqx/AWS_SECURITY_GROUP_CONFIG.md](./AWS_SECURITY_GROUP_CONFIG.md)
- **HTTP 认证接口文档**: [services/device/docs/EMQX_MQTT_AUTH_CONFIG.md](../../services/device/docs/EMQX_MQTT_AUTH_CONFIG.md)

---

## 🆘 获取帮助

如果遇到问题：

1. **查看日志**: `sudo journalctl -u emqx -n 100 --no-pager`
2. **检查端口**: `sudo netstat -tlnp | grep emqx`
3. **官方文档**: https://www.emqx.io/docs/en/latest/
4. **社区支持**: https://github.com/emqx/emqx/issues

---

**📅 文档版本**: v1.0  
**📅 最后更新**: 2026-05-12  
**👤 作者**: Device Service Team  
**🎯 适用场景**: 生产环境部署、测试环境搭建、CI/CD集成

---

## 🏁 下一步行动

完成本部署后，你可以开始：

1. **设备联调测试** - 将真实设备固件配置为连接此 Broker
2. **监控告警配置** - 配置 Prometheus + Grafana 监控 EMQX 指标
3. **集群扩展** - 如需高可用，部署 EMQX 集群模式
4. **SSL/TLS 加密** - 为生产环境启用证书加密传输

祝部署顺利！🚀
