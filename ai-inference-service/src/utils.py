"""
工具函数
"""

import uuid
import aiohttp
import aiofiles
from pathlib import Path
from datetime import datetime
from typing import Optional
from loguru import logger

from src.config import settings


def generate_task_id() -> str:
    """生成唯一任务 ID"""
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    unique_id = str(uuid.uuid4())[:8]
    return f"task_{timestamp}_{unique_id}"


def generate_filename(extension: str) -> str:
    """生成唯一文件名"""
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    unique_id = str(uuid.uuid4())[:8]
    return f"{timestamp}_{unique_id}{extension}"


def ensure_dir(path: Path):
    """确保目录存在"""
    if isinstance(path, str):
        path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)


async def download_audio_file(url: str, save_path: Path) -> bool:
    """
    下载音频文件
    
    Args:
        url: 音频文件 URL
        save_path: 保存路径
        
    Returns:
        bool: 下载是否成功
    """
    try:
        ensure_dir(save_path)
        
        async with aiohttp.ClientSession() as session:
            async with session.get(url, timeout=60) as response:
                if response.status != 200:
                    logger.error(f"下载失败：{url}, 状态码：{response.status}")
                    return False
                
                async with aiofiles.open(save_path, 'wb') as f:
                    async for chunk in response.content.iter_chunked(8192):
                        await f.write(chunk)
        
        logger.info(f"下载成功：{save_path}")
        return True
        
    except Exception as e:
        logger.error(f"下载失败：{e}")
        return False


async def upload_file(file_path: Path, upload_url: str) -> bool:
    """
    上传文件到指定 URL
    
    Args:
        file_path: 文件路径
        upload_url: 上传 URL
        
    Returns:
        bool: 上传是否成功
    """
    try:
        async with aiohttp.ClientSession() as session:
            async with aiofiles.open(file_path, 'rb') as f:
                file_data = await f.read()
            
            data = aiohttp.FormData()
            data.add_field('file', file_data, filename=file_path.name)
            
            async with session.put(upload_url, data=data, timeout=60) as response:
                if response.status not in [200, 201]:
                    logger.error(f"上传失败：{upload_url}, 状态码：{response.status}")
                    return False
        
        logger.info(f"上传成功：{upload_url}")
        return True
        
    except Exception as e:
        logger.error(f"上传失败：{e}")
        return False


def get_audio_extension(url: str) -> str:
    """从 URL 获取音频文件扩展名"""
    path = url.split('?')[0]  # 移除查询参数
    return Path(path).suffix.lower() or '.wav'


def format_time(seconds: float) -> str:
    """格式化时间"""
    minutes = int(seconds // 60)
    secs = int(seconds % 60)
    return f"{minutes}:{secs:02d}"


def calculate_rtf(processing_time: float, audio_duration: float) -> float:
    """计算实时因子"""
    if audio_duration == 0:
        return 0.0
    return processing_time / audio_duration
