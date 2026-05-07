"""
BSRoformer SCNet 音轨分离服务 - 完整版
支持设备端上传音频链接进行云端处理
"""

import os
import sys
from pathlib import Path
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from loguru import logger
import uvicorn

from src.config import settings
from src.models import (
    StartSeparationRequest,
    TaskResponse,
    TaskStatusResponse,
    SeparationTask
)
from src.service import separation_service
from src.separator import separator
from src.task_manager import task_manager


# 配置日志
logger.remove()
logger.add(
    sys.stderr,
    level=settings.log_level,
    format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <cyan>{name}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>"
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    # 启动时初始化
    logger.info("正在启动 BSRoformer SCNet 音轨分离服务...")
    
    try:
        # 确保目录存在
        Path(settings.input_path).mkdir(parents=True, exist_ok=True)
        Path(settings.output_path).mkdir(parents=True, exist_ok=True)
        
        # 加载模型
        separator.load_model(settings.model_path)
        
        logger.info("服务初始化完成")
        
    except Exception as e:
        logger.error(f"服务初始化失败：{e}")
        raise
    
    yield
    
    # 关闭时清理
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
        }
    }


@app.get("/health")
async def health_check():
    """健康检查"""
    stats = task_manager.get_stats()
    
    return {
        "status": "healthy",
        "model_name": settings.model_name,
        "device": str(separator.device),
        "gpu_available": separator.device.type == "cuda",
        "tasks": stats
    }


@app.post("/api/v1/separate/start", response_model=TaskResponse)
async def start_separation(request: StartSeparationRequest):
    """
    开始音轨分离任务
    
    设备端调用此接口，上传音频链接到云端处理
    
    **请求参数:**
    - audio_url: 音频文件 URL（必填）
    - user_id: 用户 ID（可选）
    - device_id: 设备 ID（可选）
    - callback_url: 回调 URL（可选）
    
    **返回:**
    - task_id: 任务 ID，用于查询进度
    - status: 任务状态
    """
    try:
        logger.info(f"收到分离请求：{request.audio_url}")
        
        # 创建分离任务
        task = await separation_service.start_separation(
            audio_url=request.audio_url,
            user_id=request.user_id,
            device_id=request.device_id,
            callback_url=request.callback_url
        )
        
        logger.info(f"任务创建成功：{task.task_id}")
        
        return TaskResponse(
            success=True,
            task_id=task.task_id,
            message="任务创建成功，正在处理",
            data=task
        )
        
    except ValueError as e:
        logger.error(f"参数错误：{e}")
        raise HTTPException(status_code=400, detail=str(e))
    
    except Exception as e:
        logger.error(f"创建任务失败：{e}")
        raise HTTPException(status_code=500, detail=f"创建任务失败：{str(e)}")


@app.get("/api/v1/separate/status/{task_id}", response_model=TaskStatusResponse)
async def get_task_status(task_id: str):
    """
    查询任务状态
    
    **路径参数:**
    - task_id: 任务 ID
    
    **返回:**
    - status: 任务状态 (pending/downloading/processing/uploading/completed/failed)
    - progress: 进度百分比
    - output_files: 输出文件路径（完成后）
    """
    task = await separation_service.get_task_status(task_id)
    
    if not task:
        raise HTTPException(status_code=404, detail=f"任务不存在：{task_id}")
    
    return TaskStatusResponse(
        success=True,
        task_id=task.task_id,
        status=task.status,
        progress=task.progress,
        output_files=task.output_files,
        error_message=task.error_message,
        processing_time=task.processing_time
    )


@app.get("/api/v1/separate/download/{task_id}/{stem}")
async def download_result(task_id: str, stem: str):
    """
    下载分离结果
    
    **路径参数:**
    - task_id: 任务 ID
    - stem: 音轨类型 (vocals/instrumental)
    
    **返回:**
    - 音频文件
    """
    from fastapi.responses import FileResponse
    
    task = await separation_service.get_task_status(task_id)
    
    if not task:
        raise HTTPException(status_code=404, detail=f"任务不存在：{task_id}")
    
    if task.status != "completed":
        raise HTTPException(
            status_code=400,
            detail=f"任务未完成，当前状态：{task.status}"
        )
    
    if not task.output_files or stem not in task.output_files:
        raise HTTPException(
            status_code=404,
            detail=f"未找到音轨文件：{stem}"
        )
    
    file_path = task.output_files[stem]
    
    if not Path(file_path).exists():
        raise HTTPException(status_code=404, detail="文件不存在")
    
    return FileResponse(
        path=file_path,
        filename=f"{task_id}_{stem}.wav",
        media_type="audio/wav"
    )


@app.post("/api/v1/separate/{task_id}/cancel")
async def cancel_task(task_id: str):
    """取消任务"""
    success = await separation_service.cancel_task(task_id)
    
    if not success:
        raise HTTPException(
            status_code=400,
            detail="无法取消任务（可能已完成或不存在）"
        )
    
    return {
        "success": True,
        "message": "任务已取消"
    }


@app.get("/api/v1/tasks")
async def list_tasks(status: str = None):
    """列出所有任务"""
    from src.models import TaskStatus
    
    if status:
        try:
            task_status = TaskStatus(status)
            tasks = await task_manager.list_tasks(task_status)
        except ValueError:
            raise HTTPException(
                status_code=400,
                detail=f"无效的状态：{status}"
            )
    else:
        tasks = await task_manager.list_tasks()
    
    return {
        "success": True,
        "count": len(tasks),
        "tasks": tasks
    }


@app.exception_handler(HTTPException)
async def http_exception_handler(request, exc):
    """HTTP 异常处理"""
    logger.error(f"HTTP 错误：{exc.status_code} - {exc.detail}")
    return JSONResponse(
        status_code=exc.status_code,
        content={
            "success": False,
            "error": exc.detail
        }
    )


@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    """通用异常处理"""
    logger.error(f"服务器错误：{exc}")
    return JSONResponse(
        status_code=500,
        content={
            "success": False,
            "error": "服务器内部错误"
        }
    )


if __name__ == "__main__":
    uvicorn.run(
        "server:app",
        host=settings.host,
        port=settings.port,
        reload=False,
        log_level=settings.log_level.lower()
    )
