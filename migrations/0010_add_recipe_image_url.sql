-- Add image_url column to recipes table if it does not exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='recipes' AND column_name='image_url'
    ) THEN
        ALTER TABLE recipes ADD COLUMN image_url TEXT;
    END IF;
END$$;

-- Update existing records to have empty image_url
UPDATE recipes SET image_url = '' WHERE image_url IS NULL; 