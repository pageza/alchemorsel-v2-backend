-- Migration: Add verification token fields to users table
-- Version: 000003
-- Name: add_verification_tokens

DO $$
BEGIN
    IF NOT migration_applied('000003') THEN
        -- Add verification token fields to users table
        ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token VARCHAR(255);
        ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token_expires_at TIMESTAMPTZ;
        
        -- Record migration
        PERFORM record_migration('000003', 'add_verification_tokens');
    END IF;
END $$;