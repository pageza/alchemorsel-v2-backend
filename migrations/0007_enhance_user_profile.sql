CREATE TABLE IF NOT EXISTS user_appliances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    appliance_name VARCHAR(100) NOT NULL,
    is_custom BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_appliances_user_id ON user_appliances(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_appliances_unique ON user_appliances(user_id, appliance_name);
