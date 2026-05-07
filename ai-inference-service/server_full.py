"""
BSRoformer SCNet 音轨分离服务 - 生产版本
支持设备端上传音频链接进行云端处理
此版本使用 mock 数据，不依赖深度学习框架
"""

import os
import sys
import asyncio
from pathlib import Path
from datetime import datetime
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException, Header, Depends
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, FileResponse
from loguru import logger
import uvicorn
import jwt
from datetime import datetime, timedelta
from typing import Optional

# 配置日志
logger.remove()
logger.add(
    sys.stderr,
    level="INFO",
    format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>"
)


# 模拟数据存储器
tasks_db = {}

# JWT 密钥 - 使用用户服务的密钥来验证 token
# 从用户服务配置获取：services/user/etc/user-api.yaml
JWT_SECRET_KEY = "audio-ai-platform-secret-key-2024"
JWT_ALGORITHM = "HS256"


def verify_token(authorization: Optional[str] = Header(None)):
    """
    验证 JWT token
    
    从 Authorization header 中提取 token 并验证
    格式：Bearer <token>
    
    验证用户服务生成的 token，支持以下字段：
    - userId: 用户 ID (用户服务标准字段)
    - user_id: 用户 ID (兼容字段)
    """
    if not authorization:
        logger.warning("缺少认证信息")
        raise HTTPException(
            status_code=401,
            detail="缺少认证信息，请在 Header 中添加 Authorization: Bearer <token>"
        )
    
    # 提取 token
    try:
        scheme, token = authorization.split()
        if scheme.lower() != "bearer":
            logger.warning(f"认证格式错误：{scheme}")
            raise HTTPException(
                status_code=401,
                detail="认证格式错误，请使用 Bearer token"
            )
    except ValueError as e:
        logger.warning(f"认证格式错误：{authorization[:20]}...")
        raise HTTPException(
            status_code=401,
            detail=f"认证格式错误，请使用 Bearer <token> 格式。当前值：{authorization[:30]}..."
        )
    
    # 验证 token（使用用户服务的密钥）
    try:
        # 首先尝试不解密验证（只验证签名）
        payload = jwt.decode(token, JWT_SECRET_KEY, algorithms=[JWT_ALGORITHM])
        
        # 支持 userId 和 user_id 两种字段
        user_id = payload.get("userId") or payload.get("user_id")
        
        if user_id is None:
            logger.warning(f"Token 缺少用户 ID 字段：{token[:20]}...")
            raise HTTPException(
                status_code=401,
                detail="Token 无效，缺少用户 ID 字段 (userId 或 user_id)"
            )
        
        logger.info(f"Token 验证成功，用户：{user_id}")
        return str(user_id)  # 转换为字符串以便统一处理
        
    except jwt.ExpiredSignatureError:
        logger.warning("Token 已过期")
        raise HTTPException(
            status_code=401,
            detail="Token 已过期"
        )
    except jwt.InvalidTokenError as e:
        logger.error(f"Token 无效：{type(e).__name__}, token: {token[:20]}...")
        raise HTTPException(
            status_code=401,
            detail=f"Token 无效 ({type(e).__name__})。请确保使用用户服务生成的 token。错误：{str(e)}"
        )


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    logger.info("正在启动 BSRoformer SCNet 音轨分离服务...")
    
    try:
        # 确保目录存在
        Path("data/input").mkdir(parents=True, exist_ok=True)
        Path("data/output").mkdir(parents=True, exist_ok=True)
        
        logger.info("服务初始化完成")
        
    except Exception as e:
        logger.error(f"服务初始化失败：{e}")
        raise
    
    yield
    
    logger.info("正在关闭服务...")


app = FastAPI(
    title="BSRoformer SCNet 音轨分离服务",
    description="基于 BSRoformer SCNet 模型的云端音轨分离服务，支持设备端上传音频链接进行处理",
    version="2.0.0",
    lifespan=lifespan
)

# 配置 CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/")
async def root():
    """根路径"""
    return {
        "service": "BSRoformer SCNet Audio Separation Service",
        "version": "2.0.0",
        "status": "running",
        "endpoints": {
            "health": "/health",
            "start_separation": "/api/v1/separate/start",
            "task_status": "/api/v1/separate/status/{task_id}",
            "download": "/api/v1/separate/download/{task_id}/{stem}"
        },
        "auth": {
            "type": "Bearer Token",
            "description": "使用用户服务生成的 JWT token",
            "header": "Authorization: Bearer <token>"
        }
    }


@app.get("/health")
async def health_check():
    """健康检查"""
    return {
        "status": "healthy",
        "model_name": "bsroformer_scnet",
        "device": "cpu",
        "gpu_available": False,
        "tasks": {
            "total": len(tasks_db),
            "pending": len([t for t in tasks_db.values() if t["status"] == "pending"]),
            "processing": len([t for t in tasks_db.values() if t["status"] == "processing"]),
            "completed": len([t for t in tasks_db.values() if t["status"] == "completed"]),
            "failed": len([t for t in tasks_db.values() if t["status"] == "failed"])
        }
    }


@app.post("/api/v1/separate/start")
async def start_separation(
    request: dict,
    current_user_id: str = Depends(verify_token)
):
    """
    开始音轨分离任务
    
    设备端调用此接口，上传音频链接到云端处理
    
    **认证:**
    - Header: Authorization: Bearer <token>
    
    **请求参数:**
    - audio_url: 音频文件 URL（必填）
    - device_id: 设备 ID（可选）
    
    **返回:**
    - task_id: 任务 ID，用于查询进度
    - status: 任务状态
    """
    try:
        audio_url = request.get("audio_url")
        device_id = request.get("device_id")
        callback_url = request.get("callback_url")
        
        if not audio_url:
            raise HTTPException(status_code=400, detail="audio_url 不能为空")
        
        # 从 token 中获取 user_id
        user_id = current_user_id
        
        # 生成任务 ID
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        import uuid
        task_id = f"task_{timestamp}_{str(uuid.uuid4())[:8]}"
        
        # 创建任务
        task = {
            "task_id": task_id,
            "user_id": user_id,
            "device_id": device_id,
            "audio_url": audio_url,
            "status": "pending",
            "progress": {
                "percentage": 0,
                "stage": "pending",
                "message": "等待处理"
            },
            "output_files": None,
            "error_message": None,
            "created_at": datetime.now().isoformat(),
            "started_at": None,
            "completed_at": None,
            "processing_time": None
        }
        
        tasks_db[task_id] = task
        
        logger.info(f"创建任务：{task_id}, 用户：{user_id}")
        
        # 异步处理任务
        asyncio.create_task(process_task(task_id, callback_url))
        
        return {
            "success": True,
            "task_id": task_id,
            "message": "任务创建成功，正在处理",
            "data": task
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"创建任务失败：{e}")
        raise HTTPException(status_code=500, detail=f"创建任务失败：{str(e)}")


async def process_task(task_id: str, callback_url: str = None):
    """处理任务（模拟）"""
    try:
        task = tasks_db[task_id]
        
        # 1. 下载音频文件
        task["status"] = "downloading"
        task["progress"] = {
            "percentage": 10,
            "stage": "downloading",
            "message": "正在下载音频文件"
        }
        task["started_at"] = datetime.now().isoformat()
        logger.info(f"任务 {task_id}: 下载中...")
        
        await asyncio.sleep(2)  # 模拟下载
        
        # 2. 执行音轨分离
        task["status"] = "processing"
        task["progress"] = {
            "percentage": 50,
            "stage": "processing",
            "message": "正在分离音轨"
        }
        logger.info(f"任务 {task_id}: 处理中...")
        
        await asyncio.sleep(3)  # 模拟处理
        
        # 3. 上传结果
        task["status"] = "uploading"
        task["progress"] = {
            "percentage": 80,
            "stage": "uploading",
            "message": "正在上传结果"
        }
        logger.info(f"任务 {task_id}: 上传中...")
        
        await asyncio.sleep(1)  # 模拟上传
        
        # 4. 完成任务
        task["status"] = "completed"
        task["progress"] = {
            "percentage": 100,
            "stage": "completed",
            "message": "分离完成"
        }
        task["completed_at"] = datetime.now().isoformat()
        
        # 计算处理时间
        started = datetime.fromisoformat(task["started_at"])
        completed = datetime.fromisoformat(task["completed_at"])
        task["processing_time"] = (completed - started).total_seconds()
        
        # 模拟输出文件
        task["output_files"] = {
            "vocals": f"data/output/{task_id}/{task_id}_vocals.wav",
            "instrumental": f"data/output/{task_id}/{task_id}_instrumental.wav"
        }
        
        logger.info(f"任务 {task_id}: 完成，耗时 {task['processing_time']:.2f}秒")
        
    except Exception as e:
        logger.error(f"任务 {task_id} 失败：{e}")
        task = tasks_db[task_id]
        task["status"] = "failed"
        task["error_message"] = str(e)
        task["completed_at"] = datetime.now().isoformat()


@app.get("/api/v1/separate/status/{task_id}")
async def get_task_status(task_id: str):
    """查询任务状态"""
    task = tasks_db.get(task_id)
    
    if not task:
        raise HTTPException(status_code=404, detail=f"任务不存在：{task_id}")
    
    return {
        "success": True,
        "task_id": task["task_id"],
        "status": task["status"],
        "progress": task["progress"],
        "output_files": task.get("output_files"),
        "error_message": task.get("error_message"),
        "processing_time": task.get("processing_time")
    }


@app.get("/api/v1/separate/download/{task_id}/{stem}")
async def download_result(task_id: str, stem: str):
    """下载分离结果"""
    task = tasks_db.get(task_id)
    
    if not task:
        raise HTTPException(status_code=404, detail=f"任务不存在：{task_id}")
    
    if task["status"] != "completed":
        raise HTTPException(
            status_code=400,
            detail=f"任务未完成，当前状态：{task['status']}"
        )
    
    # 创建模拟的 WAV 文件
    output_dir = Path(f"data/output/{task_id}")
    output_dir.mkdir(parents=True, exist_ok=True)
    
    file_path = output_dir / f"{task_id}_{stem}.wav"
    
    if not file_path.exists():
        # 创建一个简单的 WAV 文件头（空的音频文件用于测试）
        import struct
        
        # WAV 文件头
        wav_header = bytearray()
        wav_header.extend(b'RIFF')
        wav_header.extend(struct.pack('<I', 36))  # 文件大小
        wav_header.extend(b'WAVE')
        wav_header.extend(b'fmt ')
        wav_header.extend(struct.pack('<I', 16))  # fmt chunk 大小
        wav_header.extend(struct.pack('<H', 1))   # PCM 格式
        wav_header.extend(struct.pack('<H', 2))   # 双声道
        wav_header.extend(struct.pack('<I', 44100))  # 采样率
        wav_header.extend(struct.pack('<I', 176400)) # 字节率
        wav_header.extend(struct.pack('<H', 4))   # 块对齐
        wav_header.extend(struct.pack('<H', 16))  # 位深度
        wav_header.extend(b'data')
        wav_header.extend(struct.pack('<I', 0))   # 数据大小（0 表示空文件）
        
        with open(file_path, 'wb') as f:
            f.write(wav_header)
    
    return FileResponse(
        path=file_path,
        filename=f"{task_id}_{stem}.wav",
        media_type="audio/wav"
    )


@app.post("/api/v1/separate/{task_id}/cancel")
async def cancel_task(task_id: str):
    """取消任务"""
    task = tasks_db.get(task_id)
    
    if not task:
        raise HTTPException(status_code=404, detail=f"任务不存在：{task_id}")
    
    if task["status"] in ["completed", "failed"]:
        raise HTTPException(
            status_code=400,
            detail="无法取消任务（可能已完成或不存在）"
        )
    
    task["status"] = "cancelled"
    task["error_message"] = "用户取消"
    task["completed_at"] = datetime.now().isoformat()
    
    return {
        "success": True,
        "message": "任务已取消"
    }


@app.get("/api/v1/tasks")
async def list_tasks(status: str = None):
    """列出所有任务"""
    if status:
        tasks = [t for t in tasks_db.values() if t["status"] == status]
    else:
        tasks = list(tasks_db.values())
    
    return {
        "success": True,
        "count": len(tasks),
        "tasks": tasks
    }


if __name__ == "__main__":
    uvicorn.run(
        "server_full:app",
        host="0.0.0.0",
        port=8004,
        reload=False,
        log_level="info"
    )
