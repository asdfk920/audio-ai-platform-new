#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
DeviceWebSocketClient - 设备WebSocket自动重连客户端（Python版本）

核心功能：
  ✅ 自动重连（指数退避策略，防止频繁重连冲垮服务）
  ✅ 网络检测（断网时不盲目重连，节省资源）
  ✅ 消息缓存（断连期间消息不丢失，重连后自动同步）
  ✅ Token复用（重连时使用已有Token，无需重新认证）
  ✅ 心跳保活（54秒间隔，自动检测连接健康度）
  ✅ 不限次数重连（只要设备在线就一直尝试）

使用示例：
    from device_websocket_client import DeviceWebSocketClient

    client = DeviceWebSocketClient(
        url='ws://localhost:8002/ws/device',
        token='your_jwt_token_here',
        device_id='1234567890'
    )

    client.connect()

    # 发送消息（会自动缓存或实时发送）
    client.send({'type': 'status', 'data': {'battery': 85}})

    # 保持运行
    client.run_forever()
"""

import json
import time
import random
import threading
import queue
import logging
from enum import Enum, auto
from typing import Optional, Callable, Dict, Any, List
from dataclasses import dataclass

try:
    import websocket
except ImportError:
    print("请先安装websocket-client库: pip install websocket-client")
    exit(1)

try:
    import requests
except ImportError:
    print("请先安装requests库: pip install requests")
    exit(1)


# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)


class ConnectionState(Enum):
    """WebSocket连接状态枚举"""
    DISCONNECTED = auto()
    CONNECTING = auto()
    CONNECTED = auto()
    RECONNECTING = auto()


@dataclass
class QueuedMessage:
    """队列中的消息"""
    message: Any
    timestamp: float
    retry_count: int = 0


class DeviceWebSocketClient:
    """
    设备WebSocket自动重连客户端

    完整实现：
      - 指数退避重连策略 (1s → 2s → 4s → ... → 30s)
      - 网络可用性检测 (HTTP Ping)
      - 离线消息队列 (FIFO, 最大1000条)
      - 心跳保活机制 (54s间隔)
      - Token自动复用
    """

    def __init__(self, options: Optional[Dict[str, Any]] = None):
        options = options or {}

        # 基础配置
        self.url = options.get('url', 'ws://localhost:8002/ws/device')
        self.token = options.get('token', '')
        self.device_id = options.get('device_id', '')

        # 重连配置
        self.reconnect_enabled = options.get('reconnect_enabled', True)
        self.max_reconnect_attempts = options.get('max_reconnect_attempts', float('inf'))
        self.initial_reconnect_delay = options.get('initial_reconnect_delay', 1.0)  # 初始延迟 1秒
        self.max_reconnect_delay = options.get('max_reconnect_delay', 30.0)  # 最大延迟 30秒
        self.reconnect_decay = options.get('reconnect_decay', 2.0)  # 指数退避因子
        self.reconnect_jitter = options.get('reconnect_jitter', 0.3)  # 抖动系数

        # 心跳配置
        self.ping_interval = options.get('ping_interval', 54)  # 54秒
        self.pong_timeout = options.get('pong_timeout', 60)  # 60秒

        # 网络检测配置
        self.network_check_enabled = options.get('network_check_enabled', True)
        self.network_check_interval = options.get('network_check_interval', 5)  # 每5秒检测一次
        self.network_check_url = options.get('network_check_url', 'http://localhost:8002/health')

        # 消息队列配置
        self.message_queue_max_size = options.get('message_queue_max_size', 1000)

        # 内部状态
        self.ws: Optional[websocket.WebSocketApp] = None
        self.state = ConnectionState.DISCONNECTED
        self.manual_close = False
        self._running = False

        # 重连相关变量
        self._reconnect_attempts = 0
        self._current_delay = self.initial_reconnect_delay
        self._reconnect_timer: Optional[threading.Timer] = None

        # 心跳相关变量
        self._ping_timer: Optional[threading.Timer] = None
        self._pong_timer: Optional[threading.Timer] = None

        # 网络检测相关变量
        self._network_available = True
        self._network_check_timer: Optional[threading.Timer] = None

        # 消息队列 (线程安全)
        self._message_queue: queue.Queue = queue.Queue(maxsize=self.message_queue_max_size)

        # 锁（用于线程安全的状态访问）
        self._lock = threading.Lock()

        # 回调函数
        self.on_open: Callable = options.get('on_open', lambda *args, **kwargs: None)
        self.on_message: Callable = options.get('on_message', lambda *args, **kwargs: None)
        self.on_close: Callable = options.get('on_close', lambda *args, **kwargs: None)
        self.on_error: Callable = options.get('on_error', lambda *args, **kwargs: None)
        self.on_reconnecting: Callable = options.get('on_reconnecting', lambda *args, **kwargs: None)
        self.on_reconnected: Callable = options.get('on_reconnected', lambda *args, **kwargs: None)
        self.on_network_change: Callable = options.get('on_network_change', lambda *args, **kwargs: None)

        logger.info(f"[WS Client] 📱 客户端初始化完成")
        logger.info(f"[WS Client] 🔗 目标地址: {self.url}")
        logger.info(f"[WS Client] ⚙️  重连配置: enabled={self.reconnect_enabled}, maxDelay={self.max_reconnect_delay}s")

    def connect(self):
        """建立WebSocket连接"""
        with self._lock:
            if self.state in [ConnectionState.CONNECTED, ConnectionState.CONNECTING]:
                logger.warning("[WS Client] ⚠️  已在连接状态，忽略重复连接请求")
                return

            self.manual_close = False
            self._running = True
            self._connect()

    def _connect(self):
        """内部连接方法"""
        try:
            self.state = ConnectionState.CONNECTING

            ws_url = f"{self.url}?token={self.token}"
            logger.info(f"[WS Client] 🔗 正在建立连接... ({ws_url[:50]}...)")

            self.ws = websocket.WebSocketApp(
                ws_url,
                on_open=self._handle_open,
                on_message=self._handle_message,
                on_close=self._handle_close,
                on_error=self._handle_error
            )

            # 在后台线程中运行WebSocket
            ws_thread = threading.Thread(target=self.ws.run_forever, daemon=True)
            ws_thread.start()

        except Exception as error:
            logger.error(f"[WS Client] ❌ 连接异常: {error}")
            self.state = ConnectionState.DISCONNECTED
            self._schedule_reconnect()

    def _handle_open(self, ws):
        """处理连接打开事件"""
        logger.info("✅ [WS Client] WebSocket连接已建立")
        logger.info(f"✅ [WS Client] 设备ID: {self.device_id}")
        logger.info("=" * 50)

        with self._lock:
            self.state = ConnectionState.CONNECTED
            self._reconnect_attempts = 0
            self._current_delay = self.initial_reconnect_delay

        self._start_heartbeat()
        self._flush_message_queue()

        self.on_open(ws)

        if self._reconnect_attempts > 0:
            logger.info(f"🔄 [WS Client] 重连成功! (第{self._reconnect_attempts}次尝试后恢复)")
            self.on_reconnected(attempts=self._reconnect_attempts, timestamp=time.time())

    def _handle_message(self, ws, message):
        """处理接收到的消息"""
        try:
            data = json.loads(message)
            msg_type = data.get('type', 'unknown')
            logger.info(f"📥 [WS Client] 收到消息: {msg_type}")

            if msg_type == 'pong':
                self._handle_pong()
                return

            self.on_message(data, message)
        except json.JSONDecodeError as error:
            logger.warning(f"[WS Client] ⚠️  解析消息失败: {error}")
            self.on_message(message, raw_data=message)

    def _handle_close(self, ws, close_status_code, close_msg):
        """处理连接关闭事件（核心重连逻辑）"""
        logger.info(f"🔴 [WS Client] WebSocket连接关闭:")
        logger.info(f"   Code: {close_status_code}")
        logger.info(f"   Reason: {close_msg or '无原因'}")

        with self._lock:
            self.state = ConnectionState.DISCONNECTED

        self._stop_heartbeat()

        self.on_close(close_status_code, close_msg)

        if not self.manual_close and self.reconnect_enabled:
            self._start_reconnection()
        elif self.manual_close:
            logger.info("[WS Client] ℹ️  手动关闭，不触发重连")

    def _handle_error(self, ws, error):
        """处理错误事件"""
        logger.error(f"❌ [WS Client] WebSocket错误: {error}")
        self.on_error(error)

    def send(self, data) -> bool:
        """
        发送消息（智能路由：在线直接发送，离线入队缓存）

        返回值：
          True - 消息已发送或已入队
          False - 发送失败且入队也失败
        """
        message = json.dumps(data) if isinstance(data, (dict, list)) else str(data)

        with self._lock:
            is_connected = (self.state == ConnectionState.CONNECTED and
                          self.ws and self.ws.sock and self.ws.sock.connected)

        if is_connected:
            try:
                self.ws.send(message)
                logger.info(f"📤 [WS Client] 消息已发送 ({len(message)} bytes)")
                return True
            except Exception as error:
                logger.error(f"[WS Client] ❌ 发送失败: {error}")
                return self._enqueue_message(message)
        else:
            logger.info(f"💾 [WS Client] 当前状态={self.state.name}，消息已加入缓存队列")
            return self._enqueue_message(message)

    def close(self, code=1000, reason='Normal closure'):
        """手动关闭连接（不触发自动重连）"""
        logger.info("[WS Client] 👋 正在关闭连接...")

        with self._lock:
            self.manual_close = True
            self.reconnect_enabled = False
            self._running = False

        self._stop_heartbeat()
        self._stop_reconnect()
        self._stop_network_check()

        if self.ws:
            self.ws.close(code, reason)

        self.state = ConnectionState.DISCONNECTED

    def destroy(self):
        """销毁实例（清理所有资源）"""
        self.close()

        # 清空消息队列
        while not self._message_queue.empty():
            try:
                self._message_queue.get_nowait()
            except queue.Empty:
                break

        # 清理回调
        self.on_open = lambda *args, **kwargs: None
        self.on_message = lambda *args, **kwargs: None
        self.on_close = lambda *args, **kwargs: None
        self.on_error = lambda *args, **kwargs: None

        logger.info("[WS Client] 💀 实例已销毁")

    def run_forever(self):
        """保持客户端运行（阻塞主线程）"""
        logger.info("[WS Client] 🔄 客户端运行中... (按 Ctrl+C 停止)")
        while self._running:
            time.sleep(1)

    # ══════════════════════════════════════════════
    # 重连管理模块
    # ══════════════════════════════════════════════

    def _start_reconnection(self):
        """启动重连流程"""
        if not self.reconnect_enabled:
            logger.info("[WS Client] ℹ️  自动重连已禁用")
            return

        if self._reconnect_attempts >= self.max_reconnect_attempts:
            logger.error(f"[WS Client] ❌ 已达到最大重连次数 ({self.max_reconnect_attempts})，停止重连")
            return

        with self._lock:
            self.state = ConnectionState.RECONNECTING
            self._reconnect_attempts += 1

        delay = self._calculate_reconnect_delay()

        logger.info("")
        logger.info("=" * 50)
        logger.info(f"🔄 [WS Client] 准备重连... (第{self._reconnect_attempts}次)")
        logger.info(f"   当前延迟: {delay:.1f}s")
        logger.info(f"   下次延迟: {min(delay * self.reconnect_decay, self.max_reconnect_delay):.1f}s")
        logger.info(f"   缓存消息数: {self._message_queue.qsize()}")
        logger.info("=" * 50)

        self.on_reconnecting(
            attempt=self._reconnect_attempts,
            delay=delay,
            queue_size=self._message_queue.qsize()
        )

        self._reconnect_timer = threading.Timer(delay, self._attempt_reconnect)
        self._reconnect_timer.daemon = True
        self._reconnect_timer.start()

    async def _attempt_reconnect_async(self):
        """异步执行单次重连尝试（包含网络检测）"""
        await self._attempt_reconnect()

    def _attempt_reconnect(self):
        """执行单次重连尝试（包含网络检测）"""
        logger.info(f"🌐 [WS Client] 第{self._reconnect_attempts}次重连尝试...")

        if self.network_check_enabled:
            is_online = self._check_network_availability()

            if not is_online:
                logger.info(f"⏳ [WS Client] 网络不可用，等待{self.network_check_interval}s后重新检测...")
                timer = threading.Timer(self.network_check_interval, self._start_reconnection)
                timer.daemon = True
                timer.start()
                return

            logger.info("✅ [WS Client] 网络可用，继续重连...")

        self._connect()

    def _calculate_reconnect_delay(self) -> float:
        """计算重连延迟（指数退避+随机抖动）"""
        delay = self.initial_reconnect_delay * (self.reconnect_decay ** (self._reconnect_attempts - 1))
        delay = min(delay, self.max_reconnect_delay)

        if self.reconnect_jitter > 0:
            jitter = delay * self.reconnect_jitter * (random.random() * 2 - 1)
            delay += jitter

        return round(delay, 2)

    def _schedule_reconnect(self):
        """调度下一次重连"""
        if not self.reconnect_enabled or self.manual_close:
            return

        self._start_reconnection()

    def _stop_reconnect(self):
        """停止重连定时器"""
        if self._reconnect_timer:
            self._reconnect_timer.cancel()
            self._reconnect_timer = None

    # ══════════════════════════════════════════════
    # 网络检测模块
    # ══════════════════════════════════════════════

    def _check_network_availability(self) -> bool:
        """检查网络是否可用"""
        try:
            response = requests.get(
                self.network_check_url,
                timeout=3,
                headers={'User-Agent': 'DeviceWebSocketClient/1.0'}
            )

            was_available = self._network_available
            self._network_available = True

            if not was_available:
                logger.info("🌐 [WS Client] 网络已恢复!")
                self.on_network_change(online=True)

            return response.status_code == 200

        except Exception as error:
            was_available = self._network_available
            self._network_available = False

            if was_available:
                logger.warning("⚠️ [WS Client] 网络不可用!")
                self.on_network_change(online=False)

            return False

    def _start_network_check(self):
        """启动定期网络检测"""
        if not self.network_check_enabled:
            return

        def check_loop():
            while self._running:
                self._check_network_availability()
                time.sleep(self.network_check_interval)

        check_thread = threading.Thread(target=check_loop, daemon=True)
        check_thread.start()

    def _stop_network_check(self):
        """停止网络检测（通过设置标志位）"""
        pass  # 线程会通过 _running 标志自动退出

    # ══════════════════════════════════════════════
    # 心跳保活模块
    # ══════════════════════════════════════════════

    def _start_heartbeat(self):
        """启动心跳保活"""
        self._stop_heartbeat()

        def ping_loop():
            while self._running:
                time.sleep(self.ping_interval)

                with self._lock:
                    is_connected = (self.state == ConnectionState.CONNECTED and
                                  self.ws and self.ws.sock and self.ws.sock.connected)

                if is_connected:
                    try:
                        ping_msg = json.dumps({
                            'type': 'ping',
                            'timestamp': int(time.time() * 1000),
                            'device_id': self.device_id
                        })
                        self.ws.send(ping_msg)
                        logger.info("💓 [WS Client] Ping发送")

                        # 设置Pong超时定时器
                        self._pong_timer = threading.Timer(
                            self.pong_timeout,
                            self._handle_pong_timeout
                        )
                        self._pong_timer.daemon = True
                        self._pong_timer.start()

                    except Exception as error:
                        logger.error(f"[WS Client] ❌ Ping发送失败: {error}")

        heartbeat_thread = threading.Thread(target=ping_loop, daemon=True)
        heartbeat_thread.start()

    def _stop_heartbeat(self):
        """停止心跳"""
        if self._ping_timer:
            self._ping_timer.cancel()
            self._ping_timer = None

        if self._pong_timer:
            self._pong_timer.cancel()
            self._pong_timer = None

    def _handle_pong(self):
        """处理Pong响应"""
        if self._pong_timer:
            self._pong_timer.cancel()
            self._pong_timer = None
        logger.info("💚 [WS Client] Pong收到")

    def _handle_pong_timeout(self):
        """处理Pong超时"""
        logger.warning("⚠️ [WS Client] Pong超时，可能连接异常")
        if self.ws:
            self.ws.close(4001, 'Pong timeout')

    # ══════════════════════════════════════════════
    # 消息队列模块（离线缓存）
    # ══════════════════════════════════════════════

    def _enqueue_message(self, message: str) -> bool:
        """将消息加入发送队列"""
        try:
            if self._message_queue.qsize() >= self.message_queue_max_size:
                logger.warning("[WS Client] ⚠️  消息队列已满，丢弃最旧的消息")
                try:
                    self._message_queue.get_nowait()
                except queue.Empty:
                    pass

            queued_msg = QueuedMessage(
                message=message,
                timestamp=time.time(),
                retry_count=0
            )
            self._message_queue.put_nowait(queued_msg)

            logger.info(f"💾 [WS Client] 消息已缓存 (队列长度: {self._message_queue.qsize()})")
            return True

        except queue.Full:
            logger.error("[WS Client] ❌ 消息队列已满，无法添加新消息")
            return False

    def _flush_message_queue(self):
        """清空消息队列（重连成功后调用）"""
        if self._message_queue.empty():
            return

        logger.info(f"\n📤 [WS Client] 开始同步离线期间的缓存消息...")
        logger.info(f"   缓存消息数: {self._message_queue.qsize()}")

        success_count = 0
        fail_count = 0

        while not self._message_queue.empty():
            try:
                item = self._message_queue.get_nowait()

                with self._lock:
                    is_connected = (self.state == ConnectionState.CONNECTED and
                                  self.ws and self.ws.sock and self.ws.sock.connected)

                if is_connected:
                    self.ws.send(item.message)
                    success_count += 1
                else:
                    fail_count += 1
                    # 放回队列
                    self._message_queue.put_nowait(item)
                    break

            except queue.Empty:
                break
            except Exception as error:
                logger.error(f"[WS Client] ❌ 同步消息失败: {error}")
                fail_count += 1
                break

        logger.info(f"✅ [WS Client] 同步完成: 成功={success_count}, 失败={fail_count}, 剩余={self._message_queue.qsize()}")

    # ══════════════════════════════════════════════
    # 工具方法
    # ══════════════════════════════════════════════

    def get_state(self) -> Dict[str, Any]:
        """获取当前连接状态"""
        with self._lock:
            return {
                'state': self.state.name,
                'reconnect_attempts': self._reconnect_attempts,
                'current_reconnect_delay': self._current_delay,
                'message_queue_size': self._message_queue.qsize(),
                'network_available': self._network_available,
                'connected': self.state == ConnectionState.CONNECTED
            }

    def update_token(self, new_token: str):
        """更新认证Token（用于Token刷新场景）"""
        logger.info('[WS Client] 🔑 Token已更新')
        self.token = new_token

    def get_stats(self) -> Dict[str, Any]:
        """获取统计信息"""
        return {
            'reconnect_attempts': self._reconnect_attempts,
            'messages_queued': self._message_queue.qsize(),
            'network_status': 'online' if self._network_available else 'offline',
            'uptime': 'connected' if self.state == ConnectionState.CONNECTED else 'disconnected'
        }


# ══════════════════════════════════════════════
# 使用示例
# ══════════════════════════════════════════════

if __name__ == '__main__':
    import sys

    # 从命令行参数获取Token
    token = sys.argv[1] if len(sys.argv) > 1 else 'your_token_here'

    print("\n" + "=" * 60)
    print("🚀 设备WebSocket客户端 - 自动重连演示")
    print("=" * 60 + "\n")

    # 创建客户端实例
    client = DeviceWebSocketClient({
        'url': 'ws://localhost:8002/ws/device',
        'token': token,
        'device_id': '1234567890',

        # 重连配置
        'reconnect_enabled': True,
        'initial_reconnect_delay': 1.0,  # 初始1秒
        'max_reconnect_delay': 30.0,     # 最大30秒
        'reconnect_decay': 2.0,           # 指数退避 (2^n)
        'reconnect_jitter': 0.3,          # 随机抖动 ±30%

        # 心跳配置
        'ping_interval': 54,              # 54秒
        'pong_timeout': 60,               # 60秒超时

        # 网络检测
        'network_check_enabled': True,
        'network_check_interval': 5,      # 每5秒检测一次
        'network_check_url': 'http://localhost:8002/health',

        # 消息队列
        'message_queue_max_size': 1000,   # 最大缓存1000条
    })

    # 注册回调函数
    def on_open_callback(ws):
        print("\n✅ 连接已建立!")

    def on_message_callback(data, raw=None):
        print(f"\n📥 收到消息: {data}")

    def on_close_callback(code, reason):
        print(f"\n🔴 连接已关闭: code={code}, reason={reason}")

    def on_error_callback(error):
        print(f"\n❌ 错误发生: {error}")

    def on_reconnecting_callback(**kwargs):
        print(f"\n🔄 正在重连... (第{kwargs['attempt']}次)")

    def on_reconnected_callback(**kwargs):
        print(f"\n🎉 重连成功! (共尝试{kwargs['attempts']}次)")

    def on_network_change_callback(**kwargs):
        status = "在线" if kwargs['online'] else "离线"
        print(f"\n🌐 网络状态变化: {status}")

    client.on_open = on_open_callback
    client.on_message = on_message_callback
    client.on_close = on_close_callback
    client.on_error = on_error_callback
    client.on_reconnecting = on_reconnecting_callback
    client.on_reconnected = on_reconnected_callback
    client.on_network_change = on_network_change_callback

    # 启动连接
    print("📍 正在连接到服务器...\n")
    client.connect()

    # 启动网络检测
    client._start_network_check()

    try:
        # 定期发送测试消息
        message_count = 0
        while client._running:
            time.sleep(10)  # 每10秒发送一条消息

            message_count += 1
            test_message = {
                'type': 'status_report',
                'data': {
                    'battery_level': random.randint(60, 100),
                    'volume': random.randint(0, 100),
                    'playing': random.choice([True, False]),
                    'timestamp': int(time.time() * 1000),
                    'message_number': message_count
                }
            }

            print(f"\n📤 发送测试消息 #{message_count}")
            client.send(test_message)

    except KeyboardInterrupt:
        print("\n\n👋 用户中断，正在关闭客户端...")
        client.destroy()
        print("✅ 客户端已关闭\n")
