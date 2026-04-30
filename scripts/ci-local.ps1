# 本地执行与 .github/workflows/ci.yml 中 lint + test + build 相近的步骤。
# 使用根目录 .golangci.yml；lint 前同步根模块，与子模块 replace 对齐。
# 用法:
#   Windows (PowerShell):  .\scripts\ci-local.ps1
#   Linux / macOS / Git Bash:  bash scripts/ci-local.sh

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $root

$env:GOTOOLCHAIN = "local"

$golangci = Join-Path (go env GOPATH) "bin\golangci-lint.exe"
if (-not (Test-Path $golangci)) {
    Write-Host "未找到 golangci-lint: $golangci" -ForegroundColor Red
    Write-Host "请安装与 CI 相同版本: go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8" -ForegroundColor Yellow
    exit 1
}

$golangciYml = Join-Path $root ".golangci.yml"
if (-not (Test-Path $golangciYml)) {
    Write-Host ('缺少 ' + $golangciYml + ' (应与 CI 共用)') -ForegroundColor Red
    exit 1
}

$services = @(
    "services/user",
    "services/device",
    "services/content",
    "services/media-processing"
)

# ---------- 根模块（platform）----------
Write-Host ''
Write-Host '=== Root: go mod download / verify ===' -ForegroundColor Cyan
go mod download
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go mod verify
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ''
Write-Host '=== Check DB migrations (*_down.sql paired) ===' -ForegroundColor Cyan
# 使用 PowerShell 脚本，避免本机 bash 指向 WSL 时 Windows 路径无法解析
$migratePs = Join-Path $root 'scripts/check-migrations.ps1'
& $migratePs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ''
Write-Host '=== Lint common ===' -ForegroundColor Cyan
Push-Location $root
& $golangci run --config $golangciYml --out-format=line-number --timeout=5m ./common/...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
Pop-Location

# ---------- Lint ----------
foreach ($svc in $services) {
    Write-Host ''
    Write-Host "=== Lint $svc ===" -ForegroundColor Cyan
    Push-Location (Join-Path $root $svc)
    go mod tidy
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
    go mod verify
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
    & $golangci run --config $golangciYml --out-format=line-number --timeout=5m ./...
    $lintCode = $LASTEXITCODE
    Pop-Location
    if ($lintCode -ne 0) { exit $lintCode }
}
Write-Host ''
Write-Host 'Lint 全部通过' -ForegroundColor Green

# ---------- Test（合并 coverage.out 供与 CI 一致核对）----------
$covOut = Join-Path $root 'coverage.out'
Set-Content -Path $covOut -Value "mode: atomic" -Encoding utf8
foreach ($svc in $services) {
    Write-Host ''
    Write-Host "=== Test $svc ===" -ForegroundColor Cyan
    Push-Location (Join-Path $root $svc)
    go test -v -race -covermode=atomic -count=1 "-coverprofile=coverage.tmp" ./...
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
    if (Test-Path "coverage.tmp") {
        Get-Content "coverage.tmp" | Select-Object -Skip 1 | Add-Content -Path $covOut -Encoding utf8
        Remove-Item "coverage.tmp" -Force
    }
    Pop-Location
}
Write-Host ''
Write-Host ('Test OK, merged coverage: ' + $covOut) -ForegroundColor Green

# ---------- Build（与各服务子模块一致）----------
Write-Host ''
Write-Host '=== Build ===' -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path (Join-Path $root 'bin') | Out-Null

Push-Location (Join-Path $root 'services/user')
go mod tidy
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
go mod verify
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
$userBin = Join-Path $root 'bin/user-service.exe'
go build -o $userBin .
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
Pop-Location

Push-Location (Join-Path $root 'services/device')
go mod tidy
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
go mod verify
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
$deviceBin = Join-Path $root 'bin/device-service.exe'
go build -o $deviceBin .
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
Pop-Location

Push-Location (Join-Path $root 'services/content')
go mod tidy
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
go mod verify
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
$contentBin = Join-Path $root 'bin/content-service.exe'
go build -o $contentBin .
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
Pop-Location

Push-Location (Join-Path $root 'services/media-processing')
go mod tidy
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
go mod verify
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
$mediaBin = Join-Path $root 'bin/media-processing.exe'
go build -o $mediaBin .
if ($LASTEXITCODE -ne 0) { Pop-Location; exit $LASTEXITCODE }
Pop-Location

Write-Host ('Build OK -> ' + (Join-Path $root 'bin')) -ForegroundColor Green
Write-Host 'Local CI finished.' -ForegroundColor Green
