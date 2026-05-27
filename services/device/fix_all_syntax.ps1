$baseDir = "d:\audio-ai-platform\services\device"

function Fix-File($filePath) {
    $content = Get-Content $filePath -Raw -Encoding UTF8
    $original = $content
    
    # 修复 strings.TrimSpace(variable)) 模式
    # 匹配: strings.TrimSpace(简单变量)) 
    # 不匹配: strings.TrimSpace(函数调用))
    
    # 简单变量模式: req.Sn, sn, snNorm 等
    $patterns = @(
        'strings\.TrimSpace\((req\.\w+)\)\)',
        'strings\.TrimSpace\((sn)\)\)',
        'strings\.TrimSpace\((snNorm)\)\)',
        'strings\.TrimSpace\((in\.\w+)\)\)',
        'strings\.TrimSpace\((principal\.\w+)\)\)',
        'strings\.TrimSpace\((device\.\w+)\)\)'
    )
    
    foreach ($pattern in $patterns) {
        $content = $content -replace $pattern, "strings.TrimSpace(`$1)"
    }
    
    if ($content -ne $original) {
        Set-Content $filePath -Value $content -Encoding UTF8 -NoNewline
        return $true
    }
    return $false
}

# 获取所有 .go 文件并修复
$files = Get-ChildItem -Path $baseDir -Recurse -Filter "*.go"
$fixedCount = 0

foreach ($file in $files) {
    if (Fix-File $file.FullName) {
        Write-Host "✅ Fixed: $($file.FullName)"
        $fixedCount++
    }
}

Write-Host "`n✨ Total files fixed: $fixedCount"
