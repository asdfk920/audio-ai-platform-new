# 与 check-migrations.sh 相同：校验 *_down.sql 均有对应正向 .sql
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$dir = Join-Path $root "scripts/db/migrations"
if (-not (Test-Path $dir)) {
    Write-Error "missing $dir"
    exit 1
}
Get-ChildItem -Path $dir -Filter "*_down.sql" -File | ForEach-Object {
    $base = $_.Name
    $forward = $base -replace '_down\.sql$', '.sql'
    $fwdPath = Join-Path $dir $forward
    if (-not (Test-Path $fwdPath)) {
        Write-Error "migration: rollback $base has no forward file $forward"
        exit 1
    }
}
Write-Host "check-migrations: OK ($dir)"
