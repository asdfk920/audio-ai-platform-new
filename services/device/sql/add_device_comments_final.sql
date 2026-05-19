-- Add comments to device table columns (Final version with correct field names)
-- Based on actual database schema

-- Main identification fields
COMMENT ON COLUMN device.id IS 'Device primary key ID, auto-increment';
COMMENT ON COLUMN device.sn IS 'Device Serial Number (SN), globally unique identifier';
COMMENT ON COLUMN device.model IS 'Device model/type, e.g., AudioSpeaker, AudioSpeaker_Pro';
COMMENT ON COLUMN device.product_key IS 'Product Key for distinguishing product lines, e.g., audio_platform_default';
COMMENT ON COLUMN device.device_secret IS 'Device secret key for authentication/HMAC-SHA256 signing';

-- Version information
COMMENT ON COLUMN device.firmware_version IS 'Current firmware version, e.g., 1.0.0';
COMMENT ON COLUMN device.hardware_version IS 'Hardware version, e.g., HW_v1.0';

-- Network information
COMMENT ON COLUMN device.mac IS 'MAC address, physical network card address, format: 00:11:22:33:44:55';
COMMENT ON COLUMN device.ip IS 'Last reported IP address (public or private IP when connecting to cloud)';

-- Status fields (Three-state management model)
COMMENT ON COLUMN device.online_status IS 'Online status: 0=Offline, 1=Online';
COMMENT ON COLUMN device.usage_status IS 'Usage status (admin control): 1=Enabled, 2=Disabled';
COMMENT ON COLUMN device.status IS 'Device lifecycle status: 0=Default, 1=Normal, 2=Disabled, 3=Inactive, 4=Unregistered, 5=Unauthenticated';

-- Registration and authentication
COMMENT ON COLUMN device.register_signature IS 'Device registration signature: HMAC-SHA256(device_secret, sn + timestamp), used for WebSocket authentication verification';
COMMENT ON COLUMN device.auth_token IS 'Device authentication token for MQTT/WebSocket connection, JWT format';

-- Timestamps
COMMENT ON COLUMN device.last_active_at IS 'Last active timestamp (last data report or heartbeat)';
COMMENT ON COLUMN device.created_at IS 'Creation timestamp (record insertion time)';
COMMENT ON COLUMN device.updated_at IS 'Update timestamp (last modification time)';
COMMENT ON COLUMN device.deleted_at IS 'Soft delete timestamp (GORM DeletedAt), NULL means not deleted';

-- Audit fields
COMMENT ON COLUMN device.create_by IS 'Creator ID (backend user ID who created this device)';
COMMENT ON COLUMN device.update_by IS 'Updater ID (backend user ID who last modified this device)';
COMMENT ON COLUMN device.admin_display_name IS 'Admin display name (custom name set by admin)';

-- Admin management fields
COMMENT ON COLUMN device.admin_remark IS 'Admin remark/note about this device';
COMMENT ON COLUMN device.admin_location IS 'Device location info (set by admin)';
COMMENT ON COLUMN device.admin_group_id IS 'Device group ID for grouping devices';
COMMENT ON COLUMN device.admin_tags IS 'Device tags in JSONB format for categorization';
COMMENT ON COLUMN device.admin_config IS 'Device admin configuration in JSONB format';

-- User binding
COMMENT ON COLUMN device.bound_user_id IS 'Bound user ID (which user owns this device)';
COMMENT ON COLUMN device.bound_at IS 'Binding timestamp (when device was bound to user)';
COMMENT ON COLUMN device.bind_status IS 'Binding status: 0=Unbound, 1=Bound, 2=Pending';

-- Verify all comments were added successfully
SELECT 
    column_name,
    data_type,
    pg_catalog.col_description((table_schema || '.' || table_name)::regclass::oid, ordinal_position) as comment
FROM information_schema.columns 
WHERE table_schema = 'public' AND table_name = 'device'
ORDER BY ordinal_position;
