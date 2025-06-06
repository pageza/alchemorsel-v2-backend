-- Add missing columns to user_profiles table
ALTER TABLE user_profiles
ADD COLUMN username VARCHAR(50) NOT NULL DEFAULT '',
ADD COLUMN email VARCHAR(255) NOT NULL DEFAULT '',
ADD COLUMN bio TEXT,
ADD COLUMN profile_picture_url VARCHAR(255),
ADD COLUMN privacy_level VARCHAR(50) NOT NULL DEFAULT 'private';

-- Create unique index on username
CREATE UNIQUE INDEX idx_user_profiles_username ON user_profiles(username);

-- Update existing rows to use email from users table
UPDATE user_profiles up
SET email = u.email
FROM users u
WHERE up.user_id = u.id; 