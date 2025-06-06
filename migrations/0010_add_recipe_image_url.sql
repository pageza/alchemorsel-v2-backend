-- Add image_url column to recipes table
ALTER TABLE recipes ADD COLUMN image_url TEXT;

-- Update existing records to have empty image_url
UPDATE recipes SET image_url = '' WHERE image_url IS NULL; 