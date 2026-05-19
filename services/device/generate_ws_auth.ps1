# WebSocket 首包认证 JSON 生成器（signData = sn + timestamp(ms)，JWT 仅放在握手 Authorization）
#
# 用法：填写 SN / device_secret / timestamp（默认当前毫秒），复制输出的 JSON；
#       在 Apifox 的 WS「请求头」添加 Authorization: Bearer <注册接口返回的 token>

$sn = "AUSP2605000002Y2"
$deviceSecret = "G2WCNIrxLdYGBVbBze9KCH2pkQCUiKkq"
$timestamp = [long]([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())

Write-Host "====================================" -ForegroundColor Cyan
Write-Host "WebSocket 认证首包 JSON" -ForegroundColor Yellow
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "若库中 device_secret 为 bcrypt，请将 timestamp 改为注册返回的 register_timestamp，signature 改为注册返回的 signature。" -ForegroundColor DarkYellow
Write-Host ""

$signData = "$sn$timestamp"
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($deviceSecret)
$signatureBytes = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($signData))
$signature = -join ($signatureBytes | ForEach-Object { $_.ToString("x2") })

Write-Host "signData = sn + timestamp = $signData" -ForegroundColor Gray
Write-Host "signature (hex) = $signature" -ForegroundColor Gray
Write-Host ""

$authMessage = @{
    type      = "auth"
    sn        = $sn
    timestamp = $timestamp
    signature = $signature
} | ConvertTo-Json -Depth 3 -Compress

Write-Host $authMessage -ForegroundColor White
$authMessage | Set-Clipboard
Write-Host ""
Write-Host "已复制到剪贴板。握手时请设置 Authorization: Bearer <token>" -ForegroundColor Green
