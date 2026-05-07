"""
配置管理
"""

from pydantic_settings import BaseSettings
from pathlib import Path
from typing import Optional


class Settings(BaseSettings):
    # 服务配置
    host: str = "0.0.0.0"
    port: int = 8004
    
    # 模型配置
    model_name: str = "bsroformer_scnet"
    model_path: str = "models"
    
    # 数据路径
    data_path: str = "data"
    input_path: str = "data/input"
    output_path: str = "data/output"
    
    # GPU 配置
    gpu_device: int = 0
    use_gpu: bool = False
    
    # 性能配置
    max_batch_size: int = 1
    max_concurrent: int = 3
    
    # 任务配置
    task_timeout: int = 300  # 任务超时时间（秒）
    max_tasks: int = 10  # 最大任务数
    
    # 日志配置
    log_level: str = "INFO"
    
    class Config:
        env_file = ".env"
        case_sensitive = False


settings = Settings()
