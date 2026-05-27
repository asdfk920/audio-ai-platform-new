$files = @(
    "internal\logic\ws_auth_logic.go",
    "internal\logic\device_cmd_cache.go",
    "internal\logic\device_shadow_query_logic.go",
    "internal\logic\device_list_logic.go",
    "internal\logic\device_location_query_logic.go",
    "internal\repo\device_repo.go",
    "internal\repo\device_status_log.go",
    "internal\repo\device_status_write.go",
    "internal\device\shadow\keys.go",
    "internal\device\shadow\offline.go",
    "internal\deviceauthsvc\service.go",
    "internal\shadowsvc\service.go",
    "internal\commandsvc\service.go",
    "internal\reportsvc\service.go",
    "internal\handler\status_report.go",
    "generate_ws_auth.go"
)

foreach ($file in $files) {
    $fullPath = Join-Path "d:\audio-ai-platform\services\device" $file
    if (Test-Path $fullPath) {
        $content = Get-Content $fullPath -Raw -Encoding UTF8
        $original = $content
        
        $content = $content -replace 'strings\.ToUpper\(strings\.TrimSpace\(', 'strings.TrimSpace('
        
        if ($content -ne $original) {
            Set-Content $fullPath -Value $content -Encoding UTF8 -NoNewline
            Write-Host "✅ Modified: $file"
        } else {
            Write-Host "⏭️  No changes: $file"
        }
    } else {
        Write-Host "❌ Not found: $file"
    }
}

Write-Host "`n✨ Batch replacement completed!"
