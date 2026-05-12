# ============================================================
# AWS 安全组配置指南 - EMQX MQTT Broker
# 服务器: 14.103.202.69 (Amazon Linux 2023)
# 用途: 开放EMQX所需端口，允许设备连接和Dashboard访问
#
# 适用场景:
#   - EC2实例安全组配置
#   - VPC安全组规则添加
#   - 防火墙端口开放验证
#
# 作者: Device Service Team
# 日期: 2026-05-12
# 版本: v1.0
# ============================================================

## 📋 一、需要开放的端口清单

| 端口 | 协议 | 用途 | 来源IP | 优先级 |
|------|------|------|--------|--------|
| **1883** | TCP | MQTT 标准协议（设备连接） | 0.0.0.0/0 (所有IP) | ⭐⭐⭐ 必需 |
| **8883** | TCP | MQTT over SSL/TLS (加密连接) | 0.0.0.0/0 (所有IP) | ⭐⭐ 推荐 |
| **8083** | TCP | MQTT WebSocket (Web客户端) | 0.0.0.0/0 (所有IP) | ⭐ 可选 |
| **18083** | TCP | EMQX Dashboard 控制台 | 你的IP / VPN | ⭐⭐⭐ 必需 |
| **18084** | TCP | EMQX Management API | 127.0.0.1 或 内网 | ⭐ 可选 |

---

## 🔧 二、AWS Console 配置方法（推荐）

### 步骤1：登录 AWS Console

1. 打开浏览器访问: https://console.aws.amazon.com/ec2/
2. 选择正确的区域（你的EC2所在区域）
3. 左侧菜单 → Network & Security → Security Groups

### 步骤2：找到目标安全组

1. 在安全组列表中找到绑定到 `14.103.202.69` 的安全组
2. 点击该安全组名称进入详情页
3. 切换到 **Inbound rules（入站规则）** 标签页

### 步骤3：添加入站规则

#### 规则1：MQTT 协议（1883端口）⭐ 最重要

点击 **Edit inbound rules** → **Add rule**：

```
Type: Custom TCP Rule
Port Range: 1883
Source: Anywhere-IPv4 (0.0.0.0/0)
Description: MQTT Protocol for IoT Devices
```

**说明**：
- 允许所有物联网设备通过标准MQTT协议连接
- 这是设备接入的主要通道
- 如果只允许特定设备网段，可将 Source 改为具体CIDR（如 `10.0.0.0/16`）

---

#### 规则2：MQTT over TLS（8883端口）

```
Type: Custom TCP Rule
Port Range: 8883
Source: Anywhere-IPv4 (0.0.0.0/0)
Description: MQTT over SSL/TLS (Encrypted)
```

**说明**：
- 用于需要加密传输的场景
- 需要预先配置SSL证书（后续步骤）

---

#### 规则3：Dashboard 控制台（18083端口）

```
Type: Custom TCP Rule
Port Range: 18083
Source: My IP (自动填充当前公网IP) 
        或者: 你的固定IP/32
Description: EMQX Dashboard Admin Panel
```

**⚠️ 重要安全提示**：
- ❌ **不要**设置为 `0.0.0.0/0`（任何人都能访问管理后台）
- ✅ 建议限制为你的办公网络IP或VPN地址
- ✅ 生产环境建议使用 VPN + 白名单访问

---

#### 规则4：WebSocket（8083端口，可选）

```
Type: Custom TCP Rule
Port Range: 8083
Source: Anywhere-IPv4 (0.0.0.0/0) 或 前端域名IP
Description: MQTT WebSocket for Web Clients
```

**说明**：
- 用于Web端MQTT调试工具连接
- 如不需要Web客户端可跳过此规则

---

### 步骤4：保存规则

点击右下角 **Save rules** 按钮
等待规则生效（通常几秒钟）

---

## 💻 三、AWS CLI 快速配置（自动化脚本）

如果你使用 AWS CLI，可以一键执行：

```bash
#!/bin/bash
# ============================================================
# AWS CLI 批量添加安全组规则脚本
# 使用方式: chmod +x add_security_rules.sh && ./add_security_rules.sh
# ============================================================

# ====== 配置变量 ======
SECURITY_GROUP_ID="sg-xxxxxxxx"  # 替换为你的安全组ID
YOUR_IP=$(curl -s https://api.ipify.org)  # 自动获取当前公网IP

echo "🔧 开始配置 AWS 安全组..."
echo "📍 目标安全组: ${SECURITY_GROUP_ID}"
echo "🌐 当前公网IP: ${YOUR_IP}"
echo ""

# 规则1: MQTT (1883)
aws ec2 authorize-security-group-ingress \
    --group-id "${SECURITY_GROUP_ID}" \
    --protocol tcp \
    --port 1883 \
    --cidr "0.0.0.0/0" \
    --description "MQTT Protocol for IoT Devices"

echo "✅ 已添加: TCP 1883 (MQTT)"

# 规则2: MQTT SSL (8883)
aws ec2 authorize-security-group-ingress \
    --group-id "${SECURITY_GROUP_ID}" \
    --protocol tcp \
    --port 8883 \
    --cidr "0.0.0.0/0" \
    --description "MQTT over SSL/TLS"

echo "✅ 已添加: TCP 8883 (MQTTS)"

# 规则3: Dashboard (18083) - 仅限当前IP
aws ec2 authorize-security-group-ingress \
    --group-id "${SECURITY_GROUP_ID}" \
    --protocol tcp \
    --port 18083 \
    --cidr "${YOUR_IP}/32" \
    --description "EMQX Dashboard - Admin Only"

echo "✅ 已添加: TCP 18083 (Dashboard) - 仅限 ${YOUR_IP}"

# 规则4: WebSocket (8083, 可选)
read -p "是否开放 WebSocket 端口 8083? (y/n): " ws_choice
if [ "$ws_choice" = "y" ]; then
    aws ec2 authorize-security-group-ingress \
        --group-id "${SECURITY_GROUP_ID}" \
        --protocol tcp \
        --port 8083 \
        --cidr "0.0.0.0/0" \
        --description "MQTT WebSocket"
    
    echo "✅ 已添加: TCP 8083 (WebSocket)"
fi

echo ""
echo "=========================================="
echo "🎉 所有安全组规则已添加完成！"
echo "=========================================="
echo ""
echo "📋 下一步:"
echo "   1. 访问 Dashboard: http://14.103.202.69:18083"
echo "   2. 测试 MQTT 连接: mqtt://14.103.202.69:1883"
echo ""
```

---

## 🔒 四、安全加固建议（生产环境必读）

### 4.1 最小权限原则

❌ **不推荐的配置**：
```json
{
  "PortRange": "18083",
  "Source": "0.0.0.0/0"  // 危险！任何人都可访问管理后台
}
```

✅ **推荐的配置**：
```json
{
  "PortRange": "18083",
  "Source": "203.0.113.50/32"  // 仅限管理员IP
}
```

### 4.2 网络ACL额外防护（可选但推荐）

如果安全级别要求极高，可同时配置 NACL：

```bash
# 创建网络ACL规则（仅允许特定IP段访问MQTT）
aws ec2 create-network-acl-entry \
    --network-acl-id acl-xxxxxxxx \
    --rule-number 100 \
    --protocol tcp \
    --port-range From=1883,To=1883 \
    --allow \
    --cidr-block "10.0.0.0/16"  // 仅允许内网VPC
```

### 4.3 定期审计规则

```bash
# 查看当前安全组规则
aws ec2 describe-security-groups \
    --group-ids "${SECURITY_GROUP_ID}" \
    --query 'SecurityGroups[0].IpPermissions[]' \
    --output table

# 输出示例:
# -------------------------------------------------------
# |                     DescribeSecurityGroups          |
#+-------------------+-----------+----------+-------------+
# |  FromPort         |  IpProtocol|  IpRanges|  ToPort     |
+-------------------+-----------+----------+-------------+
# |  1883            |  tcp      |  0.0.0.0/0 |  1883      |
# |  8883            |  tcp      |  0.0.0.0/0 |  8883      |
# |  18083           |  tcp      |  203.x.x.x/32 | 18083  |
+-------------------+-----------+----------+-------------+
```

---

## 🛡️ 五、防火墙层配置（OS层面）

除了AWS安全组，还需确保Linux防火墙也放行：

### 方法1：firewalld（Amazon Linux 2023默认）

```bash
# 检查firewalld状态
sudo systemctl status firewalld

# 如果正在运行，添加规则
sudo firewall-cmd --permanent --add-port=1883/tcp      # MQTT
sudo firewall-cmd --permanent --add-port=8883/tcp      # MQTTS
sudo firewall-cmd --permanent --add-port=18083/tcp     # Dashboard
sudo firewall-cmd --permanent --add-port=8083/tcp      # WebSocket

# 重载使生效
sudo firewall-cmd --reload

# 验证规则
sudo firewall-cmd --list-all
```

### 方法2：iptables（传统方式）

```bash
# 添加规则
sudo iptables -I INPUT -p tcp --dport 1883 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8883 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 18083 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8083 -j ACCEPT

# 保存规则（重启后仍然有效）
sudo service iptables save
# 或 (Amazon Linux 2023):
sudo iptables-save > /etc/iptables.rules
```

---

## 🧪 六、连通性测试

### 6.1 从本地测试端口可达性

```powershell
# Windows PowerShell 测试
Test-NetConnection -ComputerName 14.103.202.69 -Port 1883
Test-NetConnection -ComputerName 14.103.202.69 -Port 18083

# Linux/Mac 测试
nc -zv 14.103.202.69 1883
nc -zv 14.103.202.69 18083

# Telnet 测试（通用）
telnet 14.103.202.69 1883
telnet 14.103.202.69 18083
```

**预期输出**：
```
TcpTestSucceeded : True
# 或
Connection to 14.103.202.69 1883 port [tcp/*] succeeded!
```

### 6.2 在线端口检测工具

使用以下网站快速检测：
- https://www.yougetsignal.com/tools/open-ports/
- https://portchecker.co/
- https://www.portcheckers.com/

输入：`14.103.202.69` + 端口号（如1883）

---

## ⚠️ 七、常见问题排查

### Q1: 设备无法连接，提示超时

**排查步骤**：
1. ✅ 检查AWS安全组是否有TCP 1883规则
2. ✅ 检查EC2实例是否在公有子网且有公网IP
3. ✅ 检查本地防火墙/公司网络是否阻止出站连接
4. ✅ 使用 `telnet 14.103.202.69 1883` 测试基础连通性

### Q2: Dashboard 无法打开（403 Forbidden）

**原因**：安全组未开放18083或来源IP被限制  
**解决**：
1. 检查安全组规则中18083的Source是否包含当前IP
2. 如果使用了VPN，确认VPN出口IP已加入白名单
3. 尝试更换网络环境（如手机热点）测试

### Q3: 可以Telnet通但MQTT连接失败

**可能原因**：
1. EMQX服务未启动：`systemctl status emqx`
2. EMQX监听地址错误：检查是否绑定到 `0.0.0.0`
3. HTTP认证器配置错误导致拒绝连接

**诊断命令**：
```bash
# 检查EMQX状态
systemctl status emqx

# 查看EMQX日志
journalctl -u emqx -f

# 检查端口监听
netstat -tlnp | grep emqx
ss -tlnp | grep emqx
```

### Q4: 安全组规则显示成功但仍不通

**可能原因**：
1. **NACL（网络ACL）拦截**：检查子网的NACL是否允许入站
2. **路由表问题**：EC2没有Internet Gateway路由
3. **VPC对等连接限制**：跨VPC通信未正确配置

**解决**：
```bash
# 检查NACL
aws ec2 describe-network-acls --network-acl-ids acl-xxxxxxxx

# 检查路由表
aws ec2 describe-route-tables --route-table-ids rtb-xxxxxxxx
```

---

## 📊 八、安全组最佳实践总结

### ✅ 推荐配置（生产环境）

| 端口 | Source | 说明 |
|------|--------|------|
| 1883 | 0.0.0.0/0 | MQTT必须对所有设备开放 |
| 8883 | 0.0.0.0/0 | 加密连接可选开放 |
| 18083 | 你的IP/32 | **严格限制！仅管理员IP** |
| 8083 | VPC CIDR 或 前端服务器IP | 限制Web访问范围 |

### ❌ 危险配置（禁止使用）

| 端口 | Source | 风险等级 |
|------|--------|----------|
| 18083 | 0.0.0.0/0 | 🔴 极高（任何人可管理Broker）|
| 18084 | 0.0.0.0/0 | 🔴 高（API接口暴露）|

---

## 🎯 九、快速检查清单

部署完成后，逐项确认：

- [ ] AWS安全组已添加TCP 1883规则（来源：0.0.0.0/0）
- [ ] AWS安全组已添加TCP 18083规则（来源：你的IP/32）
- [ ] Linux防火墙(firewalld/iptables)已放行上述端口
- [ ] 本地能Ping通14.103.202.69
- [ ] 本地能Telnet通1883和18083端口
- [ ] 浏览器能打开 http://14.103.202.69:18083
- [ ] 能用MQTT客户端工具连接mqtt://14.103.202.69:1883
- [ ] 连接时触发HTTP认证请求到后端服务
- [ ] 数据库device_auth_log表有认证记录生成

全部打勾后即可进入下一阶段：**设备联调测试**

---

## 📚 十、相关文档

- [EMQX官方安装文档](https://www.emqx.io/docs/en/latest/deploy/install.html)
- [AWS安全组最佳实践](https://docs.aws.amazon.com/vpc/latest/userguide/VPC_SecurityGroups.html)
- [EMQX HTTP认证配置](./EMQX_MQTT_AUTH_CONFIG.md)

---

**📅 文档版本**: v1.0  
**📅 更新日期**: 2026-05-12  
**👤 维护者**: Device Service Team  
**🌐 服务器**: 14.103.202.69 (Amazon Linux 2023)
