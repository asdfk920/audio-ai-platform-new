-- Update device table with mock data for admin fields
-- Fields: deleted_at, create_by, update_by, admin_display_name, admin_remark, admin_location, admin_group_id, admin_tags, admin_config, auth_token

UPDATE public.device SET
    deleted_at = NULL,
    create_by = 1,
    update_by = 1,
    admin_display_name = CASE id
        WHEN 83 THEN 'Living Room Speaker'
        WHEN 86 THEN 'Bedroom Earbuds'
        WHEN 87 THEN 'Study Room Speaker'
        WHEN 88 THEN 'Kitchen Speaker'
        WHEN 89 THEN 'Outdoor Speaker'
        WHEN 90 THEN 'Conference Speaker'
        WHEN 91 THEN 'Kids Story Teller'
        WHEN 92 THEN 'Garage Speaker'
        WHEN 93 THEN 'Office Speaker'
        WHEN 94 THEN 'Gym Speaker'
    END,
    admin_remark = CASE id
        WHEN 83 THEN 'Main living room speaker, supports multi-room sync'
        WHEN 86 THEN 'Bedroom wireless earbuds with noise cancelling'
        WHEN 87 THEN 'Compact speaker for study room, reading mode'
        WHEN 88 THEN 'Waterproof kitchen speaker with voice control'
        WHEN 89 THEN 'Outdoor speaker, IP67 waterproof rating'
        WHEN 90 THEN 'Professional conference speaker system'
        WHEN 91 THEN 'Kids room story teller with content filter'
        WHEN 92 THEN 'Garage speaker with remote control support'
        WHEN 93 THEN 'Premium office speaker with HD audio'
        WHEN 94 THEN 'Gym speaker with high volume mode'
    END,
    admin_location = CASE id
        WHEN 83 THEN '1F Living Room - Above TV Cabinet'
        WHEN 86 THEN '2F Bedroom - Nightstand'
        WHEN 87 THEN '2F Study Room - Bookshelf Level 3'
        WHEN 88 THEN '1F Kitchen - Under Cabinet'
        WHEN 89 THEN '1F Balcony - East Railing'
        WHEN 90 THEN '3F Conference Room - Area A'
        WHEN 91 THEN '2F Kids Room - Left of Desk'
        WHEN 92 THEN '1F Garage - Next to Tool Rack'
        WHEN 93 THEN '4F Office - Manager Desk'
        WHEN 94 THEN '1F Gym - Front of Treadmill'
    END,
    admin_group_id = CASE id
        WHEN 83 THEN 'living_room'
        WHEN 86 THEN 'bedroom'
        WHEN 87 THEN 'study_room'
        WHEN 88 THEN 'kitchen'
        WHEN 89 THEN 'outdoor'
        WHEN 90 THEN 'conference_room'
        WHEN 91 THEN 'kids_room'
        WHEN 92 THEN 'garage'
        WHEN 93 THEN 'office'
        WHEN 94 THEN 'gym'
    END,
    admin_tags = CASE id
        WHEN 83 THEN '["smart_speaker", "wifi", "voice_control", "multi_room"]'::jsonb
        WHEN 86 THEN '["wireless_earbuds", "bluetooth", "noise_cancelling"]'::jsonb
        WHEN 87 THEN '["smart_speaker", "compact", "reading_mode"]'::jsonb
        WHEN 88 THEN '["smart_speaker", "waterproof", "voice_control"]'::jsonb
        WHEN 89 THEN '["outdoor_speaker", "waterproof", "ip67", "solar_power"]'::jsonb
        WHEN 90 THEN '["conference_speaker", "professional", "multi_user", "echo_cancel"]'::jsonb
        WHEN 91 THEN '["kids_speaker", "content_filter", "story_mode", "parental_control"]'::jsonb
        WHEN 92 THEN '["garage_speaker", "remote_control", "weather_resistant"]'::jsonb
        WHEN 93 THEN '["office_speaker", "premium", "multi_room_sync", "hd_audio"]'::jsonb
        WHEN 94 THEN '["gym_speaker", "high_volume", "bass_boost", "sweat_resistant"]'::jsonb
    END,
    admin_config = CASE id
        WHEN 83 THEN '{"report_interval": 30, "volume": 60, "auto_standby": true, "night_mode": false}'::jsonb
        WHEN 86 THEN '{"report_interval": 60, "volume": 40, "noise_cancelling": true, "battery_alert": true}'::jsonb
        WHEN 87 THEN '{"report_interval": 30, "volume": 30, "reading_mode": true, "auto_standby": true}'::jsonb
        WHEN 88 THEN '{"report_interval": 30, "volume": 50, "waterproof_mode": true, "voice_wake": true}'::jsonb
        WHEN 89 THEN '{"report_interval": 120, "volume": 70, "solar_mode": true, "weather_alert": true}'::jsonb
        WHEN 90 THEN '{"report_interval": 15, "volume": 65, "echo_cancel": true, "multi_user": true}'::jsonb
        WHEN 91 THEN '{"report_interval": 60, "volume": 35, "content_filter": true, "sleep_timer": true}'::jsonb
        WHEN 92 THEN '{"report_interval": 300, "volume": 55, "remote_control": true, "motion_sensor": true}'::jsonb
        WHEN 93 THEN '{"report_interval": 15, "volume": 45, "hd_audio": true, "multi_room": true}'::jsonb
        WHEN 94 THEN '{"report_interval": 30, "volume": 80, "bass_boost": true, "workout_mode": true}'::jsonb
    END,
    auth_token = CASE id
        WHEN 83 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjgzLCJzbiI6IkFVU1AyNjA1MDAwMDAxRzIiLCJleHAiOjE3NzgzMDY1Njh9.abc123def456'
        WHEN 86 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjg2LCJzbiI6IkFVU1AyNjA1MDAwMDAyRzIiLCJleHAiOjE3NzgzMDY1Njh9.bcd234efg567'
        WHEN 87 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjg3LCJzbiI6IkFVU1AyNjA1MDAwMDAzRzIiLCJleHAiOjE3NzgzMDY1Njh9.cde345fgh678'
        WHEN 88 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjg4LCJzbiI6IkFVU1AyNjA1MDAwMDA0RzIiLCJleHAiOjE3NzgzMDY1Njh9.def456ghi789'
        WHEN 89 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjg5LCJzbiI6IkFVU1AyNjA1MDAwMDA1RzIiLCJleHAiOjE3NzgzMDY1Njh9.efg567hij890'
        WHEN 90 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjkwLCJzbiI6IkFVU1AyNjA1MDAwMDA2RzIiLCJleHAiOjE3NzgzMDY1Njh9.fgh678ijk901'
        WHEN 91 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjkxLCJzbiI6IkFVU1AyNjA1MDAwMDA3RzIiLCJleHAiOjE3NzgzMDY1Njh9.ghi789jkl012'
        WHEN 92 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjkyLCJzbiI6IkFVU1AyNjA1MDAwMDA4RzIiLCJleHAiOjE3NzgzMDY1Njh9.hij890klm123'
        WHEN 93 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjkzLCJzbiI6IkFVU1AyNjA1MDAwMDA5RzIiLCJleHAiOjE3NzgzMDY1Njh9.ijk901lmn234'
        WHEN 94 THEN 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkZXZpY2VfaWQiOjk0LCJzbiI6IkFVU1AyNjA1MDAwMDEwRzIiLCJleHAiOjE3NzgzMDY1Njh9.jkl012mno345'
    END,
    updated_at = NOW()
WHERE id IN (83, 86, 87, 88, 89, 90, 91, 92, 93, 94);

-- Verify updated data
SELECT 
    id, 
    sn, 
    admin_display_name, 
    admin_remark, 
    admin_location, 
    admin_group_id, 
    admin_tags, 
    admin_config, 
    auth_token,
    create_by,
    update_by,
    deleted_at
FROM public.device 
WHERE id IN (83, 86, 87, 88, 89, 90, 91, 92, 93, 94)
ORDER BY id;
