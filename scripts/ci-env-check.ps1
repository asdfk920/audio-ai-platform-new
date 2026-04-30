# CI environment check for local scripts (ci-local.ps1). Usage: .\scripts\ci-env-check.ps1

$ErrorActionPreference = "Continue"
$ok = $true

Write-Host "========== CI env check ==========" -ForegroundColor Cyan

Write-Host ""
Write-Host "[Go]"
go version | Out-Host
if (-not $?) { $ok = $false }
Write-Host ("  GOPATH: " + (go env GOPATH))
Write-Host ("  GOROOT: " + (go env GOROOT))
Write-Host ("  GOTOOLCHAIN: " + (go env GOTOOLCHAIN))

Write-Host ""
Write-Host "[golangci-lint]"
$gl = Join-Path (go env GOPATH) "bin\golangci-lint.exe"
if (Test-Path $gl) {
    & $gl version
} else {
    Write-Host "  MISSING: install with:" -ForegroundColor Red
    Write-Host "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
    $ok = $false
}

Write-Host ""
Write-Host "[scripts]"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$ciLocal = Join-Path $root "scripts\ci-local.ps1"
if (Test-Path $ciLocal) {
    Write-Host ("  ci-local.ps1 OK: " + $ciLocal)
} else {
    Write-Host "  ci-local.ps1 MISSING" -ForegroundColor Red
    $ok = $false
}

Write-Host ""
Write-Host "[GitHub Actions]"
Write-Host "  Workflow: .github/workflows/ci.yml"
Write-Host "  Local:    .\scripts\ci-local.ps1"

Write-Host ""
Write-Host "=================================="
if ($ok) {
    Write-Host "RESULT: OK (ready to run ci-local.ps1)" -ForegroundColor Green
    exit 0
}
Write-Host "RESULT: FIX items above" -ForegroundColor Red
exit 1
