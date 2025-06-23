CREATE EXTENSION IF NOT EXISTS vector;

-- Create update timestamp function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create dietary preference type enum (idempotent)
-- E2E-FIX-2025-F: Use DO block to create enum type only if it doesn't exist
DO $$ BEGIN
    CREATE TYPE dietary_preference_type AS ENUM (
        'vegetarian',
        'vegan',
        'pescatarian',
        'gluten-free',
        'dairy-free',
        'nut-free',
        'soy-free',
        'egg-free',
        'shellfish-free',
        'custom'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL
);

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create user profiles table
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    username VARCHAR(50) NOT NULL UNIQUE,
    bio TEXT,
    profile_picture_url VARCHAR(255),
    privacy_level VARCHAR(50) NOT NULL DEFAULT 'private',
    UNIQUE(user_id)
);

CREATE TRIGGER update_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create dietary preferences table
CREATE TABLE IF NOT EXISTS dietary_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    preference_type dietary_preference_type NOT NULL,
    custom_name VARCHAR(50)
);

CREATE TRIGGER update_dietary_preferences_updated_at
    BEFORE UPDATE ON dietary_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create allergens table
CREATE TABLE IF NOT EXISTS allergens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    allergen_name VARCHAR(50) NOT NULL,
    severity_level INTEGER NOT NULL
);

CREATE TRIGGER update_allergens_updated_at
    BEFORE UPDATE ON allergens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create profile history table
CREATE TABLE profile_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    height FLOAT,
    weight FLOAT,
    age INTEGER,
    gender VARCHAR(50),
    activity_level VARCHAR(50),
    dietary_preferences JSONB,
    allergies JSONB,
    goals JSONB
);

CREATE TRIGGER update_profile_history_updated_at
    BEFORE UPDATE ON profile_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create recipes table
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    ingredients JSONB NOT NULL,
    instructions JSONB NOT NULL,
    prep_time INTEGER,
    cook_time INTEGER,
    servings INTEGER,
    difficulty VARCHAR(50),
    category VARCHAR(50),
    cuisine VARCHAR(50),
    image_url TEXT,
    dietary_preferences JSONB,
    tags JSONB,
    calories INTEGER,
    protein DOUBLE PRECISION,
    carbs DOUBLE PRECISION,
    fat DOUBLE PRECISION,
    embedding vector(1536)
);

CREATE TRIGGER update_recipes_updated_at
    BEFORE UPDATE ON recipes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create recipe favorites table
CREATE TABLE recipe_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id),
    recipe_id UUID NOT NULL REFERENCES recipes(id),
    UNIQUE(user_id, recipe_id)
);

CREATE TRIGGER update_recipe_favorites_updated_at
    BEFORE UPDATE ON recipe_favorites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_recipes_user_id ON recipes(user_id);
CREATE INDEX IF NOT EXISTS idx_recipes_category ON recipes(category);
CREATE INDEX IF NOT EXISTS idx_recipes_cuisine ON recipes(cuisine);
CREATE INDEX IF NOT EXISTS idx_recipes_difficulty ON recipes(difficulty);
CREATE INDEX IF NOT EXISTS idx_recipes_tags ON recipes USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_recipes_dietary_preferences ON recipes USING GIN(dietary_preferences);
CREATE INDEX IF NOT EXISTS idx_recipes_embedding ON recipes USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100); 