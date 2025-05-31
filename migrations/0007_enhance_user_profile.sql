ALTER TABLE user_profiles ADD COLUMN cooking_ability_level VARCHAR(20) DEFAULT 'beginner' CHECK (cooking_ability_level IN ('beginner', 'intermediate', 'advanced', 'expert'));

CREATE TABLE IF NOT EXISTS user_appliances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    appliance_name VARCHAR(100) NOT NULL,
    is_custom BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_appliances_user_id ON user_appliances(user_id);
CREATE INDEX IF NOT EXISTS idx_user_appliances_deleted_at ON user_appliances(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_appliances_unique ON user_appliances(user_id, appliance_name) WHERE deleted_at IS NULL;

CREATE OR REPLACE TRIGGER update_user_appliances_updated_at
    BEFORE UPDATE ON user_appliances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
