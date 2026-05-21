/**
 * DeviceWebSocketClient - 设备WebSocket自动重连客户端
 *
 * 核心功能：
 *   ✅ 自动重连（指数退避策略，防止频繁重连冲垮服务）
 *   ✅ 网络检测（断网时不盲目重连，节省资源）
 *   ✅ 消息缓存（断连期间消息不丢失，重连后自动同步）
 *   ✅ Token复用（重连时使用已有Token，无需重新认证）
 *   ✅ 心跳保活（54秒间隔，自动检测连接健康度）
 *   ✅ 不限次数重连（只要设备在线就一直尝试）
 *
 * 使用示例：
 *   const client = new DeviceWebSocketClient({
 *     url: 'ws://localhost:8002/ws/device',
 *     token: 'your_jwt_token_here',
 *     deviceId: '1234567890'
 *   });
 *
 *   client.connect();
 *
 *   // 发送消息（会自动缓存或实时发送）
 *   client.send({ type: 'status', data: { battery: 85 } });
 */

class DeviceWebSocketClient {
  constructor(options = {}) {
    // 基础配置
    this.url = options.url || 'ws://localhost:8002/ws/device';
    this.token = options.token || '';
    this.deviceId = options.deviceId || '';

    // 重连配置
    this.reconnectEnabled = options.reconnectEnabled !== false; // 默认开启
    this.maxReconnectAttempts = options.maxReconnectAttempts || Infinity; // 不限次数
    this.initialReconnectDelay = options.initialReconnectDelay || 1000; // 初始延迟 1秒
    this.maxReconnectDelay = options.maxReconnectDelay || 30000; // 最大延迟 30秒
    this.reconnectDecay = options.reconnectDecay || 2; // 指数退避因子 (2^n)
    this.reconnectJitter = options.reconnectJitter || 0.3; // 抖动系数 (0-1)，避免同时重连

    // 心跳配置（修复假死问题的核心参数）
    this.pingInterval = options.pingInterval || 15000; // 15秒发送一次ping（原来54秒太长！）
    this.pongTimeout = options.pongTimeout || 30000;   // 30秒等待pong超时（原来60秒太长！）

    // 网络检测配置
    this.networkCheckEnabled = options.networkCheckEnabled !== false;
    this.networkCheckInterval = options.networkCheckInterval || 5000; // 每5秒检测一次网络
    this.networkCheckUrl = options.networkCheckUrl || 'http://localhost:8002/health';

    // 消息队列配置
    this.messageQueueMaxSize = options.messageQueueMaxSize || 1000; // 最大缓存1000条消息

    // 内部状态
    this.ws = null;
    this.state = 'DISCONNECTED'; // DISCONNECTED | CONNECTING | CONNECTED | RECONNECTING
    this.manualClose = false; // 是否手动关闭（手动关闭不触发重连）
    this._pendingReboot = false; // 🔴 新增：是否等待重启完成（用于延迟重连）

    // 重连相关变量
    this.reconnectAttempts = 0;
    this.currentReconnectDelay = this.initialReconnectDelay;
    this.reconnectTimer = null;

    // 心跳相关变量
    this.pingTimer = null;
    this.pongTimer = null;

    // 网络检测相关变量
    this.networkAvailable = true;
    this.networkCheckTimer = null;

    // 消息队列
    this.messageQueue = [];

    // 绑定回调函数
    this.onOpen = options.onOpen || (() => {});
    this.onMessage = options.onMessage || (() => {});
    this.onClose = options.onClose || (() => {});
    this.onError = options.onError || (() => {});
    this.onReconnecting = options.onReconnecting || (() => {});
    this.onReconnected = options.onReconnected || (() => {});
    this.onNetworkChange = options.onNetworkChange || (() => {});

    // 新增：连接健康度监控（用于检测假死）
    this.lastMessageTime = 0; // 最后一次收到消息的时间
    this.connectionHealthCheckTimer = null;

    // 新增：浏览器网络状态事件监听（修复手动断网无法感知的bug）
    if (typeof window !== 'undefined') {
      this._bindBrowserNetworkEvents();
    }

    console.log(`[WS Client] 📱 客户端初始化完成`);
    console.log(`[WS Client] 🔗 目标地址: ${this.url}`);
    console.log(`[WS Client] ⚙️  重连配置: enabled=${this.reconnectEnabled}, maxDelay=${this.maxReconnectDelay}ms`);
    console.log(`[WS Client] 💓 心跳配置: ping=${this.pingInterval/1000}s, pongTimeout=${this.pongTimeout/1000}s (修复假死)`);
  }

  /**
   * connect - 建立WebSocket连接
   */
  connect() {
    if (this.state === 'CONNECTED' || this.state === 'CONNECTING') {
      console.warn('[WS Client] ⚠️  已在连接状态，忽略重复连接请求');
      return;
    }

    this.manualClose = false;
    this._connect();
  }

  /**
   * _connect - 内部连接方法
   */
  _connect() {
    try {
      this.state = 'CONNECTING';

      const wsUrl = `${this.url}?token=${encodeURIComponent(this.token)}`;
      console.log(`[WS Client] 🔗 正在建立连接... (${wsUrl.substring(0, 50)}...)`);

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = (event) => this._handleOpen(event);
      this.ws.onmessage = (event) => this._handleMessage(event);
      this.ws.onclose = (event) => this._handleClose(event);
      this.ws.onerror = (event) => this._handleError(event);

    } catch (error) {
      console.error('[WS Client] ❌ 连接异常:', error);
      this.state = 'DISCONNECTED';
      this._scheduleReconnect();
    }
  }

  /**
   * _handleOpen - 处理连接打开事件
   */
  _handleOpen(event) {
    console.log('✅ [WS Client] WebSocket连接已建立');
    console.log(`✅ [WS Client] 设备ID: ${this.deviceId}`);
    console.log(`✅ [WS Client] 协议: ${this.ws.protocol}`);
    console.log('====================================');

    this.state = 'CONNECTED';
    this.reconnectAttempts = 0;
    this.currentReconnectDelay = this.initialReconnectDelay;
    this.lastMessageTime = Date.now(); // 记录连接建立时间

    // 启动多重保活机制（修复假死问题）
    this._startHeartbeat();
    this._startConnectionHealthCheck(); // 新增：连接健康度监控
    this._flushMessageQueue();

    this.onOpen(event);

    if (this.reconnectAttempts > 0) {
      console.log(`🔄 [WS Client] 重连成功! (第${this.reconnectAttempts}次尝试后恢复)`);
      this.onReconnected({ attempts: this.reconnectAttempts, timestamp: Date.now() });
    }
  }

  /**
   * _handleMessage - 处理接收到的消息
   */
  _handleMessage(event) {
    try {
      const data = JSON.parse(event.data);
      console.log(`📥 [WS Client] 收到消息:`, data.type || 'unknown');

      // 更新最后消息接收时间（用于健康度检测）
      this.lastMessageTime = Date.now();

      if (data.type === 'pong') {
        this._handlePong();
        return;
      }

      // 🔴 关键新增：检测reboot指令，主动断开连接
      if (this._handleRebootCommand(data)) {
        return;  // 已处理reboot指令，不继续传递
      }

      this.onMessage(data, event);
    } catch (error) {
      console.warn('[WS Client] ⚠️  解析消息失败:', error);
      // 即使解析失败，也更新时间（说明网络还是通的）
      this.lastMessageTime = Date.now();
      this.onMessage(event.data, event); // 原始数据透传
    }
  }

  /**
   * _handleRebootCommand - 处理服务端下发的重启指令
   *
   * 作用：
   *   当收到服务端的 {cmd: "reboot"} 指令时：
   *   1. 立即返回ACK确认给服务端
   *   2. 主动断开WebSocket连接
   *   3. 等待设备实际重启完成（延迟30秒）
   *   4. 延迟结束后自动重连
   *
   * 返回 boolean:
   *   true  - 是reboot指令且已处理
   *   false - 不是reboot指令，需要继续处理
   */
  _handleRebootCommand(data) {
    // 检查是否是cmd类型的消息且cmd为reboot
    if (data.type === 'cmd' && data.cmd === 'reboot') {
      console.log('\n🔄🔄🔄 [WS Client] 收到重启指令 🔄🔄🔄');
      console.log(`   指令ID: ${data.instruction_id || 'N/A'}`);
      console.log(`   请求ID: ${data.request_id || 'N/A'}`);
      console.log(`   时间戳: ${data.timestamp || new Date().toISOString()}`);
      console.log('   处理方案: 确认收到 → 断开连接 → 等待重启 → 自动重连\n');

      // 1. 发送ACK确认给服务端（同步发送）
      this._sendRebootAck(data);

      // 2. 延迟一小段时间后主动断开（确保ACK已发送）
      setTimeout(() => {
        console.log('[WS Client] 🔴 正在断开连接以执行重启...');

        // 标记为非手动关闭（允许自动重连）
        this.manualClose = false;

        // 设置特殊标记：这是重启导致的断开，需要延迟重连
        this._pendingReboot = true;

        // 主动关闭连接
        if (this.ws) {
          this.ws.close(4006, 'Device rebooting as requested');
        }
      }, 200);  // 200ms延迟，足够ACK发送完成

      return true;  // 表示已处理
    }

    return false;  // 不是reboot指令
  }

  /**
   * _sendRebootAck - 发送重启确认ACK给服务端
   *
   * 消息格式：
   * {
   *   "type": "cmd_response",
   *   "cmd": "reboot",
   *   "sn": "设备序列号",
   *   "request_id": "原始请求ID",
   *   "status": "success",
   *   "message": "设备即将重启"
   * }
   */
  _sendRebootAck(rebootCmd) {
    try {
      const ackMessage = {
        type: 'cmd_response',
        cmd: 'reboot',
        sn: rebootCmd.sn,
        request_id: rebootCmd.request_id,
        status: 'success',
        message: '设备即将重启',
        timestamp: new Date().toISOString()
      };

      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(ackMessage));
        console.log('[WS Client] ✅ 重启ACK已发送:', JSON.stringify(ackMessage));
      } else {
        console.warn('[WS Client] ⚠️  无法发送ACK：连接已关闭');
      }
    } catch (error) {
      console.error('[WS Client] ❌ 发送ACK失败:', error);
    }
  }

  /**
   * _handleClose - 处理连接关闭事件（核心重连逻辑）
   * 修复：增强状态更新，确保UI能正确感知断连
   */
  _handleClose(event) {
    console.log('\n🔴🔴🔴 [WS Client] WebSocket连接已关闭 🔴🔴🔴');
    console.log(`   Code: ${event.code}`);
    console.log(`   Reason: ${event.reason || '无原因'}`);
    console.log(`   WasClean: ${event.wasClean}`);
    console.log(`   时间: ${new Date().toLocaleString()}`);

    // 强制更新状态为DISCONNECTED（修复假死bug的关键！）
    const wasDisconnected = (this.state === 'DISCONNECTED');
    this.state = 'DISCONNECTED';

    // 停止所有保活机制
    this._stopHeartbeat();
    this._stopConnectionHealthCheck(); // 新增：停止健康度监控

    // 🔴 关键修复：只在非强制更新时才触发onClose回调
    // （避免与 _forceUpdateUIState 的回调重复）
    if (event.code !== 0) { // code=0 是强制状态更新的特殊码
      this.onClose(event);
    }

    // 根据原因决定是否重连
    if (!this.manualClose && this.reconnectEnabled) {
      // 🔴 关键新增：检查是否是重启导致的断开
      if (this._pendingReboot && event.code === 4006) {
        console.log('\n⏳ [WS Client] 检测到重启断开，等待设备重启完成...');
        console.log('   延迟时间: 30秒（给设备足够的时间完成重启）');

        // 重置标记（避免影响后续重连）
        this._pendingReboot = false;

        // 延迟30秒后重连（设备重启通常需要10-20秒，留余量）
        setTimeout(() => {
          console.log('[WS Client] ⏰ 等待结束，开始重新连接...');
          console.log('[WS Client] 🔄 设备应该已经重启完成，正在重连...\n');
          this._startReconnection();
        }, 30000);  // 30秒延迟

      } else {
        // 普通断开：正常快速重连
        console.log(`\n🔄 [WS Client] 准备启动自动重连...`);

        // 🔴 关键修复：延迟启动重连，让"已断开"状态显示一会儿
        setTimeout(() => {
          this._startReconnection();
        }, wasDisconnected ? 0 : 100); // 如果已经显示过断开，立即重连；否则等100ms
      }
    } else if (this.manualClose) {
      console.log('[WS Client] ℹ️  手动关闭，不触发重连');
    }
  }

  /**
   * _handleError - 处理错误事件
   */
  _handleError(event) {
    console.error('❌ [WS Client] WebSocket错误:', event);
    this.onError(event);
  }

  /**
   * send - 发送消息（智能路由：在线直接发送，离线入队缓存）
   */
  send(data) {
    const message = typeof data === 'string' ? data : JSON.stringify(data);

    if (this.state === 'CONNECTED' && this.ws.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(message);
        console.log(`📤 [WS Client] 消息已发送 (${message.length} bytes)`);
        return true;
      } catch (error) {
        console.error('[WS Client] ❌ 发送失败:', error);
        return this._enqueueMessage(message);
      }
    } else {
      console.log(`💾 [WS Client] 当前状态=${this.state}，消息已加入缓存队列`);
      return this._enqueueMessage(message);
    }
  }

  /**
   * close - 手动关闭连接（不触发自动重连）
   */
  close(code = 1000, reason = 'Normal closure') {
    console.log('[WS Client] 👋 正在关闭连接...');
    this.manualClose = true;
    this.reconnectEnabled = false;

    // 停止所有保活机制
    this._stopHeartbeat();
    this._stopConnectionHealthCheck(); // 新增
    this._stopReconnect();
    this._stopNetworkCheck();

    if (this.ws) {
      this.ws.close(code, reason);
    }

    // 强制更新状态
    this.state = 'DISCONNECTED';
    console.log('[WS Client] ✅ 连接已手动关闭');
  }

  /**
   * destroy - 销毁实例（清理所有资源）
   */
  destroy() {
    this.close();
    this.messageQueue = [];
    
    // 解绑浏览器事件（防止内存泄漏）
    if (typeof window !== 'undefined') {
      window.removeEventListener('offline', this._handleOffline);
      window.removeEventListener('online', this._handleOnline);
    }
    
    this.onOpen = null;
    this.onMessage = null;
    this.onClose = null;
    this.onError = null;
    console.log('[WS Client] 💀 实例已销毁');
  }

  // ══════════════════════════════════════════════
  // 重连管理模块
  // ══════════════════════════════════════════════

  /**
   * _startReconnection - 启动重连流程
   */
  _startReconnection() {
    if (!this.reconnectEnabled) {
      console.log('[WS Client] ℹ️  自动重连已禁用');
      return;
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(`[WS Client] ❌ 已达到最大重连次数 (${this.maxReconnectAttempts})，停止重连`);
      return;
    }

    this.state = 'RECONNECTING';
    this.reconnectAttempts++;

    const delay = this._calculateReconnectDelay();

    console.log('');
    console.log('====================================');
    console.log(`🔄 [WS Client] 准备重连... (第${this.reconnectAttempts}次)`);
    console.log(`   当前延迟: ${(delay / 1000).toFixed(1)}s`);
    console.log(`   下次延迟: ${(Math.min(delay * this.reconnectDecay, this.maxReconnectDelay) / 1000).toFixed(1)}s`);
    console.log(`   缓存消息数: ${this.messageQueue.length}`);
    console.log('====================================');

    this.onReconnecting({
      attempt: this.reconnectAttempts,
      delay: delay,
      queueSize: this.messageQueue.length
    });

    this.reconnectTimer = setTimeout(() => {
      this._attemptReconnect();
    }, delay);
  }

  /**
   * _attemptReconnect - 执行单次重连尝试（包含网络检测）
   */
  async _attemptReconnect() {
    console.log(`🌐 [WS Client] 第${this.reconnectAttempts}次重连尝试...`);

    if (this.networkCheckEnabled) {
      const isOnline = await this._checkNetworkAvailability();

      if (!isOnline) {
        console.log(`⏳ [WS Client] 网络不可用，等待${this.networkCheckInterval / 1000}s后重新检测...`);
        setTimeout(() => this._startReconnection(), this.networkCheckInterval);
        return;
      }

      console.log('✅ [WS Client] 网络可用，继续重连...');
    }

    this._connect();
  }

  /**
   * _calculateReconnectDelay - 计算重连延迟（指数退避+随机抖动）
   */
  _calculateReconnectDelay() {
    let delay = this.initialReconnectDelay * Math.pow(this.reconnectDecay, this.reconnectAttempts - 1);
    delay = Math.min(delay, this.maxReconnectDelay);

    if (this.reconnectJitter > 0) {
      const jitter = delay * this.reconnectJitter * (Math.random() * 2 - 1);
      delay += jitter;
    }

    return Math.round(delay);
  }

  /**
   * _scheduleReconnect - 调度下一次重连
   */
  _scheduleReconnect() {
    if (!this.reconnectEnabled || this.manualClose) {
      return;
    }

    this._startReconnection();
  }

  /**
   * _stopReconnect - 停止重连定时器
   */
  _stopReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  // ══════════════════════════════════════════════
  // 网络检测模块
  // ══════════════════════════════════════════════

  /**
   * _checkNetworkAvailability - 检查网络是否可用
   */
  async _checkNetworkAvailability() {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3000);

      const response = await fetch(this.networkCheckUrl, {
        method: 'GET',
        mode: 'no-cors',
        signal: controller.signal
      });

      clearTimeout(timeoutId);

      const wasAvailable = this.networkAvailable;
      this.networkAvailable = true;

      if (!wasAvailable) {
        console.log('🌐 [WS Client] 网络已恢复!');
        this.onNetworkChange({ online: true });
      }

      return true;
    } catch (error) {
      const wasAvailable = this.networkAvailable;
      this.networkAvailable = false;

      if (wasAvailable) {
        console.log('⚠️ [WS Client] 网络不可用!');
        this.onNetworkChange({ online: false });
      }

      return false;
    }
  }

  /**
   * _startNetworkCheck - 启动定期网络检测
   */
  _startNetworkCheck() {
    if (!this.networkCheckEnabled) return;

    this.networkCheckTimer = setInterval(async () => {
      await this._checkNetworkAvailability();
    }, this.networkCheckInterval);
  }

  /**
   * _stopNetworkCheck - 停止网络检测
   */
  _stopNetworkCheck() {
    if (this.networkCheckTimer) {
      clearInterval(this.networkCheckTimer);
      this.networkCheckTimer = null;
    }
  }

  // ══════════════════════════════════════════════
  // 心跳保活模块
  // ══════════════════════════════════════════════

  /**
   * _startHeartbeat - 启动心跳保活（修复假死问题的核心）
   *
   * 工作原理：
   *   1. 每15秒发送一次ping
   *   2. 发送后启动30秒超时定时器
   *   3. 如果30秒内没收到pong → 判定连接假死 → 主动关闭 → 触发重连
   */
  _startHeartbeat() {
    this._stopHeartbeat();

    console.log(`💓 [WS Client] 启动心跳保活 (间隔: ${this.pingInterval/1000}s, 超时: ${this.pongTimeout/1000}s)`);

    this.pingTimer = setInterval(() => {
      if (this.state === 'CONNECTED' && this.ws && this.ws.readyState === WebSocket.OPEN) {
        // 发送ping包
        const pingData = JSON.stringify({ type: 'ping', timestamp: Date.now() });
        this.ws.send(pingData);
        console.log(`💓 [WS Client] Ping发送 (${new Date().toLocaleTimeString()})`);

        // 启动pong超时检测（关键！）
        this.pongTimer = setTimeout(() => {
          console.error('\n⚠️⚠️⚠️ [WS Client] Pong超时警告 ⚠️⚠️⚠️');
          console.error(`   已等待 ${this.pongTimeout/1000} 秒未收到服务端响应`);
          console.error(`   最后消息时间: ${this.lastMessageTime ? new Date(this.lastMessageTime).toLocaleString() : '无'}`);
          console.error(`   判定结果: 连接可能已假死（网络静默断开）`);
          console.error('   处理方案: 主动关闭连接并触发重连\n');

          // 🔴 关键修复：先更新UI状态为"已断开"，再关闭连接
          this._forceUpdateUIState('DISCONNECTED', 'Pong超时');

          // 短暂延迟后关闭连接（让UI有时间渲染）
          setTimeout(() => {
            // 主动关闭假死的连接
            if (this.ws) {
              this.ws.close(4001, `Pong timeout after ${this.pongTimeout/1000}s`);
            }
          }, 50); // 50ms延迟，足够DOM更新
        }, this.pongTimeout);
      }
    }, this.pingInterval);
  }

  /**
   * _stopHeartbeat - 停止心跳
   */
  _stopHeartbeat() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }

    if (this.pongTimer) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }

    console.log('💓 [WS Client] 心跳已停止');
  }

  /**
   * _handlePong - 处理Pong响应
   */
  _handlePong() {
    if (this.pongTimer) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
    console.log(`💚 [WS Client] Pong收到 (${new Date().toLocaleTimeString()})`);
  }

  // ══════════════════════════════════════════════
  // 连接健康度监控（新增：双重保障机制）
  // ══════════════════════════════════════════════

  /**
   * _startConnectionHealthCheck - 启动连接健康度监控
   *
   * 作用：作为心跳机制的补充，定期检查是否长时间未收到任何消息
   * 场景：某些情况下心跳包能发但pong丢失，或者服务端不回复pong
   *
   * 检测逻辑：
   *   每10秒检查一次 lastMessageTime
   *   如果距离上次收到消息超过 45秒 → 判定连接异常 → 主动关闭
   */
  _startConnectionHealthCheck() {
    this._stopConnectionHealthCheck();

    const healthCheckInterval = 10000; // 每10秒检查一次
    const maxSilentDuration = 45000;   // 最大静默时间45秒（应该至少能收到1次ping的响应）

    console.log(`🏥 [WS Client] 启动连接健康度监控 (检查间隔: ${healthCheckInterval/1000}s, 最大静默: ${maxSilentDuration/1000}s)`);

    this.connectionHealthCheckTimer = setInterval(() => {
      if (this.state !== 'CONNECTED') {
        return; // 非连接状态，跳过检查
      }

      const timeSinceLastMessage = Date.now() - this.lastMessageTime;
      const silentSeconds = Math.floor(timeSinceLastMessage / 1000);

      if (timeSinceLastMessage > maxSilentDuration) {
        console.error('\n⛑️⛑️⛑️ [WS Client] 连接健康度检测失败 ⛑️⛑️⛑️');
        console.error(`   已静默 ${silentSeconds} 秒未收到任何消息`);
        console.error(`   最后消息时间: ${new Date(this.lastMessageTime).toLocaleString()}`);
        console.error(`   当前时间: ${new Date().toLocaleString()}`);
        console.error(`   判定结果: 连接可能已假死或网络中断`);
        console.error(`   处理方案: 立即主动关闭并触发重连\n`);

        // 🔴 关键修复：先更新UI状态为"已断开"
        this._forceUpdateUIState('DISCONNECTED', '连接不健康');

        // 延迟后关闭连接
        setTimeout(() => {
          if (this.ws) {
            this.ws.close(4002, `Connection unhealthy: silent for ${silentSeconds}s`);
          }
        }, 50);
      } else if (silentSeconds > 30) {
        // 接近阈值时发出警告
        console.warn(`⚠️ [WS Client] 连接静默警告: 已 ${silentSeconds} 秒未收到消息 (阈值: ${maxSilentDuration/1000}s)`);
      }
    }, healthCheckInterval);
  }

  /**
   * _stopConnectionHealthCheck - 停止连接健康度监控
   */
  _stopConnectionHealthCheck() {
    if (this.connectionHealthCheckTimer) {
      clearInterval(this.connectionHealthCheckTimer);
      this.connectionHealthCheckTimer = null;
    }
  }

  // ══════════════════════════════════════════════
  // 消息队列模块（离线缓存）
  // ══════════════════════════════════════════════

  /**
   * _enqueueMessage - 将消息加入发送队列
   */
  _enqueueMessage(message) {
    if (this.messageQueue.length >= this.messageQueueMaxSize) {
      console.warn('[WS Client] ⚠️  消息队列已满，丢弃最旧的消息');
      this.messageQueue.shift(); // FIFO：移除最旧的消息
    }

    this.messageQueue.push({
      message: message,
      timestamp: Date.now(),
      retryCount: 0
    });

    console.log(`💾 [WS Client] 消息已缓存 (队列长度: ${this.messageQueue.length})`);
    return false;
  }

  /**
   * _flushMessageQueue - 清空消息队列（重连成功后调用）
   */
  _flushMessageQueue() {
    if (this.messageQueue.length === 0) {
      return;
    }

    console.log(`\n📤 [WS Client] 开始同步离线期间的缓存消息...`);
    console.log(`   缓存消息数: ${this.messageQueue.length}`);

    let successCount = 0;
    let failCount = 0;

    while (this.messageQueue.length > 0) {
      const item = this.messageQueue.shift();

      try {
        if (this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(item.message);
          successCount++;
        } else {
          failCount++;
          break;
        }
      } catch (error) {
        console.error('[WS Client] ❌ 同步消息失败:', error);
        failCount++;
        break;
      }
    }

    console.log(`✅ [WS Client] 同步完成: 成功=${successCount}, 失败=${failCount}, 剩余=${this.messageQueue.length}`);
  }

  // ══════════════════════════════════════════════
  // 工具方法
  // ══════════════════════════════════════════════

  /**
   * getState - 获取当前连接状态
   */
  getState() {
    return {
      state: this.state,
      reconnectAttempts: this.reconnectAttempts,
      currentReconnectDelay: this.currentReconnectDelay,
      messageQueueSize: this.messageQueue.length,
      networkAvailable: this.networkAvailable,
      connected: this.state === 'CONNECTED'
    };
  }

  /**
   * updateToken - 更新认证Token（用于Token刷新场景）
   */
  updateToken(newToken) {
    console.log('[WS Client] 🔑 Token已更新');
    this.token = newToken;
  }

  /**
   * getStats - 获取统计信息
   */
  getStats() {
    return {
      reconnectAttempts: this.reconnectAttempts,
      messagesQueued: this.messageQueue.length,
      networkChecksPerformed: this.networkAvailable ? 'online' : 'offline',
      uptime: this.state === 'CONNECTED' ? 'connected' : 'disconnected',
      lastMessageTime: this.lastMessageTime ? new Date(this.lastMessageTime).toLocaleString() : 'N/A'
    };
  }

  // ══════════════════════════════════════════════
  // 浏览器网络事件监听（新增：修复手动断网bug）
  // ══════════════════════════════════════════════

  /**
   * _forceUpdateUIState - 强制更新UI状态（核心修复方法！）
   *
   * 作用：
   *   在主动关闭连接之前，先强制同步更新UI状态为"已断开"
   *   确保用户能看到完整的状态变化过程：已连接 → 已断开 → 重连中 → 已连接
   *
   * 使用场景：
   *   - 心跳超时主动关闭前
   *   - 健康度检测失败主动关闭前
   *   - 浏览器离线事件触发时
   *
   * @param {string} newState - 新状态 ('DISCONNECTED' | 'RECONNECTING' | 'CONNECTED')
   * @param {string} reason - 变化原因（用于日志）
   */
  _forceUpdateUIState(newState, reason = '') {
    const oldState = this.state;

    // 只在状态真正改变时才更新
    if (oldState === newState) {
      return;
    }

    console.log(`\n🎨 [WS Client] 强制UI状态更新:`);
    console.log(`   ${oldState} → ${newState}`);
    console.log(`   原因: ${reason}`);
    console.log(`   时间: ${new Date().toLocaleString()}\n`);

    // 立即更新内部状态
    this.state = newState;

    // 根据新状态触发对应的回调函数（确保UI同步更新）
    switch (newState) {
      case 'DISCONNECTED':
        // 停止所有保活机制
        this._stopHeartbeat();
        this._stopConnectionHealthCheck();

        // 触发onClose回调（让UI显示"已断开"）
        this.onClose({
          code: 0, // 特殊码，表示强制状态更新
          reason: `Force update: ${reason}`,
          wasClean: false
        });
        break;

      case 'RECONNECTING':
        // 触发onReconnecting回调（让UI显示"重连中"）
        this.onReconnecting({
          attempt: this.reconnectAttempts + 1,
          delay: this.currentReconnectDelay || this.initialReconnectDelay,
          queueSize: this.messageQueue.length,
          reason: reason
        });
        break;

      case 'CONNECTED':
        // 触发onOpen回调（让UI显示"已连接"）
        this.onOpen({});
        break;

      default:
        console.warn(`[WS Client] ⚠️ 未知状态: ${newState}`);
    }
  }

  /**
   * _bindBrowserNetworkEvents - 绑定浏览器online/offline事件
   *
   * 解决的问题：
   *   手动关闭WiFi/拔网线时，浏览器会触发offline事件
   *   但WebSocket不会立即感知到，导致UI还显示"已连接"
   *
   * 解决方案：
   *   监听浏览器事件 → 立即主动关闭WebSocket → 触发重连逻辑
   */
  _bindBrowserNetworkEvents() {
    // 监听离线事件（网络断开）
    window.addEventListener('offline', () => {
      console.error('\n🌐🌐🌐 [WS Client] 浏览器检测到网络离线 🌐🌐🌐');
      console.error('   原因: WiFi断开 / 网线拔除 / 系统网络设置变更');
      console.error('   时间:', new Date().toLocaleString());
      console.error('   处理方案: 立即主动关闭WebSocket连接\n');

      // 更新网络状态
      const wasOnline = this.networkAvailable;
      this.networkAvailable = false;

      if (wasOnline) {
        this.onNetworkChange({ online: false });
      }

      // 🔴 关键修复：先强制更新UI为"已断开"
      this._forceUpdateUIState('DISCONNECTED', '网络离线');

      // 如果当前是已连接状态，延迟后强制关闭
      if (this.state === 'CONNECTED' || this.state === 'CONNECTING') {
        console.log('🔴 [WS Client] 正在主动关闭WebSocket连接...');
        setTimeout(() => {
          if (this.ws) {
            this.ws.close(4003, 'Browser offline event detected');
          }
        }, 50);
      }
    });

    // 监听在线事件（网络恢复）
    window.addEventListener('online', () => {
      console.log('\n✅ [WS Client] 浏览器检测到网络恢复');
      console.log('   时间:', new Date().toLocaleString());

      // 更新网络状态
      const wasOffline = !this.networkAvailable;
      this.networkAvailable = true;

      if (wasOffline) {
        this.onNetworkChange({ online: true });
      }

      // 如果当前是断开状态，尝试重新连接
      if (this.state === 'DISCONNECTED' && this.reconnectEnabled && !this.manualClose) {
        console.log('🔄 [WS Client] 网络恢复，准备重新连接...');
        this._startReconnection();
      }
    });

    console.log('🌐 [WS Client] 已绑定浏览器网络状态事件监听 (online/offline)');
  }
}

// 导出（支持ES6 Module和CommonJS）
if (typeof module !== 'undefined' && module.exports) {
  module.exports = DeviceWebSocketClient;
}
