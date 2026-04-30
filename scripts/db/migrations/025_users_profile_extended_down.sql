ALTER TABLE users DROP CONSTRAINT IF EXISTS users_profile_complete_score_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_profile_complete_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_gender_visibility_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_birthday_visibility_check;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_age_check;

ALTER TABLE users DROP COLUMN IF EXISTS location;
ALTER TABLE users DROP COLUMN IF EXISTS hobbies;
ALTER TABLE users DROP COLUMN IF EXISTS profile_complete_score;
ALTER TABLE users DROP COLUMN IF EXISTS profile_complete;
ALTER TABLE users DROP COLUMN IF EXISTS gender_visibility;
ALTER TABLE users DROP COLUMN IF EXISTS birthday_visibility;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
ALTER TABLE users DROP COLUMN IF EXISTS signature;
ALTER TABLE users DROP COLUMN IF EXISTS age;
ALTER TABLE users DROP COLUMN IF EXISTS constellation;
