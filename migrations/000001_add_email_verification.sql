-- Migration: Add email verification fields to users table
-- Version: 000001
-- Name: add_email_verification

DO $$
BEGIN
    IF NOT migration_applied('000001') THEN
        -- Add email verification fields to users table
        ALTER TABLE users ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN DEFAULT FALSE;
        ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
        
        -- Record migration
        PERFORM record_migration('000001', 'add_email_verification');
    END IF;
END $$;