# 从各服务的 *.api 生成 Swagger 2.0 JSON（与 Makefile swagger-* 等价，便于 Windows 无 make 时使用）
$ErrorActionPreference = "Stop"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

function Invoke-Swagger {
    param([string]$ApiRel, [string]$OutDir, [string]$FileBase)
    $api = Join-Path $root $ApiRel
    $dir = Join-Path $root $OutDir
    if (-not (Test-Path $api)) { throw "API file not found: $api" }
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Write-Host "swagger: $ApiRel -> $OutDir/${FileBase}.json"
    & go run github.com/zeromicro/go-zero/tools/goctl@latest api swagger --api $api --dir $dir --filename $FileBase
    if ($LASTEXITCODE -ne 0) { throw "goctl swagger failed for $ApiRel" }
}

Invoke-Swagger "services/user/user.api" "doc/swagger/user" "user"
Invoke-Swagger "services/device/device.api" "doc/swagger/device" "device"
Invoke-Swagger "services/content/content.api" "doc/swagger/content" "content"

function Ensure-ToolInPath {
    param([string]$ExeName, [string]$InstallCmd)
    $cmd = Get-Command $ExeName -ErrorAction SilentlyContinue
    if ($cmd) { return }
    Write-Host "Installing $ExeName ..."
    iex $InstallCmd
    $cmd2 = Get-Command $ExeName -ErrorAction SilentlyContinue
    if (-not $cmd2) { throw "$ExeName not found after install" }
}

# go-admin swagger (swaggo/swag)
Ensure-ToolInPath "swag" "go install github.com/swaggo/swag/cmd/swag@latest"
Push-Location (Join-Path $root "admin")
Write-Host "swagger: admin -> admin/docs/admin/admin_swagger.json"
& swag init --parseDependency --parseDepth=6 --instanceName admin -o ./docs/admin
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "swag init failed for admin" }
Pop-Location

$adminOutDir = Join-Path $root "doc/swagger/admin"
New-Item -ItemType Directory -Force -Path $adminOutDir | Out-Null
Copy-Item -Force (Join-Path $root "admin/docs/admin/admin_swagger.json") (Join-Path $adminOutDir "admin.json")
Copy-Item -Force (Join-Path $root "admin/docs/admin/admin_swagger.yaml") (Join-Path $adminOutDir "admin.yaml")

Write-Host "Done. Outputs: doc/swagger/{user,device,content,admin}/*.{json,yaml}"
