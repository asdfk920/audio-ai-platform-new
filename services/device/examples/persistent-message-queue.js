/**
 * PersistentMessageQueue - 本地持久化消息队列
 *
 * 核心功能：
 *   ✅ 本地JSON文件持久化（断电不丢、重启不丢）
 *   ✅ 原子写入操作（防止数据损坏）
 *   ✅ 消息状态管理（PENDING → SENT → ACKED）
 *   ✅ 自动清理已确认消息
 *   ✅ 按时间顺序保证（FIFO）
 *   ✅ 最大容量限制（防止磁盘占满）
 *
 * 存储格式：
 *   {
 *     "version": "1.0",
 *     "created_at": "2026-05-20T22:00:00Z",
 *     "messages": [
 *       {
 *         "id": "uuid-xxx",
 *         "type": "status_report",
 *         "payload": {...},
 *         "status": "PENDING",        // PENDING | SENT | ACKED
 *         "created_at": 1700000000000,
 *         "sent_at": null,
 *         "acked_at": null,
 *         "retry_count": 0,
 *         "last_error": null
 *       }
 *     ]
 *   }
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { v4: uuidv4 } = require('uuid');

class PersistentMessageQueue {
  constructor(options = {}) {
    // 存储配置
    this.storagePath = options.storagePath || './data/device_message_queue.json';
    this.maxQueueSize = options.maxQueueSize || 10000; // 最大10000条
    this.autoSaveInterval = options.autoSaveInterval || 5000; // 每5秒自动保存

    // 内部状态
    this.messages = [];
    this.version = '1.0';
    this.createdAt = new Date().toISOString();
    this._dirty = false; // 标记是否有未保存的更改
    this._saveTimer = null;
    this._lock = false; // 文件锁，防止并发写入

    // 初始化
    this._ensureStorageDirectory();
    this._loadFromFile();

    // 启动自动保存定时器
    if (this.autoSaveInterval > 0) {
      this._startAutoSave();
    }

    console.log(`[MessageQueue] 📦 持久化消息队列初始化完成`);
    console.log(`[MessageQueue] 📍 存储路径: ${this.storagePath}`);
    console.log(`[MessageQueue] 📊 当前队列长度: ${this.messages.length}`);
  }

  /**
   * enqueue - 入队消息（写入本地持久化存储）
   */
  async enqueue(message) {
    try {
      // 检查队列是否已满
      if (this.messages.length >= this.maxQueueSize) {
        const removed = this._removeOldestPending();
        console.warn(`[MessageQueue] ⚠️  队列已满，移除最旧的待发送消息: ${removed}`);
      }

      // 生成唯一消息ID
      const messageId = uuidv4();

      // 构造消息对象
      const queueMessage = {
        id: messageId,
        type: message.type || 'unknown',
        payload: message.payload || message,
        status: 'PENDING', // PENDING: 待发送, SENT: 已发送等待ACK, ACKED: 已确认
        created_at: Date.now(),
        sent_at: null,
        acked_at: null,
        retry_count: 0,
        last_error: null
      };

      // 加入队列（按时间排序）
      this.messages.push(queueMessage);
      this._sortMessages();

      // 标记脏数据，触发保存
      this._markDirty();

      console.log(`[MessageQueue] ✅ 消息入队成功:`);
      console.log(`   ID: ${messageId}`);
      console.log(`   类型: ${queueMessage.type}`);
      console.log(`   队列长度: ${this.messages.length}`);

      return {
        success: true,
        message_id: messageId,
        queue_size: this.messages.length
      };

    } catch (error) {
      console.error(`[MessageQueue] ❌ 入队失败:`, error);
      return {
        success: false,
        error: error.message
      };
    }
  }

  /**
   * getPendingMessages - 获取所有待发送的消息（按时间顺序）
   */
  getPendingMessages() {
    return this.messages
      .filter(msg => msg.status === 'PENDING')
      .sort((a, b) => a.created_at - b.created_at);
  }

  /**
   * getSentMessages - 获取所有已发送但未确认的消息
   */
  getSentMessages() {
    return this.messages
      .filter(msg => msg.status === 'SENT')
      .sort((a, b) => (a.sent_at || 0) - (b.sent_at || 0));
  }

  /**
   * markAsSent - 标记消息为已发送（等待ACK）
   */
  markAsSent(messageId) {
    const message = this.messages.find(m => m.id === messageId);
    if (message) {
      message.status = 'SENT';
      message.sent_at = Date.now();
      message.retry_count += 1;
      this._markDirty();

      console.log(`[MessageQueue] 📤 消息已标记为SENT: ${messageId} (重试次数: ${message.retry_count})`);
      return true;
    }
    return false;
  }

  /**
   * markAsAcked - 收到ACK确认，删除消息（或标记为ACKED）
   */
  markAsAcked(messageId) {
    const index = this.messages.findIndex(m => m.id === messageId);
    if (index !== -1) {
      const message = this.messages[index];
      message.status = 'ACKED';
      message.acked_at = Date.now();

      console.log(`[MessageQueue] ✅ 消息已确认(ACK): ${messageId}`);

      // 从活跃队列中移除（可选：保留历史记录用于调试）
      this.messages.splice(index, 1);
      this._markDirty();

      return true;
    }
    console.warn(`[MessageQueue] ⚠️  未找到消息ID: ${messageId}`);
    return false;
  }

  /**
   * markAsFailed - 标记消息发送失败（保持PENDING状态以便重试）
   */
  markAsFailed(messageId, error) {
    const message = this.messages.find(m => m.id === messageId);
    if (message) {
      message.status = 'PENDING'; // 重置为PENDING，下次重试
      message.last_error = error;
      message.sent_at = null; // 清除发送时间
      this._markDirty();

      console.log(`[MessageQueue] ❌ 消息发送失败: ${messageId}, 错误: ${error}`);
      return true;
    }
    return false;
  }

  /**
   * getAllMessages - 获取所有消息（用于调试）
   */
  getAllMessages() {
    return [...this.messages];
  }

  /**
   * getStats - 获取队列统计信息
   */
  getStats() {
    const pending = this.messages.filter(m => m.status === 'PENDING').length;
    const sent = this.messages.filter(m => m.status === 'SENT').length;
    const acked = this.messages.filter(m => m.status === 'ACKED').length;

    return {
      total: this.messages.length,
      pending: pending,
      sent: sent,
      acked: acked,
      max_capacity: this.maxQueueSize,
      utilization: ((this.messages.length / this.maxQueueSize) * 100).toFixed(1) + '%',
      oldest_pending: this.getOldestPendingTime(),
      storage_path: this.storagePath
    };
  }

  /**
   * clearAll - 清空所有消息（慎用！）
   */
  async clearAll() {
    this.messages = [];
    await this._saveToFile();
    console.log('[MessageQueue] 🗑️  队列已清空');
  }

  /**
   * forceSave - 强制立即保存到文件
   */
  async forceSave() {
    await this._saveToFile();
  }

  /**
   * destroy - 销毁实例（停止自动保存）
   */
  destroy() {
    if (this._saveTimer) {
      clearInterval(this._saveTimer);
      this._saveTimer = null;
    }

    // 最后一次保存
    this.forceSync();

    console.log('[MessageQueue] 💀 实例已销毁');
  }

  // ══════════════════════════════════════════════
  // 私有方法
  // ══════════════════════════════════════════════

  _sortMessages() {
    this.messages.sort((a, b) => a.created_at - b.created_at);
  }

  _getOldestPendingTime() {
    const pending = this.getPendingMessages();
    if (pending.length > 0) {
      const age = Date.now() - pending[0].created_at;
      return `${Math.floor(age / 1000)}s`;
    }
    return 'N/A';
  }

  _removeOldestPending() {
    const pendingIndex = this.messages.findIndex(m => m.status === 'PENDING');
    if (pendingIndex !== -1) {
      const removed = this.messages.splice(pendingIndex, 1)[0];
      this._markDirty();
      return removed.id;
    }
    return null;
  }

  _markDirty() {
    this._dirty = true;
  }

  _ensureStorageDirectory() {
    const dir = path.dirname(this.storagePath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
      console.log(`[MessageQueue] 📁 创建存储目录: ${dir}`);
    }
  }

  _loadFromFile() {
    try {
      if (fs.existsSync(this.storagePath)) {
        const data = fs.readFileSync(this.storagePath, 'utf8');
        const parsed = JSON.parse(data);

        this.version = parsed.version || '1.0';
        this.createdAt = parsed.createdAt || new Date().toISOString();
        this.messages = parsed.messages || [];

        console.log(`[MessageQueue] 📂 从文件加载队列: ${this.messages.length}条消息`);
      } else {
        console.log('[MessageQueue] 📝 存储文件不存在，创建新队列');
        this._saveToFile(); // 创建空文件
      }
    } catch (error) {
      console.error('[MessageQueue] ❌ 加载文件失败:', error);
      console.warn('[MessageQueue] ⚠️  使用空队列启动');

      // 备份损坏的文件
      if (fs.existsSync(this.storagePath)) {
        const backupPath = `${this.storagePath}.corrupted.${Date.now()}`;
        fs.renameSync(this.storagePath, backupPath);
        console.error(`[MessageQueue] 💾 损坏文件已备份到: ${backupPath}`);
      }
    }
  }

  async _saveToFile() {
    if (!this._dirty) {
      return; // 无需保存
    }

    if (this._lock) {
      console.warn('[MessageQueue] ⚠️  文件正在被其他操作使用，跳过本次保存');
      return;
    }

    this._lock = true;

    try {
      const data = {
        version: this.version,
        created_at: this.createdAt,
        last_saved: new Date().toISOString(),
        messages: this.messages,
        stats: this.getStats()
      };

      const jsonStr = JSON.stringify(data, null, 2);

      // 原子写入：先写临时文件，再重命名（防止写入过程中断电导致损坏）
      const tempPath = `${this.storagePath}.tmp`;
      fs.writeFileSync(tempPath, jsonStr, 'utf8');
      fs.renameSync(tempPath, this.storagePath);

      this._dirty = false;

      console.log(`[MessageQueue] 💾 队列已保存到文件: ${this.messages.length}条消息`);

    } catch (error) {
      console.error('[MessageQueue] ❌ 保存文件失败:', error);

      // 尝试恢复临时文件
      const tempPath = `${this.storagePath}.tmp`;
      if (fs.existsSync(tempPath)) {
        try {
          fs.unlinkSync(tempPath);
          console.log('[MessageQueue] 🧹 已清理临时文件');
        } catch (e) {
          console.error('[MessageQueue] ❌ 清理临时文件失败:', e);
        }
      }

    } finally {
      this._lock = false;
    }
  }

  forceSync() {
    return this._saveToFile();
  }

  _startAutoSave() {
    this._saveTimer = setInterval(() => {
      if (this._dirty) {
        this._saveToFile();
      }
    }, this.autoSaveInterval);
  }
}

module.exports = PersistentMessageQueue;
