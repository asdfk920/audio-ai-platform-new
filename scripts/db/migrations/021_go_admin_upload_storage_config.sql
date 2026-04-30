-- 021: go-admin 上传存储配置（写入 sys_config，便于界面修改）
SET search_path TO public;

-- 约定：
-- - upload_storage_driver: local|oss
-- - upload_storage_public_base_url: 对外访问前缀（例如 https://<bucket>.<region>.aliyuncs.com）
-- - upload_storage_oss_endpoint: OSS endpoint（例如 https://oss-cn-beijing.aliyuncs.com）
-- - upload_storage_oss_bucket: OSS bucket 名称
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_frontend, remark, create_by, update_by, created_at, updated_at, deleted_at)
VALUES
('uploadStorageDriver', 'upload_storage_driver', 'local', 'Y', '0', '上传存储驱动：local 或 oss', 1, 1, NOW(), NOW(), NULL),
('uploadStoragePublicBaseURL', 'upload_storage_public_base_url', '', 'Y', '0', '上传文件对外访问前缀；driver=oss 时生效', 1, 1, NOW(), NOW(), NULL),
('uploadStorageOSSEndpoint', 'upload_storage_oss_endpoint', '', 'Y', '0', '阿里云 OSS endpoint；driver=oss 时生效', 1, 1, NOW(), NOW(), NULL),
('uploadStorageOSSBucket', 'upload_storage_oss_bucket', '', 'Y', '0', '阿里云 OSS bucket；driver=oss 时生效', 1, 1, NOW(), NOW(), NULL)
ON CONFLICT DO NOTHING;

