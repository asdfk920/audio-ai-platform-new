# 生成WebSocket认证消息的完整脚本
# 自动调用注册接口 + 计算签名 + 输出可用的JSON

Write-Host "====================================" -ForegroundColor Cyan
Write-Host "🔧 WebSocket认证消息自动生成器" -ForegroundColor Yellow
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""

# 设备信息
$sn = "AUSP2605000002Y2"
$deviceSecret = "G2WCNIrxLdYGBVbBze9KCH2pkQCUiKkq"
$baseUrl = "http://localhost:8002"

Write-Host "步骤1: 调用注册接口获取Token..." -ForegroundColor Green
Write-Host "  POST $baseUrl/api/device/register" -ForegroundColor Gray

# 构建注册请求
$registerBody = @{
    sn = $sn
    device_secret = $deviceSecret
} | ConvertTo-Json

$headers = @{"Content-Type" = "application/json"}

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/device/register" -Method POST -Headers $headers -Body $registerBody
    # go-zero 成功响应常为扁平 JSON：token / device_id / expires_in / register_timestamp / signature
    if ($response.PSObject.Properties.Name -contains 'code' -and $response.code -ne 200) {
        Write-Host "❌ 注册失败: $($response.msg)" -ForegroundColor Red
        exit 1
    }

    $token = $response.token
    if (-not $token -and $response.data) { $token = $response.data.token }
    $deviceId = $response.device_id
    if (-not $deviceId -and $response.data) { $deviceId = $response.data.device_id }
    $expiresIn = $response.expires_in
    if (-not $expiresIn -and $response.data) { $expiresIn = $response.data.expires_in }
    $regTs = $response.register_timestamp

    if (-not $token) {
        Write-Host "❌ 注册响应中无 token" -ForegroundColor Red
        exit 1
    }

    Write-Host "✅ 注册成功!" -ForegroundColor Green
    Write-Host "  DeviceID:   $deviceId"
    Write-Host "  Token:      $($token.Substring(0, [Math]::Min(50, $token.Length)))..."
    Write-Host "  ExpiresIn:  $expiresIn 秒 ($([Math]::Round($expiresIn/3600, 1)) 小时)"
    if ($regTs) { Write-Host "  RegisterTs: $regTs（bcrypt 密钥时请用作 WS timestamp）" -ForegroundColor DarkGray }
} catch {
    Write-Host "❌ 注册接口调用失败: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "步骤2: 生成时间戳和签名..." -ForegroundColor Green

# 生成当前时间戳（毫秒）
$timestamp = [long]([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())
$timestampFormatted = [DateTimeOffset]::FromUnixTimeMilliseconds($timestamp).ToString("yyyy-MM-dd HH:mm:ss.fff")

Write-Host "  当前时间戳: $timestamp ($timestampFormatted UTC)"

# 计算HMAC-SHA256签名（与云端一致）：signData = sn + timestamp(ms)，JWT 只在握手 Authorization，不参与签名
$signData = "$sn$timestamp"

$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($deviceSecret)
$signatureBytes = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($signData))
$signature = -join ($signatureBytes | ForEach-Object { $_.ToString("x2") })

Write-Host "  签名数据长度: $($signData.Length) 字符"
Write-Host "  HMAC-SHA256:  $signature"

Write-Host ""
Write-Host "步骤3: 构建认证消息..." -ForegroundColor Green

# 构建完整的认证消息JSON（标准格式，无注释！）
$authMessage = [ordered]@{
    type = "auth"
    sn = $sn
    timestamp = $timestamp
    signature = $signature
} | ConvertTo-Json -Depth 3 -Compress

Write-Host "✅ 认证消息已生成!" -ForegroundColor Green

Write-Host ""
Write-Host "====================================" -ForegroundColor Yellow
Write-Host "📋 复制下面的完整JSON到Apifox:" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Yellow
Write-Host ""
Write-Host $authMessage -ForegroundColor White
Write-Host ""

# 保存到文件
$outputFile = "$PSScriptRoot\ws_auth_message.json"
$authMessage | Out-File -FilePath $outputFile -Encoding UTF8
Write-Host "✅ 已保存到文件: $outputFile" -ForegroundColor Green

# 复制到剪贴板
$authMessage | Set-Clipboard
Write-Host "✅ 已复制到剪贴板! (Ctrl+V粘贴)" -ForegroundColor Green

Write-Host ""
Write-Host "====================================" -ForegroundColor Cyan
Write-Host "🎯 使用步骤:" -ForegroundColor Yellow
Write-Host "====================================" -ForegroundColor Cyan
Write-Host "1. 打开Apifox → WebSocket界面" -ForegroundColor White
Write-Host "2. 在「请求头」添加 Authorization: Bearer <token>（勿把 token 放进 JSON）" -ForegroundColor White
Write-Host "3. 输入URL: ws://localhost:8002/ws/device" -ForegroundColor White
Write-Host "4. 连接成功后立即发送下面生成的 JSON（10 秒内）" -ForegroundColor White
Write-Host "5. ✅ 应收到 type=auth_response 的成功响应" -ForegroundColor White
Write-Host ""
Write-Host "⚠️  注意事项:" -ForegroundColor Yellow
Write-Host "  - 签名 signData = sn + timestamp(ms)；HMAC-SHA256(device_secret) → hex" -ForegroundColor White
Write-Host "  - device_secret 存 bcrypt 时，timestamp 须为注册返回的 register_timestamp，signature 须与注册返回一致" -ForegroundColor White
Write-Host "  - Token 过期请重新运行脚本" -ForegroundColor White
Write-Host "====================================" -ForegroundColor Cyan
