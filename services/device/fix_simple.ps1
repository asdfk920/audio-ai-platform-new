$baseDir = "d:\audio-ai-platform\services\device"
$files = Get-ChildItem -Path $baseDir -Recurse -Filter "*.go"

foreach ($file in $files) {
    $content = [System.IO.File]::ReadAllText($file.FullName)
    $original = $content
    
    # 修复常见模式
    $content = $content -replace 'strings\.TrimSpace\((req\.\w+)\)\)', 'strings.TrimSpace($1)'
    $content = $content -replace 'strings\.TrimSpace\((sn)\)\)', 'strings.TrimSpace($1)'
    $content = $content -replace 'strings\.TrimSpace\((snNorm)\)\)', 'strings.TrimSpace($1)'
    $content = $content -replace 'strings\.TrimSpace\((in\.\w+)\)\)', 'strings.TrimSpace($1)'
    $content = $content -replace 'strings\.TrimSpace\((principal\.\w+)\)\)', 'strings.TrimSpace($1)'
    
    if ($content -ne $original) {
        [System.IO.File]::WriteAllText($file.FullName, $content)
        Write-Host "Fixed: $($file.Name)"
    }
}

Write-Host "Done!"
