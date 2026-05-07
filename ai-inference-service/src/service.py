"""
音轨分离服务处理器
处理完整的音轨分离流程
"""

import asyncio
import aiohttp
from pathlib import Path
from datetime import datetime
from loguru import logger
from typing import Optional

from src.models import SeparationTask, TaskStatus, TaskProgress
from src.task_manager import task_manager
from src.utils import (
    generate_task_id,
    download_audio_file,
    get_audio_extension,
    ensure_dir
)
from src.separator import separator
from src.config import settings


class SeparationService:
    """音轨分离服务"""
    
    def __init__(self):
        self.separator = separator
        self.task_manager = task_manager
    
    async def start_separation(
        self,
        audio_url: str,
        user_id: Optional[str] = None,
        device_id: Optional[str] = None,
        callback_url: Optional[str] = None
    ) -> SeparationTask:
        """
        开始音轨分离任务
        
        Args:
            audio_url: 音频文件 URL
            user_id: 用户 ID
            device_id: 设备 ID
            callback_url: 回调 URL
            
        Returns:
            SeparationTask: 创建的任务
        """
        # 创建任务
        task = SeparationTask(
            task_id=generate_task_id(),
            user_id=user_id,
            device_id=device_id,
            audio_url=audio_url
        )
        
        await self.task_manager.create_task(task)
        logger.info(f"创建分离任务：{task.task_id}")
        
        # 异步处理任务
        asyncio.create_task(
            self._process_task(task, callback_url)
        )
        
        return task
    
    async def _process_task(
        self,
        task: SeparationTask,
        callback_url: Optional[str] = None
    ):
        """处理任务"""
        try:
            # 1. 下载音频文件
            await self._download_audio(task)
            
            # 2. 执行音轨分离
            await self._separate_audio(task)
            
            # 3. 上传分离结果（如果有回调 URL）
            if callback_url:
                await self._upload_results(task, callback_url)
            
            # 4. 完成任务
            await self.task_manager.update_status(
                task.task_id,
                TaskStatus.COMPLETED,
                TaskProgress(percentage=100, stage="completed", message="分离完成")
            )
            
            logger.info(f"任务完成：{task.task_id}")
            
        except Exception as e:
            logger.error(f"任务失败：{task.task_id}, 错误：{e}")
            await self.task_manager.update_status(
                task.task_id,
                TaskStatus.FAILED,
                TaskProgress(percentage=0, stage="failed", message=str(e)),
                error_message=str(e)
            )
    
    async def _download_audio(self, task: SeparationTask):
        """下载音频文件"""
        await self.task_manager.update_status(
            task.task_id,
            TaskStatus.DOWNLOADING,
            TaskProgress(percentage=10, stage="downloading", message="正在下载音频文件")
        )
        
        # 生成保存路径
        ext = get_audio_extension(task.audio_url)
        input_filename = f"{task.task_id}{ext}"
        input_path = Path(settings.input_path) / input_filename
        
        # 确保目录存在
        ensure_dir(input_path)
        
        # 下载文件
        success = await download_audio_file(task.audio_url, input_path)
        
        if not success:
            raise Exception(f"下载音频文件失败：{task.audio_url}")
        
        task.input_path = input_path
        logger.info(f"音频下载成功：{input_path}")
        
        await self.task_manager.update_status(
            task.task_id,
            TaskStatus.PROCESSING,
            TaskProgress(percentage=30, stage="processing", message="正在处理音频")
        )
    
    async def _separate_audio(self, task: SeparationTask):
        """执行音轨分离"""
        await self.task_manager.update_status(
            task.task_id,
            TaskStatus.PROCESSING,
            TaskProgress(percentage=50, stage="processing", message="正在分离音轨")
        )
        
        # 确保输出目录存在
        output_dir = Path(settings.output_path)
        ensure_dir(output_dir / task.task_id)
        
        # 执行分离
        output_files = await self.separator.separate(
            task.input_path,
            output_dir / task.task_id
        )
        
        # 保存输出文件路径
        await self.task_manager.set_output_files(task.task_id, output_files)
        logger.info(f"音轨分离完成：{output_files}")
        
        await self.task_manager.update_status(
            task.task_id,
            TaskStatus.UPLOADING,
            TaskProgress(percentage=80, stage="uploading", message="正在上传结果")
        )
    
    async def _upload_results(
        self,
        task: SeparationTask,
        callback_url: str
    ):
        """上传分离结果"""
        upload_urls = {}
        
        # 为每个输出文件生成上传 URL
        for stem, file_path in task.output_files.items():
            upload_url = f"{callback_url}/{stem}"
            upload_urls[stem] = upload_url
        
        task.upload_urls = upload_urls
        
        # 上传文件
        for stem, file_path in task.output_files.items():
            upload_url = upload_urls.get(stem)
            if upload_url:
                success = await self._upload_file(Path(file_path), upload_url)
                if not success:
                    logger.warning(f"上传失败：{upload_url}")
    
    async def _upload_file(self, file_path: Path, upload_url: str) -> bool:
        """上传单个文件"""
        try:
            async with aiohttp.ClientSession() as session:
                async with aiofiles.open(file_path, 'rb') as f:
                    file_data = await f.read()
                
                data = aiohttp.FormData()
                data.add_field('file', file_data, filename=file_path.name)
                
                async with session.put(upload_url, data=data, timeout=60) as response:
                    if response.status in [200, 201]:
                        logger.info(f"上传成功：{upload_url}")
                        return True
                    else:
                        logger.error(f"上传失败：{upload_url}, 状态码：{response.status}")
                        return False
        except Exception as e:
            logger.error(f"上传异常：{e}")
            return False
    
    async def get_task_status(self, task_id: str) -> Optional[SeparationTask]:
        """获取任务状态"""
        return await self.task_manager.get_task(task_id)
    
    async def cancel_task(self, task_id: str) -> bool:
        """取消任务"""
        task = await self.task_manager.get_task(task_id)
        if not task:
            return False
        
        if task.status in [TaskStatus.COMPLETED, TaskStatus.FAILED]:
            return False
        
        await self.task_manager.update_status(
            task_id,
            TaskStatus.FAILED,
            TaskProgress(percentage=0, stage="cancelled", message="任务已取消"),
            error_message="用户取消"
        )
        
        return True


# 全局服务实例
separation_service = SeparationService()
