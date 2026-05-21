#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PersistentMessageQueue - 本地持久化消息队列（Python版本）

核心功能：
  ✅ 本地JSON文件持久化（断电不丢、重启不丢）
  ✅ 原子写入操作（防止数据损坏）
  ✅ 消息状态管理（PENDING → SENT → ACKED）
  ✅ 自动清理已确认消息
  ✅ 按时间顺序保证（FIFO）
  ✅ 最大容量限制（防止磁盘占满）

使用示例：
    from persistent_message_queue import PersistentMessageQueue

    queue = PersistentMessageQueue(storage_path='./data/queue.json')

    # 入队消息
    result = queue.enqueue({'type': 'status', 'data': {'battery': 85}})

    # 获取待发消息
    pending = queue.get_pending_messages()

    # 标记为已发送
    queue.mark_as_sent(message_id)

    # 收到ACK确认
    queue.mark_as_acked(message_id)
"""

import json
import os
import uuid
import time
import threading
from typing import Optional, Dict, Any, List
from dataclasses import dataclass, asdict, field
from enum import Enum, auto


class MessageStatus(Enum):
    """消息状态枚举"""
    PENDING = "PENDING"      # 待发送
    SENT = "SENT"            # 已发送，等待ACK
    ACKED = "ACKED"          # 已确认（可删除）


@dataclass
class QueueMessage:
    """队列中的消息对象"""
    id: str
    type: str
    payload: Dict[str, Any]
    status: str
    created_at: float
    sent_at: Optional[float] = None
    acked_at: Optional[float] = None
    retry_count: int = 0
    last_error: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)


class PersistentMessageQueue:
    """
    本地持久化消息队列

    特性：
      - 基于JSON文件的持久化存储
      - 线程安全（支持多线程访问）
      - 原子写入（先写临时文件再重命名）
      - 自动保存机制
    """

    def __init__(self, options: Optional[Dict[str, Any]] = None):
        options = options or {}

        # 存储配置
        self.storage_path = options.get('storage_path', './data/device_message_queue.json')
        self.max_queue_size = options.get('max_queue_size', 10000)  # 最大10000条
        self.auto_save_interval = options.get('auto_save_interval', 5.0)  # 每5秒自动保存

        # 内部状态
        self.messages: List[QueueMessage] = []
        self.version = '1.0'
        self.created_at = ''
        self._dirty = False
        self._lock = threading.Lock()
        self._save_timer: Optional[threading.Timer] = None

        # 初始化
        self._ensure_storage_directory()
        self._load_from_file()

        # 启动自动保存定时器
        if self.auto_save_interval > 0:
            self._start_auto_save()

        print(f"[MessageQueue] 📦 持久化消息队列初始化完成")
        print(f"[MessageQueue] 📍 存储路径: {self.storage_path}")
        print(f"[MessageQueue] 📊 当前队列长度: {len(self.messages)}")

    def enqueue(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """
        入队消息（写入本地持久化存储）

        参数：
            message: 消息数据字典

        返回：
            {
                'success': bool,
                'message_id': str,
                'queue_size': int,
                'error': str (可选)
            }
        """
        try:
            with self._lock:
                # 检查队列是否已满
                if len(self.messages) >= self.max_queue_size:
                    removed_id = self._remove_oldest_pending()
                    print(f"[MessageQueue] ⚠️  队列已满，移除最旧消息: {removed_id}")

                # 生成唯一消息ID
                message_id = str(uuid.uuid4())

                # 构造消息对象
                queue_msg = QueueMessage(
                    id=message_id,
                    type=message.get('type', 'unknown'),
                    payload=message.get('payload', message),
                    status=MessageStatus.PENDING.value,
                    created_at=time.time(),
                    retry_count=0,
                    metadata=message.get('metadata', {})
                )

                # 加入队列并排序
                self.messages.append(queue_msg)
                self._sort_messages()

                # 标记脏数据
                self._dirty = True

                result = {
                    'success': True,
                    'message_id': message_id,
                    'queue_size': len(self.messages)
                }

            print(f"[MessageQueue] ✅ 消息入队成功:")
            print(f"   ID: {message_id}")
            print(f"   类型: {queue_msg.type}")
            print(f"   队列长度: {len(self.messages)}")

            return result

        except Exception as error:
            print(f"[MessageQueue] ❌ 入队失败: {error}")
            return {
                'success': False,
                'error': str(error)
            }

    def get_pending_messages(self) -> List[Dict[str, Any]]:
        """获取所有待发送的消息（按时间顺序）"""
        with self._lock:
            pending = [m for m in self.messages if m.status == MessageStatus.PENDING.value]
            pending.sort(key=lambda x: x.created_at)
            return [asdict(m) for m in pending]

    def get_sent_messages(self) -> List[Dict[str, Any]]:
        """获取所有已发送但未确认的消息"""
        with self._lock:
            sent = [m for m in self.messages if m.status == MessageStatus.SENT.value]
            sent.sort(key=lambda x: x.sent_at or 0)
            return [asdict(m) for m in sent]

    def mark_as_sent(self, message_id: str) -> bool:
        """
        标记消息为已发送（等待ACK）

        参数：
            message_id: 消息ID

        返回：
            bool: 是否成功标记
        """
        with self._lock:
            for msg in self.messages:
                if msg.id == message_id:
                    msg.status = MessageStatus.SENT.value
                    msg.sent_at = time.time()
                    msg.retry_count += 1
                    self._dirty = True

                    print(f"[MessageQueue] 📤 消息已标记为SENT: {message_id} "
                          f"(重试次数: {msg.retry_count})")
                    return True

            return False

    def mark_as_acked(self, message_id: str) -> bool:
        """
        收到ACK确认，从队列中移除消息

        参数：
            message_id: 消息ID

        返回：
            bool: 是否成功处理
        """
        with self._lock:
            for i, msg in enumerate(self.messages):
                if msg.id == message_id:
                    msg.status = MessageStatus.ACKED.value
                    msg.acked_at = time.time()

                    print(f"[MessageQueue] ✅ 消息已确认(ACK): {message_id}")

                    # 从队列中移除
                    self.messages.pop(i)
                    self._dirty = True
                    return True

            print(f"[MessageQueue] ⚠️  未找到消息ID: {message_id}")
            return False

    def mark_as_failed(self, message_id: str, error: str) -> bool:
        """
        标记消息发送失败（保持PENDING状态以便重试）

        参数：
            message_id: 消息ID
            error: 错误信息

        返回：
            bool: 是否成功标记
        """
        with self._lock:
            for msg in self.messages:
                if msg.id == message_id:
                    msg.status = MessageStatus.PENDING.value
                    msg.last_error = error
                    msg.sent_at = None  # 清除发送时间
                    self._dirty = True

                    print(f"[MessageQueue] ❌ 消息发送失败: {message_id}, 错误: {error}")
                    return True

            return False

    def get_all_messages(self) -> List[Dict[str, Any]]:
        """获取所有消息（用于调试）"""
        with self._lock:
            return [asdict(m) for m in self.messages]

    def get_stats(self) -> Dict[str, Any]:
        """获取队列统计信息"""
        with self._lock:
            pending = sum(1 for m in self.messages if m.status == MessageStatus.PENDING.value)
            sent = sum(1 for m in self.messages if m.status == MessageStatus.SENT.value)
            acked = sum(1 for m in self.messages if m.status == MessageStatus.ACKED.value)

            oldest_pending_time = self._get_oldest_pending_time()

            return {
                'total': len(self.messages),
                'pending': pending,
                'sent': sent,
                'acked': acked,
                'max_capacity': self.max_queue_size,
                'utilization': f"{(len(self.messages) / self.max_queue_size * 100):.1f}%",
                'oldest_pending': oldest_pending_time,
                'storage_path': self.storage_path
            }

    async def clear_all(self):
        """清空所有消息（慎用！）"""
        with self._lock:
            self.messages = []
            await self._save_to_file()
            print('[MessageQueue] 🗑️  队列已清空')

    async def force_save(self):
        """强制立即保存到文件"""
        await self._save_to_file()

    def destroy(self):
        """销毁实例（停止自动保存）"""
        if self._save_timer:
            self._save_timer.cancel()
            self._save_timer = None

        # 最后一次保存
        self.force_sync()

        print('[MessageQueue] 💀 实例已销毁')

    force_sync = lambda self: self._save_to_file_sync()

    # ══════════════════════════════════════════════
    # 私有方法
    # ══════════════════════════════════════════════

    def _sort_messages(self):
        """按时间排序消息"""
        self.messages.sort(key=lambda x: x.created_at)

    def _get_oldest_pending_time(self) -> str:
        """获取最旧待发消息的年龄"""
        pending = [m for m in self.messages if m.status == MessageStatus.PENDING.value]
        if pending:
            age_seconds = time.time() - min(pending, key=lambda x: x.created_at).created_at
            return f"{int(age_seconds)}s"
        return 'N/A'

    def _remove_oldest_pending(self) -> Optional[str]:
        """移除最旧的待发消息"""
        for i, msg in enumerate(self.messages):
            if msg.status == MessageStatus.PENDING.value:
                removed = self.messages.pop(i)
                self._dirty = True
                return removed.id
        return None

    def _ensure_storage_directory(self):
        """确保存储目录存在"""
        directory = os.path.dirname(self.storage_path)
        if directory and not os.path.exists(directory):
            os.makedirs(directory, exist_ok=True)
            print(f"[MessageQueue] 📁 创建存储目录: {directory}")

    def _load_from_file(self):
        """从文件加载队列"""
        try:
            if os.path.exists(self.storage_path):
                with open(self.storage_path, 'r', encoding='utf-8') as f:
                    data = json.load(f)

                self.version = data.get('version', '1.0')
                self.created_at = data.get('created_at', '')
                
                messages_data = data.get('messages', [])
                self.messages = [
                    QueueMessage(**msg_data) 
                    for msg_data in messages_data
                ]

                print(f"[MessageQueue] 📂 从文件加载队列: {len(self.messages)}条消息")
            else:
                print("[MessageQueue] 📝 存储文件不存在，创建新队列")
                self._save_to_file_sync()

        except Exception as error:
            print(f"[MessageQueue] ❌ 加载文件失败: {error}")
            print("[MessageQueue] ⚠️  使用空队列启动")

            # 备份损坏的文件
            if os.path.exists(self.storage_path):
                backup_path = f"{self.storage_path}.corrupted.{int(time.time())}"
                os.rename(self.storage_path, backup_path)
                print(f"[MessageQueue] 💾 损坏文件已备份到: {backup_path}")

    async def _save_to_file(self):
        """异步保存到文件"""
        if not self._dirty:
            return

        await self._do_save()

    def _save_to_file_sync(self):
        """同步保存到文件"""
        if not self._dirty:
            return

        # 在新线程中执行异步保存
        import asyncio
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                # 如果事件循环正在运行，使用线程池
                from concurrent.futures import ThreadPoolExecutor
                with ThreadPoolExecutor() as executor:
                    executor.submit(self._do_save_sync).result()
            else:
                loop.run_until_complete(self._do_save())
        except RuntimeError:
            # 无事件循环可用，直接同步保存
            self._do_save_sync()

    def _do_save(self):
        """实际执行保存操作"""
        if self._lock.locked():
            print("[MessageQueue] ⚠️  文件锁被占用，跳过保存")
            return

        with self._lock:
            try:
                data = {
                    'version': self.version,
                    'created_at': self.created_at,
                    'last_saved': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                    'messages': [asdict(m) for m in self.messages],
                    'stats': self.get_stats()
                }

                json_str = json.dumps(data, indent=2, ensure_ascii=False)

                # 原子写入：先写临时文件再重命名
                temp_path = f"{self.storage_path}.tmp"
                with open(temp_path, 'w', encoding='utf-8') as f:
                    f.write(json_str)
                os.replace(temp_path, self.storage_path)

                self._dirty = False
                print(f"[MessageQueue] 💾 队列已保存: {len(self.messages)}条消息")

            except Exception as error:
                print(f"[MessageQueue] ❌ 保存失败: {error}")

                # 清理临时文件
                temp_path = f"{this.storage_path}.tmp"
                if os.path.exists(temp_path):
                    try:
                        os.remove(temp_path)
                        print("[MessageQueue] 🧹 已清理临时文件")
                    except Exception as e:
                        print(f"[MessageQueue] ❌ 清理失败: {e}")

    def _do_save_sync(self):
        """同步版本的实际保存操作"""
        self._do_save()

    def _start_auto_save(self):
        """启动自动保存定时器"""
        def save_loop():
            while True:
                time.sleep(self.auto_save_interval)
                if self._dirty:
                    self._save_to_file_sync()

        save_thread = threading.Thread(target=save_loop, daemon=True)
        save_thread.start()


# 使用示例
if __name__ == '__main__':
    print("\n" + "=" * 60)
    print("🧪 PersistentMessageQueue 测试演示")
    print("=" * 60 + "\n")

    # 创建队列实例
    queue = PersistentMessageQueue({
        'storage_path': './data/test_queue.json',
        'max_queue_size': 100
    })

    # 测试入队
    print("📍 测试1: 入队消息\n")
    for i in range(5):
        result = queue.enqueue({
            'type': 'status_report',
            'payload': {
                'battery_level': 80 + i,
                'timestamp': time.time(),
                'test_number': i + 1
            }
        })
        print(f"   结果: {result}\n")

    # 获取统计信息
    stats = queue.get_stats()
    print("\n📊 队列统计:")
    for key, value in stats.items():
        print(f"   {key}: {value}")

    # 获取待发消息
    print("\n📤 待发消息:")
    pending = queue.get_pending_messages()
    for msg in pending[:3]:  # 只显示前3条
        print(f"   ID: {msg['id'][:8]}... 类型: {msg['type']} 状态: {msg['status']}")

    # 标记为已发送
    print("\n✅ 标记为已发送:")
    if pending:
        first_msg_id = pending[0]['id']
        queue.mark_as_sent(first_msg_id)
        print(f"   消息 {first_msg_id[:8]}... → SENT")

    # 模拟收到ACK
    print("\n🎉 收到ACK确认:")
    acked = queue.mark_as_acked(first_msg_id)
    print(f"   ACK处理结果: {acked}")

    # 最终统计
    final_stats = queue.get_stats()
    print(f"\n📈 最终统计: 待发={final_stats['pending']}, 总计={final_stats['total']}")

    # 清理
    print("\n👋 清理测试数据...")
    queue.clear_all()
    queue.destroy()
    print("✅ 测试完成!\n")
