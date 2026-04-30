#!/bin/bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
USER_SERVICE_URL="${USER_SERVICE_URL:-http://localhost:8001}"
DEVICE_SERVICE_URL="${DEVICE_SERVICE_URL:-http://localhost:8002}"
CONTENT_SERVICE_URL="${CONTENT_SERVICE_URL:-http://localhost:8003}"
ADMIN_SERVICE_URL="${ADMIN_SERVICE_URL:-http://localhost:8000}"

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 打印测试结果
print_result() {
    local test_name=$1
    local status=$2
    local message=$3

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗${NC} $test_name: $message"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# 等待服务启动
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=0

    echo -e "${YELLOW}等待 $service_name 启动...${NC}"

    while [ $attempt -lt $max_attempts ]; do
        if curl -s -f "$url/health" > /dev/null 2>&1 || curl -s -f "$url/api/v1/health" > /dev/null 2>&1; then
            echo -e "${GREEN}$service_name 已就绪${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    echo -e "${RED}$service_name 启动超时${NC}"
    return 1
}

# 测试用户服务
test_user_service() {
    echo -e "\n${YELLOW}=== 测试用户服务 ===${NC}"

    # 测试用户注册
    response=$(curl -s -w "\n%{http_code}" -X POST "$USER_SERVICE_URL/api/v1/user/register" \
        -H "Content-Type: application/json" \
        -d '{"username":"testuser","password":"test123","email":"test@example.com"}')

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ] || [ "$http_code" = "400" ]; then
        print_result "用户注册接口" "PASS"
    else
        print_result "用户注册接口" "FAIL" "HTTP $http_code"
    fi

    # 测试用户登录
    response=$(curl -s -w "\n%{http_code}" -X POST "$USER_SERVICE_URL/api/v1/user/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"testuser","password":"test123","email":"test@example.com"}')

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    # 登录接口可能返回 200(成功), 400(用户不存在/参数错误), 401(密码错误)
    if [ "$http_code" = "200" ] || [ "$http_code" = "400" ] || [ "$http_code" = "401" ]; then
        print_result "用户登录接口" "PASS"
        # 提取 token（如果登录成功）
        if [ "$http_code" = "200" ]; then
            TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        fi
    else
        print_result "用户登录接口" "FAIL" "HTTP $http_code"
    fi

    # 用户分页列表已迁至管理后台：GET /api/v1/platform-user/list（需 admin JWT），不再对用户服务做 list 探测
}

# 测试设备服务
test_device_service() {
    echo -e "\n${YELLOW}=== 测试设备服务 ===${NC}"

    # 需要 token 的接口，使用 mock token
    local mock_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxfQ.test"

    # 测试设备列表
    response=$(curl -s -w "\n%{http_code}" -X GET "$DEVICE_SERVICE_URL/api/v1/device/list" \
        -H "Authorization: Bearer $mock_token")

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ]; then
        print_result "设备列表接口" "PASS"
    else
        print_result "设备列表接口" "FAIL" "HTTP $http_code"
    fi

    # 测试设备绑定
    response=$(curl -s -w "\n%{http_code}" -X POST "$DEVICE_SERVICE_URL/api/v1/device/bind" \
        -H "Authorization: Bearer $mock_token" \
        -H "Content-Type: application/json" \
        -d '{"device_id":"test-device-001","device_name":"Test Device"}')

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "400" ]; then
        print_result "设备绑定接口" "PASS"
    else
        print_result "设备绑定接口" "FAIL" "HTTP $http_code"
    fi

    # 测试设备心跳
    response=$(curl -s -w "\n%{http_code}" -X POST "$DEVICE_SERVICE_URL/api/v1/device/heartbeat" \
        -H "Authorization: Bearer $mock_token" \
        -H "Content-Type: application/json" \
        -d '{"device_id":"test-device-001"}')

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "400" ]; then
        print_result "设备心跳接口" "PASS"
    else
        print_result "设备心跳接口" "FAIL" "HTTP $http_code"
    fi
}

# 测试内容服务
test_content_service() {
    echo -e "\n${YELLOW}=== 测试内容服务 ===${NC}"

    local mock_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxfQ.test"

    # 测试内容列表
    response=$(curl -s -w "\n%{http_code}" -X GET "$CONTENT_SERVICE_URL/api/v1/content/list" \
        -H "Authorization: Bearer $mock_token")

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ]; then
        print_result "内容列表接口" "PASS"
    else
        print_result "内容列表接口" "FAIL" "HTTP $http_code"
    fi

    # 测试获取上传地址
    response=$(curl -s -w "\n%{http_code}" -X POST "$CONTENT_SERVICE_URL/api/v1/content/upload" \
        -H "Authorization: Bearer $mock_token" \
        -H "Content-Type: application/json" \
        -d '{"filename":"test.mp3","content_type":"audio/mpeg"}')

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "400" ]; then
        print_result "获取上传地址接口" "PASS"
    else
        print_result "获取上传地址接口" "FAIL" "HTTP $http_code"
    fi

    # 测试处理状态
    response=$(curl -s -w "\n%{http_code}" -X GET "$CONTENT_SERVICE_URL/api/v1/content/status?content_id=1" \
        -H "Authorization: Bearer $mock_token")

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "404" ]; then
        print_result "内容处理状态接口" "PASS"
    else
        print_result "内容处理状态接口" "FAIL" "HTTP $http_code"
    fi
}

# 测试 go-admin 服务
test_admin_service() {
    echo -e "\n${YELLOW}=== 测试 go-admin 服务 ===${NC}"

    # 测试健康检查
    response=$(curl -s -w "\n%{http_code}" "$ADMIN_SERVICE_URL/")
    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "404" ]; then
        print_result "go-admin 服务可访问" "PASS"
    else
        print_result "go-admin 服务可访问" "FAIL" "HTTP $http_code"
    fi

    # 测试登录接口
    response=$(curl -s -w "\n%{http_code}" -X POST "$ADMIN_SERVICE_URL/api/v1/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}')

    http_code=$(echo "$response" | tail -n 1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "400" ]; then
        print_result "go-admin 登录接口" "PASS"
    else
        print_result "go-admin 登录接口" "FAIL" "HTTP $http_code"
    fi
}

# 主函数
main() {
    echo -e "${YELLOW}开始 API 集成测试...${NC}\n"

    # 运行测试
    test_user_service
    test_device_service
    test_content_service
    test_admin_service

    # 打印测试总结
    echo -e "\n${YELLOW}=== 测试总结 ===${NC}"
    echo -e "总测试数: $TOTAL_TESTS"
    echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
    echo -e "${RED}失败: $FAILED_TESTS${NC}"

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}所有测试通过！${NC}"
        exit 0
    else
        echo -e "\n${RED}部分测试失败${NC}"
        exit 1
    fi
}

main
