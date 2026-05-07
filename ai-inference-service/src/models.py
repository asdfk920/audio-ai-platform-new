"""
数据模型定义
"""

from enum import Enum
from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime


class TaskStatus(str, Enum):
    """任务状态枚举"""
    PENDING = "pending"  # 等待处理
    DOWNLOADING = "downloading"  # 下载中
    PROCESSING = "processing"  # 处理中
    UPLOADING = "uploading"  # 上传中
    COMPLETED = "completed"  # 完成
    FAILED = "failed"  # 失败
    TIMEOUT = "timeout"  # 超时


class TaskProgress(BaseModel):
    """任务进度"""
    percentage: float = Field(..., ge=0, le=100, description="进度百分比")
    stage: str = Field(..., description="当前阶段")
    message: str = Field(default="", description="进度消息")


class SeparationTask(BaseModel):
    """音轨分离任务"""
    task_id: str = Field(..., description="任务 ID")
    user_id: Optional[str] = Field(None, description="用户 ID")
    device_id: Optional[str] = Field(None, description="设备 ID")
    
    # 输入信息
    audio_url: str = Field(..., description="音频文件 URL")
    
    # 任务状态
    status: TaskStatus = Field(default=TaskStatus.PENDING, description="任务状态")
    progress: TaskProgress = Field(default_factory=lambda: TaskProgress(percentage=0, stage="pending"), description="任务进度")
    
    # 输出信息
    output_files: Optional[Dict[str, str]] = Field(None, description="输出文件路径")
    upload_urls: Optional[Dict[str, str]] = Field(None, description="上传 URL 映射")
    
    # 时间信息
    created_at: datetime = Field(default_factory=datetime.now, description="创建时间")
    started_at: Optional[datetime] = Field(None, description="开始处理时间")
    completed_at: Optional[datetime] = Field(None, description="完成时间")
    
    # 错误信息
    error_message: Optional[str] = Field(None, description="错误消息")
    
    # 性能指标
    processing_time: Optional[float] = Field(None, description="处理时间（秒）")
    
    class Config:
        use_enum_values = True


class TaskResponse(BaseModel):
    """任务响应"""
    success: bool
    task_id: Optional[str] = None
    message: str
    data: Optional[SeparationTask] = None


class TaskStatusResponse(BaseModel):
    """任务状态响应"""
    success: bool
    task_id: str
    status: str
    progress: TaskProgress
    output_files: Optional[Dict[str, str]] = None
    error_message: Optional[str] = None
    processing_time: Optional[float] = None


class StartSeparationRequest(BaseModel):
    """开始分离请求"""
    audio_url: str = Field(..., description="音频文件 URL")
    user_id: Optional[str] = Field(None, description="用户 ID")
    device_id: Optional[str] = Field(None, description="设备 ID")
    callback_url: Optional[str] = Field(None, description="回调 URL")
    
    class Config:
        json_schema_extra = {
            "example": {
                "audio_url": "https://example.com/audio/song.mp3",
                "user_id": "user_123",
                "device_id": "device_456",
                "callback_url": "https://your-server.com/callback"
            }
        }
