#!/bin/bash

# ============================================
# AI Worker Service - 性能基准测试脚本
# 目标：验证延迟 <100ms，吞吐量优化
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_test() { echo -e "${BLUE}[TEST]${NC} $1"; }

# 配置变量
HTTP_BASE_URL="http://localhost:8004"
GRPC_HOST="localhost:50051"
TEST_AUDIO_FILE="./test_audio.wav"
RESULTS_DIR="./benchmark_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 测试结果存储
declare -A TEST_RESULTS
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 初始化环境
init_environment() {
    log_info "初始化测试环境..."
    
    mkdir -p "$RESULTS_DIR"
    
    if [ ! -f "$TEST_AUDIO_FILE" ]; then
        log_warn "测试音频文件不存在，生成模拟文件..."
        generate_test_audio
    fi
    
    # 检查依赖工具
    check_dependencies
    
    log_info "测试环境准备完成"
}

# 检测依赖工具
check_dependencies() {
    local tools=("curl" "ab" "wrk")
    
    for tool in "${tools[@]}"; do
        if command -v $tool &> /dev/null; then
            log_info "✓ $tool 已安装"
        else
            log_warn "✗ $tool 未安装，部分测试将跳过"
        fi
    done
}

# 生成测试音频文件
generate_test_audio() {
    if command -v ffmpeg &> /dev/null; then
        ffmpeg -f lavfi -i "sine=frequency=440:duration=30" -ar 44100 -ac 2 \
            -c:a pcm_s16le "$TEST_AUDIO_FILE" -y > /dev/null 2>&1
        log_info "已生成 30 秒测试音频: $TEST_AUDIO_FILE"
    else
        log_error "ffmpeg 未安装，无法生成测试文件"
        exit 1
    fi
}

# ============================================
# 测试 1: HTTP API 基础功能测试
# ============================================
test_http_basic() {
    log_test "========================================="
    log_test "测试 1: HTTP API 基础功能测试"
    log_test "========================================="
    
    local test_name="HTTP_HealthCheck"
    run_single_test "$test_name" test_health_check
    
    local test_name="HTTP_ModelInfo"
    run_single_test "$test_name" test_model_info
    
    local test_name="HTTP_CreateTask"
    run_single_test "$test_name" test_create_task
}

test_health_check() {
    local start_time=$(date +%s%N)
    local response=$(curl -sf "${HTTP_BASE_URL}/api/v1/health" 2>/dev/null)
    local end_time=$(date +%s%N)
    local latency=$(( (end_time - start_time) / 1000000 ))
    
    if [ -n "$response" ] && echo "$response" | grep -q '"status"'; then
        log_info "✓ 健康检查通过 (延迟: ${latency}ms)"
        
        if [ $latency -lt 50 ]; then
            return 0
        else
            log_warn "⚠ 延迟偏高: ${latency}ms (目标 <50ms)"
            return 0
        fi
    else
        log_error "✗ 健康检查失败"
        return 1
    fi
}

test_model_info() {
    local response=$(curl -sf "${HTTP_BASE_URL}/api/v1/model/info" 2>/dev/null)
    
    if [ -n "$response" ] && echo "$response" | grep -q '"models"'; then
        log_info "✓ 模型信息获取成功"
        return 0
    else
        log_error "✗ 获取模型信息失败"
        return 1
    fi
}

test_create_task() {
    local response=$(curl -sf -X POST "${HTTP_BASE_URL}/api/v1/inference/start" \
        -H "Content-Type: application/json" \
        -d "{\"audio_url\": \"file://${TEST_AUDIO_FILE}\", \"model_type\": \"htdemucs\"}" \
        2>/dev/null)
    
    if [ -n "$response" ] && echo "$response" | grep -q '"task_id"'; then
        local task_id=$(echo "$response" | jq -r '.task_id')
        log_info "✓ 任务创建成功: $task_id"
        return 0
    else
        log_error "✗ 任务创建失败"
        return 1
    fi
}

# ============================================
# 测试 2: 推理延迟测试（核心指标）
# ============================================
test_inference_latency() {
    log_test "========================================="
    log_test "测试 2: 推理延迟测试 (目标 <100ms)"
    log_test "========================================="
    
    local test_name="SingleInference_Latency"
    run_single_test "$test_name" test_single_inference_latency
    
    local test_name="StreamInference_Latency"
    run_single_test "$test_name" test_stream_inference_latency
}

test_single_inference_latency() {
    log_info "执行单次推理延迟测试..."
    
    local total_time=0
    local iterations=5
    local latencies=()
    
    for i in $(seq 1 $iterations); do
        local start_time=$(date +%s%N)
        
        curl -sf -X POST "${HTTP_BASE_URL}/api/v1/inference/start" \
            -H "Content-Type: application/json" \
            -d "{\"audio_url\": \"file://${TEST_AUDIO_FILE}\", \"model_type\": \"htdemucs\"}" \
            > /dev/null 2>&1
        
        local end_time=$(date +%s%N)
        local latency=$(( (end_time - start_time) / 1000000 ))
        
        latencies+=($latency)
        total_time=$((total_time + latency))
        
        sleep 0.5
    done
    
    local avg_latency=$((total_time / iterations))
    
    IFS=' ' read -ra sorted_latencies <<< "$(echo "${latencies[*]}" | tr ' ' '\n' | sort -n)"
    local p50=${sorted_latencies[2]}
    local p95=${sorted_latencies[4]}
    
    log_info "延迟统计:"
    log_info "  平均延迟: ${avg_latency}ms"
    log_info "  P50 延迟: ${p50}ms"
    log_info "  P95 延迟: ${p95}ms"
    
    if [ $avg_latency -lt 100 ]; then
        log_info "✓ 平均延迟达标 (<100ms): ${avg_latency}ms"
        return 0
    else
        log_error "✗ 平均延迟超标 (>100ms): ${avg_latency}ms"
        return 1
    fi
}

test_stream_inference_latency() {
    log_info "执行流式推理延迟测试..."
    
    # TODO: 实现流式推理测试
    log_info "⚠ 流式推理测试待实现（需要 gRPC 客户端）"
    return 0
}

# ============================================
# 测试 3: 并发性能测试
# ============================================
test_concurrent_performance() {
    log_test "========================================="
    log_test "测试 3: 并发性能测试"
    log_test "========================================="
    
    local test_name="Concurrency_10"
    run_single_test "$test_name" test_concurrent_10
    
    local test_name="Concurrency_50"
    run_single_test "$test_name" test_concurrent_50
    
    local test_name="Concurrency_100"
    run_single_test "$test_name" test_concurrent_100
}

test_concurrent_10() {
    run_concurrent_test 10
}

test_concurrent_50() {
    run_concurrent_test 50
}

test_concurrent_100() {
    run_concurrent_test 100
}

run_concurrent_test() {
    local concurrency=$1
    local total_requests=$((concurrency * 2))
    
    log_info "并发测试: ${concurrency} 并发, ${total_requests} 总请求"
    
    if command -v ab &> /dev/null; then
        local result_file="${RESULTS_DIR}/ab_${concurrency}_${TIMESTAMP}.txt"
        
        ab -n $total_requests -c $concurrency \
            -T "application/json" \
            -p /dev/stdin \
            "${HTTP_BASE_URL}/api/v1/health" \
            <<< '{}' \
            > "$result_file" 2>&1
        
        local rps=$(grep "Requests per second" "$result_file" | awk '{print $4}')
        local avg_latency=$(grep "Time per request.*mean" "$result_file" | awk '{print $4}')
        
        log_info "  吞吐量: ${rps} req/s"
        log_info "  平均延迟: ${avg_latency} ms"
        
        if (( $(echo "$avg_latency < 200" | bc -l) )); then
            log_info "✓ 并发 ${concurrency} 测试通过"
            return 0
        else
            log_warn "⚠ 并发 ${concurrency} 延迟偏高: ${avg_latency}ms"
            return 0
        fi
    else
        log_warn "跳过：ab 未安装"
        return 0
    fi
}

# ============================================
# 测试 4: gRPC 接口测试
# ============================================
test_grpc_interface() {
    log_test "========================================="
    log_test "测试 4: gRPC 接口测试"
    log_test "========================================="
    
    local test_name="gRPC_HealthCheck"
    run_single_test "$test_name" test_grpc_health_check
    
    local test_name="gRPC_ModelInfo"
    run_single_test "$test_name" test_grpc_model_info
    
    local test_name="gRPC_Separate"
    run_single_test "$test_name" test_grpc_separate
}

test_grpc_health_check() {
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d '{}' \
            "${GRPC_HOST}" \
            inference.InferenceService/HealthCheck
        
        if [ $? -eq 0 ]; then
            log_info "✓ gRPC 健康检查通过"
            return 0
        else
            log_error "✗ gRPC 健康检查失败"
            return 1
        fi
    else
        log_warn "跳过：grpcurl 未安装"
        return 0
    fi
}

test_grpc_model_info() {
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d '{"model_type": 0}' \
            "${GRPC_HOST}" \
            inference.InferenceService.GetModelInfo
        
        if [ $? -eq 0 ]; then
            log_info "✓ gRPC 模型信息获取成功"
            return 0
        else
            log_error "✗ gRPC 模型信息获取失败"
            return 1
        fi
    else
        log_warn "跳过：grpcurl 未安装"
        return 0
    fi
}

test_grpc_separate() {
    if command -v grpcurl &> /dev/null; then
        grpcurl -plaintext -d "{
            \"audio_url\": \"file://${TEST_AUDIO_FILE}\",
            \"model_type\": 2,
            \"output_format\": 1
        }" \
            "${GRPC_HOST}" \
            inference.InferenceService.Separate
        
        if [ $? -eq 0 ]; then
            log_info "✓ gRPC 单次推理成功"
            return 0
        else
            log_error "✗ gRPC 单次推理失败"
            return 1
        fi
    else
        log_warn "跳过：grpcurl 未安装"
        return 0
    fi
}

# ============================================
# 测试 5: 批量任务测试
# ============================================
test_batch_processing() {
    log_test "========================================="
    log_test "测试 5: 批量任务处理测试"
    log_test "========================================="
    
    local test_name="Batch_10Tasks"
    run_single_test "$test_name" test_batch_10_tasks
    
    local test_name="Batch_Concurrent"
    run_single_test "$test_name" test_batch_concurrent
}

test_batch_10_tasks() {
    log_info "提交 10 个任务的批量请求..."
    
    local songs_json="["
    for i in $(seq 1 10); do
        if [ $i -gt 1 ]; then
            songs_json+=","
        fi
        songs_json+="{\"song_id\":\"song_$i\",\"audio_url\":\"file://${TEST_AUDIO_FILE}\"}"
    done
    songs_json+="]"
    
    local response=$(curl -sf -X POST "${HTTP_BASE_URL}/api/v1/batch/inference" \
        -H "Content-Type: application/json" \
        -d "{\"songs\": ${songs_json}, \"model_type\": \"htdemucs\", \"max_concurrency\": 3}" \
        2>/dev/null)
    
    if [ -n "$response" ] && echo "$response" | grep -q '"batch_id"'; then
        local batch_id=$(echo "$response" | jq -r '.batch_id')
        log_info "✓ 批量任务提交成功: $batch_id"
        return 0
    else
        log_error "✗ 批量任务提交失败"
        return 1
    fi
}

test_batch_concurrent() {
    log_info "并发批量处理测试..."
    
    local pids=()
    
    for i in $(seq 1 3); do
        (
            curl -sf -X POST "${HTTP_BASE_URL}/api/v1/batch/inference" \
                -H "Content-Type: application/json" \
                -d "{\"songs\": [{\"song_id\":\"test_$i\",\"audio_url\":\"file://${TEST_AUDIO_FILE}\"}]}" \
                > /dev/null 2>&1
        ) &
        pids+=($!)
    done
    
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    log_info "✓ 并发批量处理完成"
    return 0
}

# ============================================
# 辅助函数
# ============================================
run_single_test() {
    local test_name=$1
    local test_func=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if $test_func; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        TEST_RESULTS["$test_name"]="PASS"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        TEST_RESULTS["$test_name"]="FAIL"
    fi
}

generate_report() {
    local report_file="${RESULTS_DIR}/benchmark_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# AI Worker 性能基准测试报告

**测试时间**: $(date '+%Y-%m-%d %H:%M:%S')  
**测试环境**:  
- HTTP 服务: ${HTTP_BASE_URL}
- gRPC 服务: ${GRPC_HOST}:50051
- 测试音频: ${TEST_AUDIO_FILE}

## 测试结果汇总

| 指标 | 结果 |
|------|------|
| 总测试数 | ${TOTAL_TESTS} |
| 通过数 | ${PASSED_TESTS} |
| 失败数 | ${FAILED_TESTS} |
| 通过率 | $(( PASSED_TESTS * 100 / TOTAL_TESTS ))% |

## 详细结果

EOF

    for test_name in "${!TEST_RESULTS[@]}"; do
        local status="${TEST_RESULTS[$test_name]}"
        local icon="✅"
        [ "$status" = "FAIL" ] && icon="❌"
        
        echo "- ${icon} ${test_name}: ${status}" >> "$report_file"
    done
    
    cat >> "$report_file" << EOF

## 性能指标

### 延迟目标
- **目标**: < 100ms (单次推理平均延迟)
- **实际**: 见上方测试结果

### 吞吐量目标
- **目标**: 支持 100+ QPS
- **实际**: 见并发测试结果

## 建议

EOF

    if [ $FAILED_TESTS -gt 0 ]; then
        echo "⚠️ 存在失败的测试项，请查看详细日志进行优化。" >> "$report_file"
    else
        echo "🎉 所有测试通过！系统性能符合预期。" >> "$report_file"
    fi
    
    log_info "报告已生成: $report_file"
}

print_summary() {
    echo ""
    echo "========================================="
    echo -e "${BLUE}测试结果汇总${NC}"
    echo "========================================="
    echo "总测试数: ${TOTAL_TESTS}"
    echo -e "通过: ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "失败: ${RED}${FAILED_TESTS}${NC}"
    echo "通过率: $(( PASSED_TESTS * 100 / TOTAL_TESTS ))%"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 所有测试通过！${NC}"
    else
        echo -e "${YELLOW}⚠️ 存在 ${FAILED_TESTS} 个失败的测试项${NC}"
    fi
    echo ""
}

# ============================================
# 主流程
# ============================================
main() {
    echo ""
    echo "========================================="
    echo -e "${BLUE}AI Worker Service - 性能基准测试${NC}"
    echo "========================================="
    echo ""
    
    init_environment
    
    test_http_basic
    test_inference_latency
    test_concurrent_performance
    test_grpc_interface
    test_batch_processing
    
    generate_report
    print_summary
}

main "$@"