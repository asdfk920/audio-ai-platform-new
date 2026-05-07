"""
任务管理器
管理音轨分离任务的生命周期
"""

import asyncio
from typing import Dict, Optional
from datetime import datetime, timedelta
from loguru import logger

from src.models import SeparationTask, TaskStatus, TaskProgress
from src.config import settings


class TaskManager:
    """任务管理器 - 单例模式"""
    
    _instance = None
    _lock = asyncio.Lock()
    
    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(TaskManager, cls).__new__(cls)
        return cls._instance
    
    def __init__(self):
        if not hasattr(self, '_initialized'):
            self.tasks: Dict[str, SeparationTask] = {}
            self._initialized = True
    
    async def create_task(self, task: SeparationTask) -> SeparationTask:
        """创建新任务"""
        async with self._lock:
            if len(self.tasks) >= settings.max_tasks:
                raise ValueError(f"任务数已达上限：{settings.max_tasks}")
            
            self.tasks[task.task_id] = task
            logger.info(f"创建任务：{task.task_id}")
            return task
    
    async def get_task(self, task_id: str) -> Optional[SeparationTask]:
        """获取任务"""
        return self.tasks.get(task_id)
    
    async def update_status(
        self,
        task_id: str,
        status: TaskStatus,
        progress: Optional[TaskProgress] = None,
        error_message: Optional[str] = None
    ):
        """更新任务状态"""
        if task_id not in self.tasks:
            raise ValueError(f"任务不存在：{task_id}")
        
        task = self.tasks[task_id]
        task.status = status
        
        if progress:
            task.progress = progress
        
        if error_message:
            task.error_message = error_message
        
        if status == TaskStatus.PROCESSING and not task.started_at:
            task.started_at = datetime.now()
        elif status in [TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.TIMEOUT]:
            task.completed_at = datetime.now()
            if task.started_at:
                task.processing_time = (task.completed_at - task.started_at).total_seconds()
        
        logger.info(f"任务 {task_id} 状态更新为：{status}")
    
    async def set_output_files(self, task_id: str, output_files: Dict[str, str]):
        """设置输出文件"""
        if task_id not in self.tasks:
            raise ValueError(f"任务不存在：{task_id}")
        
        self.tasks[task_id].output_files = output_files
    
    async def delete_task(self, task_id: str) -> bool:
        """删除任务"""
        if task_id in self.tasks:
            del self.tasks[task_id]
            logger.info(f"删除任务：{task_id}")
            return True
        return False
    
    async def list_tasks(self, status: Optional[TaskStatus] = None) -> list:
        """列出任务"""
        if status is None:
            return list(self.tasks.values())
        return [task for task in self.tasks.values() if task.status == status]
    
    async def cleanup_timeout_tasks(self):
        """清理超时任务"""
        now = datetime.now()
        timeout_tasks = []
        
        for task_id, task in self.tasks.items():
            if task.status == TaskStatus.PROCESSING and task.started_at:
                elapsed = now - task.started_at
                if elapsed > timedelta(seconds=settings.task_timeout):
                    timeout_tasks.append(task_id)
        
        for task_id in timeout_tasks:
            await self.update_status(
                task_id,
                TaskStatus.TIMEOUT,
                TaskProgress(percentage=0, stage="timeout", message="任务超时")
            )
            logger.warning(f"任务超时：{task_id}")
    
    def get_stats(self) -> Dict[str, int]:
        """获取任务统计"""
        stats = {
            "total": len(self.tasks),
            "pending": 0,
            "processing": 0,
            "completed": 0,
            "failed": 0
        }
        
        for task in self.tasks.values():
            status = task.status.value if isinstance(task.status, TaskStatus) else task.status
            if status in stats:
                stats[status] += 1
        
        return stats


# 全局任务管理器实例
task_manager = TaskManager()
