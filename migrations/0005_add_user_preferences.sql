
CREATE TYPE cooking_ability_level AS ENUM (
    'beginner',
    'intermediate', 
    'advanced',
    'expert'
);

CREATE TYPE kitchen_appliance AS ENUM (
    'oven',
    'microwave',
    'stovetop',
    'air_fryer',
    'slow_cooker',
    'pressure_cooker',
    'blender',
    'food_processor',
    'stand_mixer',
    'toaster',
    'grill',
    'rice_cooker',
    'bread_maker',
    'deep_fryer',
    'steamer',
    'sous_vide',
    'custom'
);

CREATE TABLE user_appliances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    appliance_type kitchen_appliance NOT NULL,
    custom_name VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT custom_appliance_name_required CHECK (
        (appliance_type = 'custom' AND custom_name IS NOT NULL) OR
        (appliance_type != 'custom' AND custom_name IS NULL)
    ),
    UNIQUE(user_id, appliance_type, custom_name)
);

ALTER TABLE user_profiles 
ADD COLUMN cooking_ability_level cooking_ability_level DEFAULT 'beginner';

CREATE INDEX idx_user_appliances_user_id ON user_appliances(user_id);
CREATE INDEX idx_user_profiles_ability_level ON user_profiles(cooking_ability_level);

CREATE TRIGGER update_user_appliances_updated_at
    BEFORE UPDATE ON user_appliances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE user_appliances IS 'Stores user available kitchen appliances';
COMMENT ON COLUMN user_profiles.cooking_ability_level IS 'User self-reported cooking skill level';
