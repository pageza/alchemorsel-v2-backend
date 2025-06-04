-- Create recipe_favorites table
CREATE TABLE recipe_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT recipe_favorites_unique UNIQUE (recipe_id, user_id)
);

-- Create indexes
CREATE INDEX idx_recipe_favorites_recipe_id ON recipe_favorites(recipe_id);
CREATE INDEX idx_recipe_favorites_user_id ON recipe_favorites(user_id);

-- Create trigger for updated_at
CREATE TRIGGER update_recipe_favorites_updated_at
    BEFORE UPDATE ON recipe_favorites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comment
COMMENT ON TABLE recipe_favorites IS 'Stores user recipe favorites';

