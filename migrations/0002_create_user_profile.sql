-- Create enum types for dietary preferences and privacy settings
CREATE TYPE dietary_preference_type AS ENUM (
    'vegan',
    'vegetarian',
    'keto',
    'paleo',
    'gluten_free',
    'dairy_free',
    'custom'
);

CREATE TYPE privacy_level AS ENUM (
    'public',
    'private',
    'friends_only'
);

-- Create user_profiles table
CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    bio TEXT,
    profile_picture_url VARCHAR(255),
    privacy_level privacy_level NOT NULL DEFAULT 'private',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT username_length CHECK (char_length(username) >= 3 AND char_length(username) <= 50),
    CONSTRAINT email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Create dietary_preferences table
CREATE TABLE dietary_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    preference_type dietary_preference_type NOT NULL,
    custom_name VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT custom_name_required CHECK (
        (preference_type = 'custom' AND custom_name IS NOT NULL) OR
        (preference_type != 'custom' AND custom_name IS NULL)
    )
);

-- Create allergens table
CREATE TABLE allergens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    allergen_name VARCHAR(50) NOT NULL,
    severity_level INTEGER NOT NULL CHECK (severity_level BETWEEN 1 AND 5),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create profile_history table for audit purposes
CREATE TABLE profile_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    field_name VARCHAR(50) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    changed_by UUID NOT NULL REFERENCES users(id)
);

-- Create indexes
CREATE INDEX idx_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX idx_dietary_preferences_user_id ON dietary_preferences(user_id);
CREATE INDEX idx_allergens_user_id ON allergens(user_id);
CREATE INDEX idx_profile_history_user_id ON profile_history(user_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_dietary_preferences_updated_at
    BEFORE UPDATE ON dietary_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_allergens_updated_at
    BEFORE UPDATE ON allergens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comment to tables
COMMENT ON TABLE user_profiles IS 'Stores user profile information';
COMMENT ON TABLE dietary_preferences IS 'Stores user dietary preferences and restrictions';
COMMENT ON TABLE allergens IS 'Stores user allergen information';
COMMENT ON TABLE profile_history IS 'Audit trail for profile changes'; 