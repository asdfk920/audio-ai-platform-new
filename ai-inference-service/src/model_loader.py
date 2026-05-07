"""
BSRoformer SCNet 模型加载器
支持从 HuggingFace 或本地加载模型
"""

import torch
import torch.nn as nn
from pathlib import Path
from typing import Optional, Dict, Any
from loguru import logger

from src.config import settings


class ModelLoader:
    """模型加载器"""
    
    def __init__(
        self,
        model_name: str = "bsroformer_scnet",
        model_path: str = "/app/models",
        gpu_device: int = 0
    ):
        self.model_name = model_name
        self.model_path = Path(model_path)
        self.gpu_device = gpu_device
        self.device = self._get_device()
        self.model = None
    
    def _get_device(self) -> torch.device:
        """获取运行设备"""
        if torch.cuda.is_available():
            device = torch.device(f"cuda:{self.gpu_device}")
            logger.info(f"使用 GPU: {torch.cuda.get_device_name(device)}")
            logger.info(f"GPU 内存：{torch.cuda.get_device_properties(device).total_memory / 1024**3:.2f} GB")
        else:
            device = torch.device("cpu")
            logger.warning("未检测到 GPU，使用 CPU 运行")
        
        return device
    
    def is_gpu_available(self) -> bool:
        """检查 GPU 是否可用"""
        return torch.cuda.is_available()
    
    def load_model(self) -> nn.Module:
        """加载模型"""
        if self.model is not None:
            return self.model
        
        logger.info(f"正在加载模型：{self.model_name}")
        
        try:
            # 尝试从本地加载
            model_path = self.model_path / f"{self.model_name}.pt"
            
            if model_path.exists():
                logger.info(f"从本地加载模型：{model_path}")
                self.model = self._load_from_file(model_path)
            else:
                # 从 HuggingFace 加载（示例）
                logger.info("从 HuggingFace 加载模型...")
                self.model = self._load_from_huggingface()
                
                # 保存到本地
                self._save_to_file(model_path)
            
            # 移动到设备
            self.model = self.model.to(self.device)
            self.model.eval()
            
            logger.info("模型加载成功")
            return self.model
            
        except Exception as e:
            logger.error(f"模型加载失败：{e}")
            raise
    
    def _load_from_file(self, path: Path) -> nn.Module:
        """从文件加载模型"""
        checkpoint = torch.load(path, map_location=self.device)
        
        # 创建模型架构
        model = self._create_model()
        
        # 加载权重
        if isinstance(checkpoint, dict) and "state_dict" in checkpoint:
            model.load_state_dict(checkpoint["state_dict"])
        else:
            model.load_state_dict(checkpoint)
        
        return model
    
    def _load_from_huggingface(self) -> nn.Module:
        """从 HuggingFace 加载模型"""
        try:
            from transformers import AutoModel
            
            # BSRoformer SCNet 模型 ID（示例）
            model_id = "tsurutsu/bsroformer_scnet"
            
            model = AutoModel.from_pretrained(model_id)
            return model
            
        except ImportError:
            logger.warning("无法从 HuggingFace 加载，创建默认模型")
            return self._create_model()
    
    def _create_model(self) -> nn.Module:
        """创建模型架构"""
        # 这里需要根据 BSRoformer SCNet 的实际架构实现
        # 以下是一个示例架构
        
        class BSRoformerSCNet(nn.Module):
            def __init__(self):
                super().__init__()
                # 模型定义
                self.stem = nn.Conv2d(2, 64, kernel_size=7, padding=3)
                self.encoder = nn.Sequential(
                    nn.Conv2d(64, 128, kernel_size=3, padding=1),
                    nn.BatchNorm2d(128),
                    nn.ReLU(),
                )
                self.decoder = nn.Sequential(
                    nn.ConvTranspose2d(128, 64, kernel_size=4, stride=2, padding=1),
                    nn.BatchNorm2d(64),
                    nn.ReLU(),
                )
                self.output = nn.Conv2d(64, 2, kernel_size=7, padding=3)
            
            def forward(self, x):
                x = self.stem(x)
                x = self.encoder(x)
                x = self.decoder(x)
                x = self.output(x)
                return x
        
        return BSRoformerSCNet()
    
    def _save_to_file(self, path: Path):
        """保存模型到文件"""
        if self.model is None:
            return
        
        path.parent.mkdir(parents=True, exist_ok=True)
        
        # 保存状态字典
        torch.save(self.model.state_dict(), path)
        logger.info(f"模型已保存到：{path}")
    
    def unload_model(self):
        """卸载模型"""
        if self.model is not None:
            self.model = None
            if torch.cuda.is_available():
                torch.cuda.empty_cache()
            logger.info("模型已卸载")
