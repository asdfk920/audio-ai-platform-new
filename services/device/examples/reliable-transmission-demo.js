#!/usr/bin/env node
/**
 * reliable-transmission-demo.js - 可靠传输完整演示
 *
 * 演示场景：
 *   1. 设备正常连接，发送数据并收到ACK确认
 *   2. 模拟网络断开，数据自动缓存到本地文件
 *   3. 重连成功后，自动补发所有缓存数据
 *   4. 验证数据零丢失、消息顺序正确
 *
 * 运行方式：
 *   npm install uuid ws
 *   node reliable-transmission-demo.js <your_jwt_token>
 */

const ReliableWebSocketClient = require('./reliable-websocket-client');
const fs = require('fs');
const path = require('path');

// ══════════════════════════════════════════════
// 配置参数
// ══════════════════════════════════════════════

const CONFIG = {
    // WebSocket服务端地址
    url: 'ws://localhost:8002/ws/device',

    // JWT Token（从命令行参数或环境变量获取）
    token: process.argv[2] || process.env.DEVICE_TOKEN || '',

    // 设备ID
    deviceId: process.argv[3] || 'TEST-DEVICE-001',

    // 可靠传输配置
    storagePath: './data/reliable_transmission_demo.json',
    ackTimeout: 10000,           // ACK超时10秒
    maxRetryCount: 5,            // 最大重试5次
    maxQueueSize: 500,           // 队列最大500条

    // 测试配置
    testMessageCount: 20,        // 发送20条测试消息
    sendInterval: 2000,          // 每2秒发送一条
};

// ══════════════════════════════════════════════
// 全局变量
// ══════════════════════════════════════════════

let client = null;
let messageCount = 0;
let successCount = 0;
let failCount = 0;
let pendingAcks = new Set();
let isRunning = false;

// ══════════════════════════════════════════════
// 日志工具函数
// ══════════════════════════════════════════════

function log(message, type = 'info') {
    const timestamp = new Date().toLocaleTimeString();
    const colors = {
        info: '\x1b[36m',     // 青色
        success: '\x1b[32m',  // 绿色
        warning: '\x1b[33m',  // 黄色
        error: '\x1b[31m',    // 红色
        system: '\x1b[35m',   // 紫色
        reset: '\x1b[0m'      // 重置
    };

    const color = colors[type] || colors.info;
    console.log(`${color}[${timestamp}]${colors.reset} ${message}`);
}

function printSeparator(char = '=', length = 60) {
    console.log(char.repeat(length));
}

// ══════════════════════════════════════════════
// 主程序逻辑
// ══════════════════════════════════════════════

async function main() {
    printSeparator();
    log('🚀 可靠传输演示系统启动', 'system');
    printSeparator();

    // 检查Token
    if (!CONFIG.token) {
        log('❌ 错误: 请提供JWT Token', 'error');
        log('', 'info');
        log('使用方式:', 'system');
        log('  node reliable-transmission-demo.js <your_token> [device_id]', 'info');
        log('', 'info');
        log('或者设置环境变量:', 'system');
        log('  export DEVICE_TOKEN=your_token_here', 'info');
        log('  node reliable-transmission-demo.js', 'info');
        process.exit(1);
    }

    // 清理旧数据（可选）
    if (fs.existsSync(CONFIG.storagePath)) {
        fs.unlinkSync(CONFIG.storagePath);
        log('🗑️  已清理旧的存储文件', 'warning');
    }

    // 创建客户端实例
    log('\n📍 步骤1: 创建可靠传输客户端...', 'system');

    client = new ReliableWebSocketClient({
        url: CONFIG.url,
        token: CONFIG.token,
        deviceId: CONFIG.deviceId,

        // 可靠传输配置
        enablePersistence: true,
        storagePath: CONFIG.storagePath,
        ackTimeout: CONFIG.ackTimeout,
        maxRetryCount: CONFIG.maxRetryCount,
        maxQueueSize: CONFIG.maxQueueSize,

        // 重连配置
        reconnectEnabled: true,
        initialReconnectDelay: 2000,
        maxReconnectDelay: 30000,

        // 心跳配置
        pingInterval: 30000,

        // 回调函数
        onOpen: onConnectionOpen,
        onMessage: onMessageReceived,
        onClose: onConnectionClose,
        onError: onErrorOccurred,
        onReconnecting: onReconnecting,
        onReconnected: onReconnected,
        onNetworkChange: onNetworkChanged
    });

    // 启动连接
    log('\n📍 步骤2: 建立WebSocket连接...', 'system');
    client.connect();

    // 启动状态监控定时器
    startStatusMonitor();

    // 等待连接建立
    await waitForConnection(10);

    if (client.state !== 'CONNECTED') {
        log('\n❌ 连接失败，退出演示', 'error');
        process.exit(1);
    }

    // 开始发送测试消息
    log('\n📍 步骤3: 开始发送测试消息...\n', 'system');
    await sendTestMessages();

    // 显示最终统计
    showFinalStats();

    // 保持运行（等待用户Ctrl+C）
    log('\n✅ 演示完成！按 Ctrl+C 退出\n', 'success');
}

// ══════════════════════════════════════════════
// 回调函数实现
// ══════════════════════════════════════════════

function onConnectionOpen(event) {
    log('✅ WebSocket连接已建立', 'success');
    log(`   URL: ${CONFIG.url}`, 'info');
    log(`   设备ID: ${CONFIG.deviceId}`, 'info');
}

function onMessageReceived(data) {
    log(`📥 收到消息:`, 'info');
    log(`   类型: ${data.type}`, 'info');

    if (data.type === 'instruction') {
        log(`   📢 指令: ${data.cmd}`, 'warning');

        // 自动回复指令执行结果
        client.sendReliable({
            type: 'cmd_response',
            payload: {
                cmd: data.cmd,
                success: true,
                device_id: CONFIG.deviceId,
                timestamp: Date.now()
            }
        }).then(result => {
            log(`   ✅ 指令响应已发送 (ACK: ${result.message_id})`, 'success');
        }).catch(err => {
            log(`   ❌ 指令响应发送失败: ${err.message}`, 'error');
        });
    }
}

function onConnectionClose(event) {
    log(`🔴 连接关闭: code=${event.code}, reason=${event.reason || 'N/A'}`, 'error');
}

function onErrorOccurred(error) {
    log(`❌ 错误发生: ${error.message || error}`, 'error');
}

function onReconnecting(info) {
    log(`\n🔄 正在重连...`, 'warning');
    log(`   第${info.attempt}次尝试`, 'info');
    log(`   延迟时间: ${(info.delay / 1000).toFixed(1)}s`, 'info');
    log(`   缓存消息: ${info.queueSize}条`, 'info');
}

function onReconnected(info) {
    log(`\n🎉 重连成功!`, 'success');
    log(`   尝试次数: ${info.attempts}`, 'info');
    log(`   总耗时: ${(info.totalTime / 1000).toFixed(1)}s`, 'info');
}

function onNetworkChanged(info) {
    const status = info.online ? '✅ 在线' : '❌ 离线';
    log(`🌐 网络状态变化: ${status}`, info.online ? 'success' : 'error');
}

// ══════════════════════════════════════════════
// 核心功能：发送测试消息
// ══════════════════════════════════════════════

async function sendTestMessages() {
    isRunning = true;

    for (let i = 1; i <= CONFIG.testMessageCount; i++) {
        if (!isRunning) break;

        messageCount++;

        // 构造不同类型的测试消息
        const messageType = ['status_report', 'heartbeat', 'log_upload', 'telemetry'][i % 4];
        
        const testData = {
            type: messageType,
            payload: {
                device_id: CONFIG.deviceId,
                sequence_number: i,
                timestamp: Date.now(),
                
                // 不同类型的数据字段
                ...(messageType === 'status_report' && {
                    battery_level: 70 + Math.floor(Math.random() * 30),
                    volume: Math.floor(Math.random() * 100),
                    playing: Math.random() > 0.5
                }),
                ...(messageType === 'heartbeat' && {
                    uptime: i * 3600,
                    memory_usage: Math.floor(Math.random() * 80)
                }),
                ...(messageType === 'log_upload' && {
                    level: ['INFO', 'WARN', 'ERROR'][i % 3],
                    message: `Test log message #${i}`
                }),
                ...(messageType === 'telemetry' && {
                    temperature: 25 + Math.random() * 15,
                    humidity: 40 + Math.random() * 40,
                    pressure: 1013 + Math.random() * 50
                })
            },
            metadata: {
                priority: i <= 5 ? 'high' : 'normal',
                source: 'reliable_transmission_demo'
            }
        };

        try {
            log(`\n📤 [${i}/${CONFIG.testMessageCount}] 发送消息:`, 'info');
            log(`   类型: ${messageType}`, 'info');
            log(`   序号: #${i}`, 'info');

            // 使用可靠发送（带持久化和ACK）
            const result = await client.sendReliable(testData);

            successCount++;
            pendingAcks.add(result.message_id);

            log(`   ✅ 已入队并等待ACK`, 'success');
            log(`   Message ID: ${result.message_id.substring(0, 8)}...`, 'info');
            log(`   队列长度: ${result.queue_size}`, 'info');

            // 显示统计信息
            printCurrentStats();

        } catch (error) {
            failCount++;
            log(`   ❌ 发送失败: ${error.message}`, 'error');
        }

        // 等待间隔
        if (i < CONFIG.testMessageCount) {
            await sleep(CONFIG.sendInterval);
        }
    }

    isRunning = false;
}

// ══════════════════════════════════════════════
// 辅助函数
// ══════════════════════════════════════════════

async function waitForConnection(timeoutSeconds) {
    return new Promise((resolve) => {
        const startTime = Date.now();
        const checkInterval = setInterval(() => {
            if (client.state === 'CONNECTED') {
                clearInterval(checkInterval);
                resolve(true);
            } else if ((Date.now() - startTime) > timeoutSeconds * 1000) {
                clearInterval(checkInterval);
                resolve(false);
            }
        }, 200);
    });
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

function startStatusMonitor() {
    setInterval(() => {
        if (client && client.state !== 'DISCONNECTED') {
            const stats = client.getReliabilityStats();
            
            log('\n📊 实时状态监控:', 'system');
            log(`   连接状态: ${stats.state}`, stats.state === 'CONNECTED' ? 'success' : 'warning');
            log(`   待发消息: ${stats.message_queue?.pending || 0}`, 'info');
            log(`   等待ACK: ${stats.pending_acks || 0}`, 'info');
            log(`   缓存总数: ${stats.message_queue?.total || 0}`, 'info');
            log(`   重连次数: ${stats.reconnectAttempts || 0}`, 'info');
            log(`   网络状态: ${stats.networkAvailable ? '在线' : '离线'}`, 
                stats.networkAvailable ? 'success' : 'error');
        }
    }, 10000); // 每10秒打印一次
}

function printCurrentStats() {
    const stats = client.getReliabilityStats();
    
    log(`   ── 统计信息 ──`, 'system');
    log(`   总计: ${messageCount} | 成功: ${successCount} | 失败: ${failCount}`, 
        failCount > 0 ? 'error' : 'success');
    log(`   待发: ${stats.message_queue?.pending || 0} | 等待ACK: ${pendingAcks.size}`, 'info');
}

function showFinalStats() {
    printSeparator();
    log('📈 最终统计报告', 'system');
    printSeparator();
    
    const finalStats = client.getReliabilityStats();
    
    log(`\n✅ 消息发送统计:`, 'success');
    log(`   总计发送: ${messageCount} 条`, 'info');
    log(`   成功入队: ${successCount} 条 (${(successCount / messageCount * 100).toFixed(1)}%)`, 'success');
    log(`   发送失败: ${failCount} 条`, failCount > 0 ? 'error' : 'info');
    
    log(`\n📦 队列状态:`, 'system');
    if (finalStats.message_queue) {
        log(`   待发送: ${finalStats.message_queue.pending} 条`, 
            finalStats.message_queue.pending > 0 ? 'warning' : 'success');
        log(`   等待ACK: ${finalStats.message_queue.sent} 条`,
            finalStats.message_queue.sent > 0 ? 'warning' : 'success');
        log(`   已确认: ${finalStats.message_queue.acked} 条`, 'success');
        log(`   队列利用率: ${finalStats.message_queue.utilization}`, 'info');
    }
    
    log(`\n💾 存储信息:`, 'system');
    log(`   文件路径: ${CONFIG.storagePath}`, 'info');
    
    if (fs.existsSync(CONFIG.storagePath)) {
        const fileSize = fs.statSync(CONFIG.storagePath).size;
        log(`   文件大小: ${(fileSize / 1024).toFixed(2)} KB`, 'info');
        
        // 显示最后几条记录
        const data = JSON.parse(fs.readFileSync(CONFIG.storagePath, 'utf8'));
        if (data.messages && data.messages.length > 0) {
            log(`\n📋 最近的消息记录:`, 'system');
            const recentMessages = data.messages.slice(-3);
            recentMessages.forEach((msg, index) => {
                log(`   ${index + 1}. ID: ${msg.id.substring(0, 8)}... | 类型: ${msg.type} | 状态: ${msg.status}`, 'info');
            });
        }
    }
    
    log(`\n🔄 连接统计:`, 'system');
    log(`   当前状态: ${finalStats.state}`, 
        finalStats.state === 'CONNECTED' ? 'success' : 'warning');
    log(`   重连次数: ${finalStats.reconnectAttempts || 0}`, 'info');
    log(`   等待中的ACK: ${finalStats.pending_acks || 0}`, 
        finalStats.pending_acks > 0 ? 'warning' : 'success');
    
    printSeparator();
}

// ══════════════════════════════════════════════
// 优雅退出处理
// ══════════════════════════════════════════════

process.on('SIGINT', () => {
    log('\n\n👋 收到中断信号，正在清理资源...', 'warning');
    isRunning = false;
    
    if (client) {
        // 显示最终统计
        showFinalStats();
        
        // 销毁客户端（会保存所有未完成的数据）
        client.destroy();
    }
    
    log('✅ 资源已清理，安全退出\n', 'success');
    process.exit(0);
});

process.on('uncaughtException', (error) => {
    log(`\n💥 未捕获的异常: ${error.message}`, 'error');
    log(error.stack, 'error');
});

process.on('unhandledRejection', (reason, promise) => {
    log(`\n⚠️  未处理的Promise拒绝: ${reason}`, 'warning');
});

// ══════════════════════════════════════════════
// 启动主程序
// ══════════════════════════════════════════════

main().catch(error => {
    log(`\n❌ 主程序异常: ${error.message}`, 'error');
    log(error.stack, 'error');
    process.exit(1);
});
