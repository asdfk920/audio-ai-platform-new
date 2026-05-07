# PowerShell 测试脚本

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "BSRoformer SCNet 音轨分离服务 - 完整测试" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$BASE_URL = "http://localhost:8004"

# 测试 1: 健康检查
Write-Host "[测试 1] 健康检查" -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$BASE_URL/health" -Method GET
    Write-Host "OK 服务状态：$($health.status)" -ForegroundColor Green
    Write-Host "OK 模型：$($health.model_name)" -ForegroundColor Green
    Write-Host "OK 任务统计：$($health.tasks.total)" -ForegroundColor Green
} catch {
    Write-Host "X 健康检查失败：$_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 测试 2: 开始分离
Write-Host "[测试 2] 开始音轨分离" -ForegroundColor Yellow
$body = @{
    audio_url = "https://www2.cs.uic.edu/~i101/SoundFiles/BabyElephantWalk60.wav"
    user_id = "test_user"
    device_id = "test_device"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/api/v1/separate/start" -Method POST -Body $body -ContentType "application/json"
    Write-Host "OK 任务创建成功" -ForegroundColor Green
    Write-Host "  任务 ID: $($response.task_id)" -ForegroundColor Cyan
    Write-Host "  状态：$($response.data.status)" -ForegroundColor Cyan
    
    $task_id = $response.task_id
} catch {
    Write-Host "X 创建任务失败：$_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 测试 3: 查询状态
Write-Host "[测试 3] 查询任务状态" -ForegroundColor Yellow
$max_attempts = 30
$poll_interval = 2

for ($i = 1; $i -le $max_attempts; $i++) {
    try {
        $status = Invoke-RestMethod -Uri "$BASE_URL/api/v1/separate/status/$task_id" -Method GET
        
        Write-Host "  [$i/$max_attempts] 状态：$($status.status)" -NoNewline
        
        if ($status.status -eq "completed") {
            Write-Host " OK" -ForegroundColor Green
            Write-Host "  处理时间：$($status.processing_time) 秒" -ForegroundColor Cyan
            
            if ($status.output_files) {
                Write-Host "  输出文件:" -ForegroundColor Cyan
                Write-Host "    - vocals: $($status.output_files.vocals)" -ForegroundColor Cyan
                Write-Host "    - instrumental: $($status.output_files.instrumental)" -ForegroundColor Cyan
            }
            break
        } elseif ($status.status -eq "failed") {
            Write-Host " X" -ForegroundColor Red
            Write-Host "  错误：$($status.error_message)" -ForegroundColor Red
            break
        } else {
            $pct = $status.progress.percentage
            Write-Host " ($pct%)" -ForegroundColor Gray
        }
        
        Start-Sleep -Seconds $poll_interval
    } catch {
        Write-Host "X 查询失败：$_" -ForegroundColor Red
        break
    }
}
Write-Host ""

# 测试 4: 下载结果
Write-Host "[测试 4] 下载分离结果" -ForegroundColor Yellow
try {
    $status = Invoke-RestMethod -Uri "$BASE_URL/api/v1/separate/status/$task_id" -Method GET
    
    if ($status.status -eq "completed") {
        # 下载人声
        $vocals_url = "$BASE_URL/api/v1/separate/download/$task_id/vocals"
        Invoke-WebRequest -Uri $vocals_url -OutFile "test_vocals.wav"
        Write-Host "OK 人声下载成功：test_vocals.wav" -ForegroundColor Green
        
        # 下载伴奏
        $instrumental_url = "$BASE_URL/api/v1/separate/download/$task_id/instrumental"
        Invoke-WebRequest -Uri $instrumental_url -OutFile "test_instrumental.wav"
        Write-Host "OK 伴奏下载成功：test_instrumental.wav" -ForegroundColor Green
    } else {
        Write-Host "- 任务未完成，跳过下载" -ForegroundColor Yellow
    }
} catch {
    Write-Host "X 下载失败：$_" -ForegroundColor Red
}
Write-Host ""

# 测试 5: 列出任务
Write-Host "[测试 5] 列出所有任务" -ForegroundColor Yellow
try {
    $tasks = Invoke-RestMethod -Uri "$BASE_URL/api/v1/tasks" -Method GET
    Write-Host "OK 任务总数：$($tasks.count)" -ForegroundColor Green
    
    if ($tasks.tasks.Count -gt 0) {
        Write-Host "  最近任务:" -ForegroundColor Cyan
        $tasks.tasks | Select-Object -First 5 | ForEach-Object {
            Write-Host "    - $($_.task_id): $($_.status)" -ForegroundColor Cyan
        }
    }
} catch {
    Write-Host "X 列出任务失败：$_" -ForegroundColor Red
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "测试完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "服务信息:" -ForegroundColor Yellow
Write-Host "  API 地址：$BASE_URL" -ForegroundColor Cyan
Write-Host "  文档地址：$BASE_URL/docs" -ForegroundColor Cyan
Write-Host "  健康检查：$BASE_URL/health" -ForegroundColor Cyan
