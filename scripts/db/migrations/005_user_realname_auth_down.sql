DROP TABLE IF EXISTS user_real_name_auth;

ALTER TABLE users DROP COLUMN IF EXISTS real_name_type;
ALTER TABLE users DROP COLUMN IF EXISTS real_name_time;
ALTER TABLE users DROP COLUMN IF EXISTS real_name_status;
