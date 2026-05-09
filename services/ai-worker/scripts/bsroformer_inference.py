#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
BS-RoFormer 音频分离推理脚本
支持主备模型切换：BS-RoFormer (主) + Demucs (备)

使用方法：
    python bsroformer_inference.py --input <input_file> --output <output_dir> [--model bsroformer|demucs] [--fallback]

依赖安装：
    pip install BS-RoFormer demucs torch torchaudio numpy
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path
from typing import Dict, Optional, Tuple, List

try:
    import torch
    import torchaudio
    import numpy as np
except ImportError as e:
    print(f"❌ 缺少必要的Python包: {e}")
    print("请执行: pip install torch torchaudio numpy")
    sys.exit(1)

# 设置环境变量解决 OpenMP 冲突
os.environ['KMP_DUPLICATE_LIB_OK'] = 'TRUE'


class BaseSeparator:
    """音频分离基类"""

    def __init__(self, model_name: str, device: str = "cpu"):
        self.model_name = model_name
        self.device = device
        self.model = None
        self.is_loaded = False

    def load(self) -> bool:
        """加载模型"""
        raise NotImplementedError

    def separate(self, input_path: str, output_dir: str) -> Dict[str, str]:
        """执行分离"""
        raise NotImplementedError

    def unload(self):
        """卸载模型"""
        if self.model is not None:
            del self.model
            if torch.cuda.is_available():
                torch.cuda.empty_cache()
            self.model = None
            self.is_loaded = False


class BSRoformerSeparator(BaseSeparator):
    """BS-RoFormer 分离器（主模型）"""

    def __init__(self, model_name: str = "lucidrains/bs-roformer-mel", device: str = "cpu", **kwargs):
        super().__init__(model_name, device)
        self.segment_size = kwargs.get('segment_size', 10)
        self.overlap = kwargs.get('overlap', 0.25)
        self.use_fp16 = kwargs.get('use_fp16', False)

    def load(self) -> bool:
        try:
            from bs_roformer import BSRoformer, MelBandRoformer

            print(f"📥 正在加载 BS-RoFormer 模型: {self.model_name}")

            # 尝试加载 Mel-Band RoFormer（推荐）
            try:
                if "mel" in self.model_name.lower():
                    self.model = MelBandRoformer.from_pretrained(self.model_name)
                else:
                    self.model = BSRoformer.from_pretrained(self.model_name)
            except Exception as e:
                print(f"⚠️ 从 HuggingFace 加载失败: {e}")
                print("尝试使用本地模型或默认配置...")
                # 使用默认配置创建模型（需要自行训练或下载权重）
                if "mel" in self.model_name.lower():
                    self.model = MelBandRoformer(
                        dim=32,
                        depth=1,
                        time_transformer_depth=1,
                        freq_transformer_depth=1
                    )
                else:
                    self.model = BSRoformer(
                        dim=512,
                        depth=12,
                        time_transformer_depth=1,
                        freq_transformer_depth=1
                    )
                print("⚠️ 使用未训练的模型（输出将无意义），请提供预训练权重")

            self.model.eval()

            if self.use_fp16 and self.device != "cpu":
                self.model = self.model.half()

            self.model = self.model.to(self.device)
            self.is_loaded = True

            print(f"✅ BS-RoFormer 模型加载成功")
            print(f"   模型类型: {type(self.model).__name__}")
            print(f"   设备: {self.device}")
            print(f"   FP16: {self.use_fp16}")

            return True

        except ImportError as e:
            print(f"❌ BS-RoFormer 未安装: {e}")
            print("请执行: pip install BS-RoFormer")
            return False
        except Exception as e:
            print(f"❌ 加载 BS-RoFormer 失败: {e}")
            return False

    def separate(self, input_path: str, output_dir: str) -> Dict[str, str]:
        if not self.is_loaded:
            raise RuntimeError("模型未加载")

        print(f"\n🎵 开始 BS-RoFormer 分离...")
        print(f"   输入文件: {input_path}")
        print(f"   输出目录: {output_dir}")

        start_time = time.time()

        # 加载音频
        waveform, sample_rate = torchaudio.load(input_path)

        # 转换为立体声（如果是单声道）
        if waveform.shape[0] == 1:
            waveform = waveform.repeat(2, 1)

        waveform = waveform.to(self.device)
        original_length = waveform.shape[-1]

        print(f"   音频时长: {original_length / sample_rate:.2f} 秒")
        print(f"   采样率: {sample_rate} Hz")

        # 执行推理
        with torch.no_grad():
            separated = self.model(waveform)

        inference_time = time.time() - start_time
        print(f"   推理耗时: {inference_time:.3f} 秒")

        # 创建输出目录
        os.makedirs(output_dir, exist_ok=True)

        # 保存结果
        tracks = {}
        track_names = ['vocals', 'drums', 'bass', 'other']

        if isinstance(separated, (list, tuple)):
            for i, stem in enumerate(separated[:len(track_names)]):
                track_name = track_names[i] if i < len(track_names) else f"track_{i}"
                output_path = os.path.join(output_dir, f"{track_name}.wav")
                torchaudio.save(output_path, stem.cpu(), sample_rate)
                tracks[track_name] = output_path
                size_mb = os.path.getsize(output_path) / (1024 * 1024)
                print(f"   ✅ {track_name}: {output_path} ({size_mb:.2f} MB)")
        elif isinstance(separated, torch.Tensor):
            # 单个输出（可能只有 vocals）
            output_path = os.path.join(output_dir, "vocals.wav")
            torchaudio.save(output_path, separated.cpu(), sample_rate)
            tracks['vocals'] = output_path
            print(f"   ✅ vocals: {output_path}")
        else:
            print(f"⚠️ 未知的输出格式: {type(separated)}")

        total_time = time.time() - start_time
        print(f"\n✅ BS-RoFormer 分离完成!")
        print(f"   总耗时: {total_time:.3f} 秒")
        print(f"   生成音轨: {len(tracks)} 个")

        return tracks


class DemucsSeparator(BaseSeparator):
    """Demucs 分离器（备用模型）"""

    def __init__(self, model_name: str = "htdemucs", device: str = "cpu", **kwargs):
        super().__init__(model_name, device)
        self.segment_size = kwargs.get('segment_size', 10)
        self.overlap = kwargs.get('overlap', 0.25)
        self.two_stems = kwargs.get('two_stems', 'vocals')

    def load(self) -> bool:
        try:
            import demucs.api

            print(f"📥 正在加载 Demucs 模型: {self.model_name}")
            self.model = demucs.api.Separator(model=self.model_name)
            self.is_loaded = True

            print(f"✅ Demucs 模型加载成功")
            print(f"   模型: {self.model_name}")
            return True

        except ImportError as e:
            print(f"❌ Demucs 未安装: {e}")
            print("请执行: pip install demucs")
            return False
        except Exception as e:
            print(f"❌ 加载 Demucs 失败: {e}")
            return False

    def separate(self, input_path: str, output_dir: str) -> Dict[str, str]:
        if not self.is_loaded:
            raise RuntimeError("模型未加载")

        print(f"\n🎵 开始 Demucs 分离（备用模式）...")
        print(f"   输入文件: {input_path}")
        print(f"   输出目录: {output_dir}")

        start_time = time.time()

        # 执行分离
        origin, separated = self.model.separate_two_stems(
            input_path,
            stems=self.two_stems,
            progress=True
        )

        inference_time = time.time() - start_time
        print(f"   推理耗时: {inference_time:.3f} 秒")

        # 创建输出目录
        os.makedirs(output_dir, exist_ok=True)

        # 保存结果
        tracks = {}

        for name, source in separated.items():
            output_path = os.path.join(output_dir, f"{name}.wav")
            torchaudio.save(output_path, source, self.model.samplerate)
            tracks[name] = output_path
            size_mb = os.path.getsize(output_path) / (1024 * 1024)
            print(f"   ✅ {name}: {output_path} ({size_mb:.2f} MB)")

        total_time = time.time() - start_time
        print(f"\n✅ Demucs 分离完成!")
        print(f"   总耗时: {total_time:.3f} 秒")
        print(f"   生成音轨: {len(tracks)} 个")

        return tracks


class AudioSeparationService:
    """音频分离服务 - 支持主备模型自动切换"""

    def __init__(
        self,
        primary_model: str = "bsroformer",
        fallback_model: str = "demucs",
        device: str = "cpu",
        **model_kwargs
    ):
        self.primary_model_name = primary_model
        self.fallback_model_name = fallback_model
        self.device = device
        self.model_kwargs = model_kwargs

        self.primary_model: Optional[BaseSeparator] = None
        self.fallback_model: Optional[BaseSeparator] = None

        self._initialize_models()

    def _initialize_models(self):
        """初始化主备模型"""

        # 初始化主模型
        if self.primary_model_name == "bsroformer":
            self.primary_model = BSRoformerSeparator(
                model_name=self.model_kwargs.get('bsroformer_model', 'lucidrains/bs-roformer-mel'),
                device=self.device,
                **{k: v for k, v in self.model_kwargs.items() if k != 'bsroformer_model'}
            )
        elif self.primary_model_name == "demucs":
            self.primary_model = DemucsSeparator(
                model_name=self.model_kwargs.get('demucs_model', 'htdemucs'),
                device=self.device,
                **{k: v for k, v in self.model_kwargs.items() if k != 'demucs_model'}
            )

        # 初始化备用模型
        if self.fallback_model_name == "demucs":
            self.fallback_model = DemucsSeparator(
                model_name="htdemucs",
                device=self.device
            )

    def load_primary_model(self) -> bool:
        """加载主模型"""
        if self.primary_model is None:
            print("❌ 主模型未初始化")
            return False

        print(f"\n{'='*60}")
        print(f"📦 加载主模型: {self.primary_model_name.upper()}")
        print(f"{'='*60}")

        success = self.primary_model.load()
        if success:
            print(f"✅ 主模型就绪")
        else:
            print(f"❌ 主模型加载失败，将使用备用模型")
        return success

    def load_fallback_model(self) -> bool:
        """加载备用模型"""
        if self.fallback_model is None:
            print("⚠️ 备用模型未配置")
            return False

        print(f"\n{'='*60}")
        print(f"📦 加载备用模型: {self.fallback_model_name.upper()}")
        print(f"{'='*60}")

        success = self.fallback_model.load()
        if success:
            print(f"✅ 备用模型就绪")
        return success

    def separate_with_fallback(
        self,
        input_path: str,
        output_dir: str,
        use_fallback: bool = False
    ) -> Tuple[Dict[str, str], str]:
        """
        执行分离（支持自动降级）

        返回: (tracks_dict, used_model_name)
        """
        # 选择使用的模型
        if use_fallback or self.primary_model is None or not self.primary_model.is_loaded:
            if self.fallback_model is None or not self.fallback_model.is_loaded:
                if not self.load_fallback_model():
                    raise RuntimeError("所有模型都不可用")
            model = self.fallback_model
            model_name = self.fallback_model_name
        else:
            model = self.primary_model
            model_name = self.primary_model_name

        try:
            tracks = model.separate(input_path, output_dir)
            return tracks, model_name
        except Exception as e:
            print(f"❌ {model_name} 分离失败: {e}")

            # 自动切换到备用模型
            if model != self.fallback_model and self.fallback_model is not None:
                print(f"\n🔄 自动切换到备用模型: {self.fallback_model_name}")
                if self.fallback_model.is_loaded or self.load_fallback_model():
                    tracks = self.fallback_model.separate(input_path, output_dir)
                    return tracks, self.fallback_model_name

            raise


def main():
    parser = argparse.ArgumentParser(
        description="BS-RoFormer 音频分离工具（支持主备模型切换）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例用法:
  # 使用 BS-RoFormer（主模型）
  python bsroformer_inference.py -i input.mp3 -o output/

  # 使用 Demucs（备用模型）
  python bsroformer_inference.py -i input.mp3 -o output/ --model demucs

  # 强制使用备用模型
  python bsroformer_inference.py -i input.mp3 -o output/ --fallback

  # GPU 加速
  python bsroformer_inference.py -i input.mp3 -o output/ --device cuda:0
        """
    )

    parser.add_argument("-i", "--input", required=True, help="输入音频文件路径")
    parser.add_argument("-o", "--output", required=True, help="输出目录")
    parser.add_argument("--model", choices=["bsroformer", "demucs"], default="bsroformer",
                        help="选择模型 (默认: bsroformer)")
    parser.add_argument("--fallback", action="store_true", help="强制使用备用模型")
    parser.add_argument("--device", default="cpu", help="计算设备 (默认: cpu, 可选 cuda:0)")
    parser.add_argument("--segment-size", type=int, default=10, help="分段大小（秒）")
    parser.add_argument("--overlap", type=float, default=0.25, help="重叠比例")
    parser.add_argument("--fp16", action="store_true", help="使用FP16精度")
    parser.add_argument("--two-stems", default="vocals", help="双音轨模式 (仅Demucs)")
    parser.add_argument("--json", action="store_true", help="输出JSON格式结果")

    args = parser.parse_args()

    # 验证输入文件
    if not os.path.exists(args.input):
        print(f"❌ 输入文件不存在: {args.input}")
        sys.exit(1)

    # 初始化服务
    service = AudioSeparationService(
        primary_model=args.model,
        fallback_model="demucs",
        device=args.device,
        segment_size=args.segment_size,
        overlap=args.overlap,
        use_fp16=args.fp16,
        two_stems=args.two_stems
    )

    # 加载主模型
    primary_loaded = service.load_primary_model()
    if not primary_loaded and not args.fallback:
        print("\n⚠️ 主模型加载失败，将使用备用模型（Demucs）")

    # 执行分离
    try:
        tracks, used_model = service.separate_with_fallback(
            input_path=args.input,
            output_dir=args.output,
            use_fallback=args.fallback
        )

        # 输出结果
        result = {
            "status": "success",
            "used_model": used_model,
            "input_file": args.input,
            "output_dir": args.output,
            "tracks": tracks,
            "track_count": len(tracks)
        }

        if args.json:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(f"\n{'='*60}")
            print(f"🎉 分离完成！")
            print(f"{'='*60}")
            print(f"使用的模型: {used_model}")
            print(f"生成音轨数: {len(tracks)}")
            for track_name, track_path in tracks.items():
                print(f"  • {track_name}: {track_path}")

        sys.exit(0)

    except Exception as e:
        error_result = {
            "status": "error",
            "error": str(e),
            "input_file": args.input
        }

        if args.json:
            print(json.dumps(error_result, indent=2, ensure_ascii=False))
        else:
            print(f"\n❌ 分离失败: {e}")

        sys.exit(1)


if __name__ == "__main__":
    main()
