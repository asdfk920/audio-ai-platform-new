/**
 * ReliableWebSocketClient - 可靠传输WebSocket客户端
 *
 * 核心功能：
 *   ✅ 本地持久化存储（断电不丢、重启不丢）
 *   ✅ ACK确认机制（服务端确认后才删除本地数据）
 *   ✅ 断连自动缓存（不管什么原因断连，数据不丢失）
 *   ✅ 重连自动补发（恢复连接后立即补发所有待发数据）
 *   ✅ 消息顺序保证（按时间顺序补发，不乱序）
 *   ✅ 重复发送判断（服务端根据消息ID去重）
 *
 * 工作流程：
 *   1. 应用层调用 client.sendReliable(data)
 *   2. 数据写入本地持久化队列 (PENDING状态)
 *   3. 尝试通过WebSocket发送
 *   4. 发送成功 → 等待服务端ACK
 *   5. 收到ACK → 删除本地数据
 *   6. 发送失败/断连 → 保持PENDING，重连后重试
 */

const PersistentMessageQueue = require('./persistent-message-queue');
const DeviceWebSocketClient = require('./device-websocket-client');

class ReliableWebSocketClient extends DeviceWebSocketClient {
  constructor(options = {}) {
    // 调用父类构造函数
    super(options);

    // 可靠传输配置
    this.enablePersistence = options.enablePersistence !== false; // 默认开启
    this.storagePath = options.storagePath || './data/device_message_queue.json';
    this.ackTimeout = options.ackTimeout || 10000; // ACK超时时间 10秒
    this.maxRetryCount = options.maxRetryCount || 5; // 最大重试次数

    // 初始化持久化队列
    if (this.enablePersistence) {
      this.messageQueue = new PersistentMessageQueue({
        storagePath: this.storagePath,
        maxQueueSize: options.maxQueueSize || 10000,
        autoSaveInterval: options.autoSaveInterval || 5000
      });

      console.log('[ReliableWS] ✅ 可靠传输模式已启用');
      console.log(`[ReliableWS] 📦 存储路径: ${this.storagePath}`);
      console.log(`[ReliableWS] ⏱️  ACK超时: ${this.ackTimeout}ms`);
      console.log(`[ReliableWS] 🔄 最大重试: ${this.maxRetryCount}次`);
    }

    // ACK等待列表 (messageId -> {timer, resolve, reject})
    this._pendingAcks = new Map();

    // 重写父类的回调，增加ACK处理
    const originalOnMessage = this.onMessage;
    this.onMessage = (data, raw) => {
      // 先检查是否是ACK消息
      if (data && data.type === 'ack') {
        this._handleAck(data);
        return;
      }

      // 其他消息透传给原始回调
      if (originalOnMessage) {
        originalOnMessage(data, raw);
      }
    };

    // 重写连接成功回调，启动消息发送循环
    const originalOnOpen = this.onOpen;
    this.onOpen = (event) => {
      console.log('[ReliableWS] 🚀 连接已建立，开始同步待发消息...');
      this._startMessageSyncLoop();

      if (originalOnOpen) {
        originalOnOpen(event);
      }
    };
  }

  /**
   * sendReliable - 可靠发送消息（带持久化和ACK确认）
   *
   * 流程：
   *   1. 写入本地持久化队列
   *   2. 如果在线，尝试立即发送
   *   3. 等待ACK确认
   *   4. 收到ACK后删除本地数据
   */
  async sendReliable(message, options = {}) {
    try {
      // 1. 写入本地持久化队列
      const enqueueResult = await this.messageQueue.enqueue({
        type: message.type || 'custom',
        payload: message,
        priority: options.priority || 0,
        metadata: options.metadata || {}
      });

      if (!enqueueResult.success) {
        throw new Error('入队失败: ' + enqueueResult.error);
      }

      const messageId = enqueueResult.message_id;

      console.log(`[ReliableWS] 📥 消息已入队:`);
      console.log(`   ID: ${messageId}`);
      console.log(`   类型: ${message.type}`);

      // 2. 如果在线，尝试立即发送
      if (this.state === 'CONNECTED') {
        this._sendMessageWithAck(messageId);
      } else {
        console.log(`[ReliableWS] 💾 当前离线，消息将在重连后发送`);
      }

      // 3. 返回Promise，等待ACK确认
      return new Promise((resolve, reject) => {
        // 设置ACK等待定时器
        const timer = setTimeout(() => {
          this._pendingAcks.delete(messageId);
          reject(new Error(`ACK超时 (${this.ackTimeout}ms): ${messageId}`));
        }, this.ackTimeout);

        // 存入等待列表
        this._pendingAcks.set(messageId, { timer, resolve, reject });
      });

    } catch (error) {
      console.error(`[ReliableWS] ❌ 可靠发送失败:`, error);
      throw error;
    }
  }

  /**
   * send - 兼容普通发送（不带ACK确认，向后兼容）
   */
  send(message) {
    // 如果开启了可靠模式，使用可靠发送
    if (this.enablePersistence) {
      return this.sendReliable(message).catch(err => {
        console.warn('[ReliableWS] ⚠️  可靠发送失败，降级为普通发送:', err.message);
        return super.send(message); // 降级为普通发送
      });
    }

    // 否则使用父类的普通发送
    return super.send(message);
  }

  /**
   * _sendMessageWithAck - 发送单条消息并等待ACK
   */
  _sendMessageWithAck(messageId) {
    const queueMessage = this.messageQueue.messages.find(m => m.id === messageId);

    if (!queueMessage) {
      console.error(`[ReliableWS] ❌ 未找到消息: ${messageId}`);
      return;
    }

    // 检查重试次数
    if (queueMessage.retry_count >= this.maxRetryCount) {
      console.error(`[ReliableWS] ❌ 消息达到最大重试次数: ${messageId} (${this.maxRetryCount}次)`);
      this.messageQueue.markAsFailed(messageId, `达到最大重试次数(${this.maxRetryCount})`);

      // 拒绝Promise
      const pending = this._pendingAcks.get(messageId);
      if (pending) {
        clearTimeout(pending.timer);
        pending.reject(new Error('最大重试次数已达'));
        this._pendingAcks.delete(messageId);
      }
      return;
    }

    // 构造发送包（包含message_id用于ACK匹配）
    const sendPayload = {
      ...queueMessage.payload,
      _meta: {
        message_id: messageId,
        timestamp: Date.now(),
        retry_count: queueMessage.retry_count + 1
      }
    };

    // 标记为SENT
    this.messageQueue.markAsSent(messageId);

    // 通过WebSocket发送
    try {
      const success = super.send(sendPayload);

      if (!success) {
        throw new Error('WebSocket发送失败');
      }

      console.log(`[ReliableWS] 📤 消息已发送 (等待ACK): ${messageId}`);

    } catch (error) {
      console.error(`[ReliableWS] ❌ 发送失败:`, error);
      this.messageQueue.markAsFailed(messageId, error.message);

      // 延迟重试
      setTimeout(() => {
        if (this.state === 'CONNECTED') {
          this._sendMessageWithAck(messageId);
        }
      }, Math.min(1000 * Math.pow(2, queueMessage.retry_count), 30000));
    }
  }

  /**
   * _handleAck - 处理服务端返回的ACK确认
   */
  _handleAck(ackData) {
    const messageId = ackData.message_id;

    if (!messageId) {
      console.warn('[ReliableWS] ⚠️  收到无效的ACK（缺少message_id）:', ackData);
      return;
    }

    console.log(`[ReliableWS] ✅ 收到ACK确认:`);
    console.log(`   Message ID: ${messageId}`);
    console.log(`   Success: ${ackData.success}`);
    console.log(`   Timestamp: ${new Date(ackData.timestamp).toLocaleString()}`);

    // 从队列中删除消息（或标记为ACKED）
    const removed = this.messageQueue.markAsAcked(messageId);

    if (removed) {
      // 解析对应的Promise
      const pending = this._pendingAcks.get(messageId);
      if (pending) {
        clearTimeout(pending.timer);

        if (ackData.success) {
          pending.resolve({ message_id: messageId, acked_at: Date.now() });
        } else {
          pending.reject(new Error('服务端处理失败: ' + (ackData.error || '未知错误')));
        }

        this._pendingAcks.delete(messageId);
      }
    } else {
      console.warn(`[ReliableWS] ⚠️  ACK对应的消息不存在（可能已被清理）: ${messageId}`);
    }
  }

  /**
   * _startMessageSyncLoop - 启动消息同步循环（重连后调用）
   */
  _startMessageSyncLoop() {
    if (!this.enablePersistence) {
      return;
    }

    console.log('[ReliableWS] 🔄 启动消息同步循环...');

    // 立即执行一次同步
    this._syncPendingMessages();

    // 定期检查是否有新消息需要发送
    this._syncTimer = setInterval(() => {
      if (this.state === 'CONNECTED') {
        this._syncPendingMessages();
      }
    }, 2000); // 每2秒检查一次
  }

  /**
   * _syncPendingMessages - 同步所有待发送的消息
   */
  _syncPendingMessages() {
    if (this.state !== 'CONNECTED') {
      return;
    }

    // 获取所有PENDING状态的消息
    const pendingMessages = this.messageQueue.getPendingMessages();

    if (pendingMessages.length === 0) {
      return; // 无待发消息
    }

    console.log(`\n[ReliableWS] 📤 开始同步待发消息...`);
    console.log(`   待发数量: ${pendingMessages.length}`);

    // 逐条发送（按时间顺序）
    for (const msg of pendingMessages) {
      // 检查是否已在等待ACK（避免重复发送）
      if (this._pendingAcks.has(msg.id)) {
        continue; // 已在等待ACK，跳过
      }

      // 发送并等待ACK
      this._sendMessageWithAck(msg.id);
    }

    console.log(`[ReliableWS] ✅ 同步完成`);
  }

  /**
   * getReliabilityStats - 获取可靠性统计信息
   */
  getReliabilityStats() {
    const baseStats = this.getStats();
    const queueStats = this.messageQueue ? this.messageQueue.getStats() : null;

    return {
      ...baseStats,
      persistence_enabled: this.enablePersistence,
      message_queue: queueStats,
      pending_acks: this._pendingAcks.size,
      storage_path: this.storagePath
    };
  }

  /**
   * destroy - 销毁实例
   */
  destroy() {
    // 停止同步定时器
    if (this._syncTimer) {
      clearInterval(this._syncTimer);
      this._syncTimer = null;
    }

    // 清除所有等待中的ACK
    for (const [messageId, pending] of this._pendingAcks) {
      clearTimeout(pending.timer);
      pending.reject(new Error('客户端销毁'));
    }
    this._pendingAcks.clear();

    // 销毁消息队列
    if (this.messageQueue) {
      this.messageQueue.destroy();
    }

    // 调用父类destroy
    super.destroy();

    console.log('[ReliableWS] 💀 可靠传输客户端已销毁');
  }
}

module.exports = ReliableWebSocketClient;
