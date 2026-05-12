#!/bin/bash
# ============================================================
# EMQX 5.8.9 远程服务器自动部署脚本
# 目标系统: Amazon Linux 2023 (amzn2023)
# 服务器IP: 14.103.202.69:26120
# 用途: 部署MQTT Broker并配置HTTP认证对接设备管理微服务
#
# 使用方式:
#   chmod +x deploy_emqx.sh
#   sudo ./deploy_emqx.sh
#
# 作者: Device Service Team
# 日期: 2026-05-12
# 版本: v1.0
# ============================================================

set -e  # 遇到错误立即退出

# ==================== 配置变量 ====================
EMQX_VERSION="5.8.9"
EMQX_RPM_FILE="emqx-${EMQX_VERSION}-amzn2023-amd64.rpm"
EMQX_DOWNLOAD_URL="https://www.emqx.com/en/downloads/broker/v${EMQX_VERSION}"
DEVICE_API_URL="http://14.103.202.69:8002"  # 设备管理微服务地址
MQTT_PORT=1883
MQTT_SSL_PORT=8883
DASHBOARD_PORT=18083
API_PORT=18084

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ==================== 步骤1: 检查权限 ====================
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 权限运行此脚本"
        log_info "使用方式: sudo $0"
        exit 1
    fi
    log_success "✅ 权限检查通过 (当前用户: $(whoami))"
}

# ==================== 步骤2: 安装依赖库 ====================
install_dependencies() {
    log_info "📦 正在安装 EMQX 运行所需的依赖库..."

    # 更新 yum 源
    dnf update -y || true

    # 安装基础工具
    dnf install -y \
        curl \
        wget \
        openssl \
        net-tools \
        lsof \
        unzip \
        jq \
        gnupg2 \
        ca-certificates

    # 安装 EMQX 特定依赖（Amazon Linux 2023）
    dnf install -y \
        libgcc \
        libstdc++ \
        libopenssl \
        zlib \
        ncurses-libs \
        readline

    log_success "✅ 依赖库安装完成"
}

# ==================== 步骤3: 上传并安装 EMQX ====================
upload_and_install_emqx() {
    log_info "📥 准备安装 EMQX ${EMQX_VERSION}..."

    # 检查是否已上传RPM包
    if [ ! -f "/tmp/${EMQX_RPM_FILE}" ]; then
        log_warn "⚠️  未找到本地RPM包，尝试从网络下载..."

        # 尝试下载（如果网络允许）
        cd /tmp
        wget --timeout=30 \
            "${EMQX_DOWNLOAD_URL}" \
            -O "${EMQX_RPM_FILE}" || {
                log_error "❌ 无法下载 EMQX 安装包"
                log_error "请手动上传文件到服务器: scp ${EMQX_RPM_FILE} root@14.103.202.69:/tmp/"
                exit 1
            }
    fi

    log_info "🔧 开始安装 EMQX..."
    dnf install -y "/tmp/${EMQX_RPM_FILE}"

    # 验证安装
    if command -v emqx &> /dev/null; then
        emqx_version=$(emqx version 2>/dev/null || echo "${EMQX_VERSION}")
        log_success "✅ EMQX ${emqx_version} 安装成功"
    else
        log_error "❌ EMQX 安装失败"
        exit 1
    fi
}

# ==================== 步骤4: 启动 EMQX 服务 ====================
start_emqx_service() {
    log_info "🚀 启动 EMQX 服务..."

    # 启用开机自启
    systemctl enable emqx

    # 启动服务
    systemctl start emqx

    # 等待服务启动完成
    sleep 10

    # 检查状态
    if systemctl is-active --quiet emqx; then
        log_success "✅ EMQX 服务启动成功"
        log_info "   状态: $(systemctl is-active emqx)"
    else
        log_error "❌ EMQX 服务启动失败，查看日志:"
        journalctl -u emqx -n 50 --no-pager
        exit 1
    fi

    # 检查端口监听
    sleep 5
    check_ports
}

# ==================== 步骤5: 检查端口占用 ====================
check_ports() {
    log_info "🔍 检查关键端口..."

    ports=("${MQTT_PORT}:MQTT协议" "${DASHBOARD_PORT}:Dashboard控制台" "${API_PORT}:API接口")

    for port_info in "${ports[@]}"; do
        port=${port_info%%:*}
        desc=${port_info##*:}

        if netstat -tuln | grep -q ":${port} "; then
            log_success "   ✅ 端口 ${port} (${desc}) 已监听"
        else
            log_warn "   ⚠️  端口 ${port} (${desc}) 未监听，可能需要等待..."
        fi
    done
}

# ==================== 步骤6: 开放防火墙端口 ====================
configure_firewall() {
    log_info "🔥 配置防火墙规则..."

    # 检查 firewalld 是否运行
    if systemctl is-active --quiet firewalld; then
        log_info "检测到 firewalld，正在开放端口..."

        # 开放 MQTT 端口
        firewall-cmd --permanent --add-port=${MQTT_PORT}/tcp
        firewall-cmd --permanent --add-port=${MQTT_SSL_PORT}/tcp

        # 开放 Dashboard 端口
        firewall-cmd --permanent --add-port=${DASHBOARD_PORT}/tcp
        firewall-cmd --permanent --add-port=${API_PORT}/tcp

        # 重载防火墙
        firewall-cmd --reload

        log_success "✅ 防火墙端口已开放"

    elif command -v iptables &> /dev/null; then
        log_info "使用 iptables 开放端口..."

        iptables -I INPUT -p tcp --dport ${MQTT_PORT} -j ACCEPT
        iptables -I INPUT -p tcp --dport ${MQTT_SSL_PORT} -j ACCEPT
        iptables -I INPUT -p tcp --dport ${DASHBOARD_PORT} -j ACCEPT
        iptables -I INPUT -p tcp --dport ${API_PORT} -j ACCEPT

        # 保存规则
        iptables-save > /etc/iptables.rules

        log_success "✅ iptables 规则已添加"
    else
        log_warn "⚠️  未检测到防火墙，请手动开放以下端口:"
        log_warn "   - TCP ${MQTT_PORT} (MQTT)"
        log_warn "   - TCP ${MQTT_SSL_PORT} (MQTTS/SSL)"
        log_warn "   - TCP ${DASHBOARD_PORT} (Dashboard)"
        log_warn "   - TCP ${API_PORT} (Management API)"
    fi
}

# ==================== 步骤7: 配置 EMQX Dashboard ====================
configure_dashboard() {
    log_info "⚙️  配置 EMQX Dashboard..."

    local dashboard_user="admin"
    local dashboard_pass="audio_platform_2026"

    # 修改默认密码（通过 API）
    local api_url="http://127.0.0.1:${API_PORT}"

    # 等待 API 就绪
    local retry=0
    while [ $retry -lt 30 ]; do
        if curl -s -o /dev/null -w "%{http_code}" "${api_url}/status" | grep -q "200"; then
            break
        fi
        sleep 2
        ((retry++))
    done

    if [ $retry -eq 30 ]; then
        log_warn "⚠️  EMQX API 未就绪，跳过 Dashboard 密码修改"
        return
    fi

    # 修改管理员密码
    local result=$(curl -s -X PUT "${api_url}/api/v5/users/admin" \
        -u admin:public \
        -H "Content-Type: application/json" \
        -d "{\"new_password\":\"${dashboard_pass}\",\"old_password\":\"public\"}" 2>/dev/null)

    if echo "$result" | grep -q "\"code\":0"; then
        log_success "✅ Dashboard 密码已修改"
        log_info "   用户名: ${dashboard_user}"
        log_info "   密码: ${dashboard_pass}"
        log_info "   地址: http://14.103.202.69:${DASHBOARD_PORT}"
    else
        log_warn "⚠️  Dashboard 密码修改失败（可稍后在Web界面手动修改）"
    fi
}

# ==================== 步骤8: 配置 HTTP 认证器（核心功能） ====================
configure_http_auth() {
    log_info "🔐 配置 HTTP 认证器（对接设备管理微服务）..."

    local api_url="http://127.0.0.1:${API_PORT}"

    # 创建认证配置 JSON
    local auth_config=$(cat <<EOF
{
    "enable": true,
    "backend": "http",
    "method": "post",
    "url": "${DEVICE_API_URL}/mqtt/auth",
    "headers": {
        "Content-Type": "application/json"
    },
    "body": "{\"clientid\":\"\\${clientid}\",\"username\":\"\\${username}\",\"password\":\"\\${password}\"}",
    "connect_timeout": "5s",
    "pool_size": 32,
    "ssl": {
        "enable": false
    }
}
EOF
)

    # 调用 EMQX API 创建认证数据源
    local result=$(curl -s -X POST "${api_url}/api/v5/authentication" \
        -u "admin:${dashboard_pass:-public}" \
        -H "Content-Type: application/json" \
        -d "${auth_config}")

    if echo "$result" | grep -q "\"code\":0\|\"code\": 0"; then
        log_success "✅ HTTP 认证器配置成功"
        log_info "   认证URL: ${DEVICE_API_URL}/mqtt/auth"
    else
        log_warn "⚠️  API 配置可能失败，将使用配置文件方式..."

        # 备用方案：写入配置文件
        configure_emqx_conf_file
    fi
}

# ==================== 备用方案：通过配置文件设置认证 ====================
configure_emqx_conf_file() {
    log_info "📝 通过配置文件设置 HTTP 认证..."

    local conf_file="/etc/emqx/emqx.conf"

    # 备份原配置
    cp "${conf_file}" "${conf_file}.backup.$(date +%Y%m%d%H%M%S)"

    # 追加认证配置到文件末尾
    cat >> "${conf_file}" <<EOF

## ============================================================
## HTTP Authentication (Auto-configured by deploy script)
## Date: $(date)
## Backend: Device Management Microservice
## ============================================================

authentication = {
    enable = true
    backend = "http"
    http = {
        method = post
        url = "${DEVICE_API_URL}/mqtt/auth"
        headers.content-type = "application/json"
        body = "{\"clientid\":\"\${clientid}\",\"username\":\"\${username}\",\"password\":\"\${password}\"}"
        connect_timeout = "5s"
        pool_size = 32
    }
}
EOF

    # 重启 EMQX 使配置生效
    systemctl restart emqx
    sleep 5

    log_success "✅ 配置文件已更新，EMQX 已重启"
}

# ==================== 步骤9: 显示部署结果 ====================
show_deployment_result() {
    echo ""
    echo "=========================================="
    echo -e "${GREEN}🎉 EMQX 部署完成！${NC}"
    echo "=========================================="
    echo ""
    echo -e "${BLUE}📍 服务器信息:${NC}"
    echo "   服务器IP: 14.103.202.69"
    echo "   系统版本: Amazon Linux 2023"
    echo ""
    echo -e "${BLUE}🌐 访问地址:${NC}"
    echo "   MQTT Broker: mqtt://14.103.202.69:${MQTT_PORT}"
    echo "   Dashboard:   http://14.103.202.69:${DASHBOARD_PORT}"
    echo "   Management API: http://14.103.202.69:${API_PORT}"
    echo ""
    echo -e "${BLUE}👤 默认账号:${NC}"
    echo "   用户名: admin"
    echo "   密码: audio_platform_2026 (如已修改)"
    echo ""
    echo -e "${BLUE}🔐 认证配置:${NC}"
    echo "   认证类型: HTTP Server"
    echo "   认证URL: ${DEVICE_API_URL}/mqtt/auth"
    echo "   后端服务: Device Management Microservice"
    echo ""
    echo -e "${BLUE}📋 下一步操作:${NC}"
    echo "   1. 浏览器打开 Dashboard: http://14.103.202.69:${DASHBOARD_PORT}"
    echo "   2. 登录并验证认证配置: Extensions → Authentication"
    echo "   3. 使用 MQTT 客户端测试连接（需先在数据库录入设备）"
    echo "   4. 查看认证日志: SELECT * FROM device_auth_log ORDER BY created_at DESC;"
    echo ""
    echo -e "${YELLOW}⚠️  重要提示:${NC}"
    echo "   - 请确保 AWS 安全组已开放上述端口"
    echo "   - 请确保设备管理微服务(8002端口)正在运行"
    echo "   - 生产环境请修改默认密码"
    echo ""
    echo "=========================================="
}

# ==================== 主流程 ====================
main() {
    echo ""
    echo "=========================================="
    echo "  EMQX ${EMQX_VERSION} 自动部署脚本"
    echo "  目标: Amazon Linux 2023 @ 14.103.202.69"
    echo "=========================================="
    echo ""

    # 执行各步骤
    check_root
    install_dependencies
    upload_and_install_emqx
    start_emqx_service
    configure_firewall
    configure_dashboard
    configure_http_auth
    show_deployment_result

    exit 0
}

# 执行主函数
main "$@"
