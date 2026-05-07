"""
音频分离器
执行音轨分离的核心逻辑
"""

import torch
import torchaudio
from pathlib import Path
from typing import Dict, Optional
from loguru import logger
import numpy as np

from src.config import settings


class AudioSeparator:
    """音频分离器"""
    
    def __init__(self):
        self.device = self._get_device()
        self.model = None
        self.sample_rate = 44100
    
    def _get_device(self) -> torch.device:
        """获取运行设备"""
        if settings.use_gpu and torch.cuda.is_available():
            device = torch.device(f"cuda:{settings.gpu_device}")
            logger.info(f"使用 GPU: {torch.cuda.get_device_name(device)}")
        else:
            device = torch.device("cpu")
            logger.info("使用 CPU 运行")
        
        return device
    
    def load_model(self, model_path: Optional[str] = None):
        """加载模型"""
        if self.model is not None:
            logger.info("模型已加载")
            return
        
        try:
            # 尝试从本地加载
            if model_path:
                model_file = Path(model_path) / f"{settings.model_name}.pt"
                if model_file.exists():
                    logger.info(f"从本地加载模型：{model_file}")
                    self.model = self._load_from_file(model_file)
                else:
                    logger.warning(f"模型文件不存在：{model_file}, 创建默认模型")
                    self.model = self._create_model()
            else:
                logger.info("创建默认模型")
                self.model = self._create_model()
            
            # 移动到设备
            self.model = self.model.to(self.device)
            self.model.eval()
            
            logger.info("模型加载成功")
            
        except Exception as e:
            logger.error(f"模型加载失败：{e}")
            raise
    
    def _load_from_file(self, path: Path) -> torch.nn.Module:
        """从文件加载模型"""
        checkpoint = torch.load(path, map_location=self.device, weights_only=True)
        model = self._create_model()
        
        if isinstance(checkpoint, dict) and "state_dict" in checkpoint:
            model.load_state_dict(checkpoint["state_dict"])
        else:
            model.load_state_dict(checkpoint)
        
        return model
    
    def _create_model(self) -> torch.nn.Module:
        """创建模型架构"""
        class BSRoformerSCNet(torch.nn.Module):
            def __init__(self):
                super().__init__()
                # 简化的模型架构用于演示
                self.stem = torch.nn.Conv2d(2, 64, kernel_size=7, padding=3)
                self.encoder = torch.nn.Sequential(
                    torch.nn.Conv2d(64, 128, kernel_size=3, padding=1),
                    torch.nn.BatchNorm2d(128),
                    torch.nn.ReLU(),
                )
                self.decoder = torch.nn.Sequential(
                    torch.nn.ConvTranspose2d(128, 64, kernel_size=4, stride=2, padding=1),
                    torch.nn.BatchNorm2d(64),
                    torch.nn.ReLU(),
                )
                self.output = torch.nn.Conv2d(64, 2, kernel_size=7, padding=3)
            
            def forward(self, x):
                x = self.stem(x)
                x = self.encoder(x)
                x = self.decoder(x)
                x = self.output(x)
                return x
        
        return BSRoformerSCNet()
    
    async def separate(
        self,
        input_path: Path,
        output_dir: Path
    ) -> Dict[str, Path]:
        """
        分离音频文件
        
        Args:
            input_path: 输入音频文件路径
            output_dir: 输出目录
            
        Returns:
            分离后的文件路径字典
        """
        logger.info(f"开始分离：{input_path}")
        
        # 加载音频
        waveform, sample_rate = await self._load_audio(input_path)
        
        # 重采样
        if sample_rate != self.sample_rate:
            waveform = self._resample(waveform, sample_rate, self.sample_rate)
        
        # 转换为频谱图
        spectrogram = self._to_spectrogram(waveform)
        
        # 执行分离
        logger.info("执行音轨分离...")
        separated = await self._separate_spectrogram(spectrogram)
        
        # 转换回时域并保存
        output_files = {}
        
        # 人声
        if "vocals" in separated:
            vocals_waveform = self._from_spectrogram(separated["vocals"])
            vocals_path = output_dir / f"{input_path.stem}_vocals.wav"
            await self._save_audio(vocals_path, vocals_waveform)
            output_files["vocals"] = str(vocals_path)
            logger.info(f"人声已保存：{vocals_path}")
        
        # 伴奏
        if "instrumental" in separated:
            instrumental_waveform = self._from_spectrogram(separated["instrumental"])
            instrumental_path = output_dir / f"{input_path.stem}_instrumental.wav"
            await self._save_audio(instrumental_path, instrumental_waveform)
            output_files["instrumental"] = str(instrumental_path)
            logger.info(f"伴奏已保存：{instrumental_path}")
        
        return output_files
    
    async def _load_audio(self, path: Path) -> tuple:
        """加载音频文件"""
        waveform, sample_rate = await asyncio.to_thread(
            torchaudio.load, path
        )
        
        # 转换为立体声
        if waveform.shape[0] == 1:
            waveform = waveform.repeat(2, 1)
        
        logger.info(f"音频加载成功：{waveform.shape}, {sample_rate}Hz")
        return waveform, sample_rate
    
    def _resample(
        self,
        waveform: torch.Tensor,
        orig_sr: int,
        target_sr: int
    ) -> torch.Tensor:
        """重采样"""
        return torchaudio.transforms.Resample(orig_sr, target_sr)(waveform)
    
    def _to_spectrogram(self, waveform: torch.Tensor) -> torch.Tensor:
        """转换为频谱图"""
        n_fft = 2048
        hop_length = 512
        win_length = 2048
        
        spectrogram = torchaudio.transforms.Spectrogram(
            n_fft=n_fft,
            hop_length=hop_length,
            win_length=win_length,
            power=1
        )(waveform)
        
        return spectrogram
    
    async def _separate_spectrogram(
        self,
        spectrogram: torch.Tensor
    ) -> Dict[str, torch.Tensor]:
        """执行频谱图分离"""
        self.model.eval()
        
        with torch.no_grad():
            # 移动到设备
            spectrogram = spectrogram.to(self.device)
            
            # 添加 batch 维度
            if spectrogram.dim() == 3:
                spectrogram = spectrogram.unsqueeze(0)
            
            # 模型推理
            mask = self.model(spectrogram)
            
            # 应用掩码
            separated_mag = spectrogram * torch.sigmoid(mask)
            
            # 分离人声和伴奏
            vocals = separated_mag[:, 0:1]
            instrumental = separated_mag[:, 1:2]
            
            logger.info(f"分离完成：vocals={vocals.shape}, instrumental={instrumental.shape}")
            
            return {
                "vocals": vocals.squeeze(0),
                "instrumental": instrumental.squeeze(0)
            }
    
    def _from_spectrogram(self, spectrogram: torch.Tensor) -> torch.Tensor:
        """从频谱图转换回时域波形"""
        n_fft = 2048
        hop_length = 512
        win_length = 2048
        
        waveform = torchaudio.transforms.GriffinLim(
            n_fft=n_fft,
            hop_length=hop_length,
            win_length=win_length,
            power=1
        )(spectrogram)
        
        return waveform
    
    async def _save_audio(self, path: Path, waveform: torch.Tensor):
        """保存音频文件"""
        await asyncio.to_thread(
            torchaudio.save,
            path,
            waveform.cpu(),
            self.sample_rate
        )


# 全局分离器实例
separator = AudioSeparator()
