-- Migration: Add email verification fields to users table
-- Version: 000004
-- Name: add_email_verification

-- Add email verification fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;