"""
设备端调用示例
演示如何调用 AI 推理服务进行音轨分离
"""

import requests
import time
from typing import Optional


class AudioSeparationClient:
    """音轨分离客户端"""
    
    def __init__(self, base_url: str = "http://localhost:8004"):
        self.base_url = base_url
    
    def start_separation(
        self,
        audio_url: str,
        user_id: Optional[str] = None,
        device_id: Optional[str] = None,
        callback_url: Optional[str] = None
    ) -> dict:
        """
        开始音轨分离
        
        Args:
            audio_url: 音频文件 URL
            user_id: 用户 ID
            device_id: 设备 ID
            callback_url: 回调 URL
            
        Returns:
            包含 task_id 的响应
        """
        url = f"{self.base_url}/api/v1/separate/start"
        
        payload = {
            "audio_url": audio_url,
            "user_id": user_id,
            "device_id": device_id,
            "callback_url": callback_url
        }
        
        response = requests.post(url, json=payload, timeout=30)
        response.raise_for_status()
        
        return response.json()
    
    def get_task_status(self, task_id: str) -> dict:
        """
        查询任务状态
        
        Args:
            task_id: 任务 ID
            
        Returns:
            任务状态信息
        """
        url = f"{self.base_url}/api/v1/separate/status/{task_id}"
        
        response = requests.get(url, timeout=30)
        response.raise_for_status()
        
        return response.json()
    
    def download_result(self, task_id: str, stem: str, save_path: str) -> bool:
        """
        下载分离结果
        
        Args:
            task_id: 任务 ID
            stem: 音轨类型 (vocals/instrumental)
            save_path: 保存路径
            
        Returns:
            下载是否成功
        """
        url = f"{self.base_url}/api/v1/separate/download/{task_id}/{stem}"
        
        response = requests.get(url, timeout=60, stream=True)
        response.raise_for_status()
        
        with open(save_path, 'wb') as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
        
        return True
    
    def wait_for_completion(
        self,
        task_id: str,
        timeout: int = 300,
        poll_interval: int = 2
    ) -> dict:
        """
        等待任务完成
        
        Args:
            task_id: 任务 ID
            timeout: 超时时间（秒）
            poll_interval: 轮询间隔（秒）
            
        Returns:
            最终任务状态
        """
        start_time = time.time()
        
        while True:
            # 检查超时
            if time.time() - start_time > timeout:
                raise TimeoutError(f"任务超时：{task_id}")
            
            # 查询状态
            status = self.get_task_status(task_id)
            current_status = status.get("status")
            
            print(f"任务状态：{current_status}, 进度：{status.get('progress', {}).get('percentage', 0)}%")
            
            # 检查是否完成
            if current_status in ["completed", "failed", "timeout", "cancelled"]:
                return status
            
            # 等待
            time.sleep(poll_interval)


def example_usage():
    """使用示例"""
    # 创建客户端
    client = AudioSeparationClient("http://localhost:8004")
    
    # 1. 开始分离
    print("=" * 60)
    print("步骤 1: 开始音轨分离")
    print("=" * 60)
    
    response = client.start_separation(
        audio_url="https://example.com/audio/song.mp3",
        user_id="user_123",
        device_id="device_456"
    )
    
    print(f"任务创建成功：{response}")
    task_id = response.get("task_id")
    
    if not task_id:
        print("任务创建失败")
        return
    
    print(f"任务 ID: {task_id}")
    
    # 2. 等待完成
    print("\n" + "=" * 60)
    print("步骤 2: 等待任务完成")
    print("=" * 60)
    
    try:
        final_status = client.wait_for_completion(task_id, timeout=300)
        
        if final_status.get("status") == "completed":
            print("\n任务完成！")
            print(f"处理时间：{final_status.get('processing_time', 0):.2f}秒")
            
            # 3. 下载结果
            print("\n" + "=" * 60)
            print("步骤 3: 下载分离结果")
            print("=" * 60)
            
            output_files = final_status.get("output_files", {})
            
            for stem, file_path in output_files.items():
                save_path = f"{task_id}_{stem}.wav"
                print(f"下载 {stem}: {save_path}")
                client.download_result(task_id, stem, save_path)
            
            print("\n所有文件下载完成！")
            
        else:
            print(f"\n任务失败：{final_status.get('error_message', '未知错误')}")
    
    except TimeoutError as e:
        print(f"\n错误：{e}")
    except Exception as e:
        print(f"\n错误：{e}")


if __name__ == "__main__":
    example_usage()
