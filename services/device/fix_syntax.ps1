$files = @(
    "internal\repo\device_repo.go",
    "internal\repo\device_status_log.go",
    "internal\repo\device_status_write.go",
    "internal\device\shadow\keys.go",
    "internal\device\shadow\offline.go",
    "internal\deviceauthsvc\service.go"
)

foreach ($file in $files) {
    $fullPath = Join-Path "d:\audio-ai-platform\services\device" $file
    if (Test-Path $fullPath) {
        $content = Get-Content $fullPath -Raw -Encoding UTF8
        $original = $content
        
        # 修复 strings.TrimSpace(sn)) -> strings.TrimSpace(sn)
        $content = $content -replace 'strings\.TrimSpace\([^)]+\)\)', 'strings.TrimSpace($1)'
        
        if ($content -ne $original) {
            Set-Content $fullPath -Value $content -Encoding UTF8 -NoNewline
            Write-Host "✅ Fixed: $file"
        }
    }
}

Write-Host "`n✨ Syntax fixes completed!"
