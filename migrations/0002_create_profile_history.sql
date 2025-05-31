-- Create profile history table
CREATE TABLE IF NOT EXISTS profile_history (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    field VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_at TIMESTAMP NOT NULL,
    changed_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_profile_history_user_id ON profile_history(user_id);

-- Create index on changed_at for chronological queries
CREATE INDEX IF NOT EXISTS idx_profile_history_changed_at ON profile_history(changed_at); 