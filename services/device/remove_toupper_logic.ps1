$logicFiles = @(
    "internal\logic\device_bind_logic.go",
    "internal\logic\device_detail_logic.go",
    "internal\logic\device_diagnose_logic.go",
    "internal\logic\device_download_song_logic.go",
    "internal\logic\device_log_report_logic.go",
    "internal\logic\device_next_logic.go",
    "internal\logic\device_pause_logic.go",
    "internal\logic\device_playback_progress_logic.go",
    "internal\logic\device_playback_status_logic.go",
    "internal\logic\device_play_audio_logic.go",
    "internal\logic\device_play_logic.go",
    "internal\logic\device_play_playlist_logic.go",
    "internal\logic\device_prev_logic.go",
    "internal\logic\device_reboot_logic.go",
    "internal\logic\device_resume_logic.go",
    "internal\logic\device_seek_logic.go",
    "internal\logic\device_set_loop_logic.go",
    "internal\logic\device_set_shuffle_logic.go",
    "internal\logic\device_shadow_report_logic.go",
    "internal\logic\device_status_update_logic.go",
    "internal\logic\device_update_logic.go",
    "internal\logic\device_volume_down_logic.go",
    "internal\logic\device_volume_up_logic.go"
)

foreach ($file in $logicFiles) {
    $fullPath = Join-Path "d:\audio-ai-platform\services\device" $file
    if (Test-Path $fullPath) {
        $content = Get-Content $fullPath -Raw -Encoding UTF8
        $original = $content
        
        $content = $content -replace 'strings\.ToUpper\(strings\.TrimSpace\(', 'strings.TrimSpace('
        
        if ($content -ne $original) {
            Set-Content $fullPath -Value $content -Encoding UTF8 -NoNewline
            Write-Host "✅ Modified: $file"
        }
    }
}

Write-Host "`n✨ Logic files batch replacement completed!"
