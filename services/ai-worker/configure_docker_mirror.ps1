# Auto-configure Docker Registry Mirror Script

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Docker Registry Mirror Configuration" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
try {
    $dockerStatus = docker info 2>&1
    Write-Host "[OK] Docker is running" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Docker is not running. Please start Docker Desktop first." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Current Docker configuration:" -ForegroundColor Yellow
docker info | Select-String "Registry Mirrors"

Write-Host ""
Write-Host "Available registry mirrors:" -ForegroundColor Cyan
Write-Host "  1. https://hub-mirror.c.163.com (NetEase)"
Write-Host "  2. https://mirror.baidubce.com (Baidu)"
Write-Host "  3. https://docker.m.daocloud.io (DaoCloud)"
Write-Host "  4. https://docker.1panel.live (1Panel)"
Write-Host ""

# Ask user for choice
Write-Host "Please select configuration method:" -ForegroundColor Yellow
Write-Host "  1. Auto-configure multiple mirrors (Recommended)"
Write-Host "  2. Enter custom mirror URL"
Write-Host "  3. Skip configuration and build directly"
Write-Host ""

$choice = Read-Host "Enter choice (1-3)"

if ($choice -eq "1") {
    Write-Host "`nConfiguring registry mirrors..." -ForegroundColor Yellow
    
    # Docker Desktop config path
    $configPath = "$env:USERPROFILE\.docker\desktop\settings.json"
    
    # Check if config file exists
    if (Test-Path $configPath) {
        # Read existing config
        $config = Get-Content $configPath -Raw | ConvertFrom-Json
        
        # Add registry mirrors
        $config."registry-mirrors" = @(
            "https://hub-mirror.c.163.com",
            "https://mirror.baidubce.com",
            "https://docker.m.daocloud.io",
            "https://docker.1panel.live"
        )
        
        # Save config
        $config | ConvertTo-Json -Depth 10 | Set-Content $configPath
        Write-Host "[OK] Configuration saved" -ForegroundColor Green
        Write-Host "[INFO] Please restart Docker Desktop to apply changes" -ForegroundColor Yellow
        
        # Ask to restart Docker
        $restart = Read-Host "Restart Docker Desktop now? (y/n)"
        if ($restart -eq "y") {
            Write-Host "Restarting Docker Desktop..." -ForegroundColor Yellow
            Restart-Computer -Force
        }
    } else {
        Write-Host "[ERROR] Docker Desktop configuration file not found" -ForegroundColor Red
        Write-Host "  Please open Docker Desktop Settings to configure manually" -ForegroundColor Yellow
        Write-Host "  Config path: $configPath" -ForegroundColor Gray
    }
}
elseif ($choice -eq "2") {
    $mirrorUrl = Read-Host "Enter registry mirror URL"
    
    if ($mirrorUrl) {
        $configPath = "$env:USERPROFILE\.docker\desktop\settings.json"
        
        if (Test-Path $configPath) {
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config."registry-mirrors" = @($mirrorUrl)
            $config | ConvertTo-Json -Depth 10 | Set-Content $configPath
            Write-Host "[OK] Configuration saved" -ForegroundColor Green
            Write-Host "[INFO] Please restart Docker Desktop" -ForegroundColor Yellow
        } else {
            Write-Host "[ERROR] Configuration file not found" -ForegroundColor Red
        }
    }
}
elseif ($choice -eq "3") {
    Write-Host "Skipping configuration..." -ForegroundColor Yellow
}
else {
    Write-Host "Invalid option" -ForegroundColor Red
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Next Step: Build Docker Image" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Ask if user wants to build now
$build = Read-Host "Build Docker image now? (y/n)"
if ($build -eq "y") {
    Write-Host "`nStarting build..." -ForegroundColor Yellow
    Set-Location $PSScriptRoot
    docker build -t ai-worker:v1 .
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n[SUCCESS] Image built successfully!" -ForegroundColor Green
        Write-Host "`nStart container:" -ForegroundColor Cyan
        Write-Host "  docker run -d --name ai-worker --gpus all -p 8004:8004 ai-worker:v1" -ForegroundColor Gray
    } else {
        Write-Host "`n[FAILED] Image build failed" -ForegroundColor Red
        Write-Host "  Please check network connection or configure registry mirrors" -ForegroundColor Yellow
    }
} else {
    Write-Host "`nYou can build manually later:" -ForegroundColor Cyan
    Write-Host "  cd $PSScriptRoot" -ForegroundColor Gray
    Write-Host "  docker build -t ai-worker:v1 ." -ForegroundColor Gray
}

Write-Host ""
