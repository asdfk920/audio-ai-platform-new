$baseDir = "d:\audio-ai-platform\services\device"

# 获取所有 .go 文件
$files = Get-ChildItem -Path $baseDir -Recurse -Filter "*.go"

$count = 0
foreach ($file in $files) {
    $content = Get-Content $file.FullName -Raw -Encoding UTF8
    
    # 修复模式1: strings.TrimSpace(variable)) -> strings.TrimSpace(variable)
    # 使用更精确的正则表达式
    $pattern1 = 'strings\.TrimSpace\((?:req\.\w+|sn|snNorm|\w+\.\w+)\)\)'
    $replacement1 = 'strings.TrimSpace($1)'
    
    if ($content -match $pattern1) {
        Write-Host "Found pattern in: $($file.Name)"
        $count++
    }
}

Write-Host "`nTotal files with issues: $count"
