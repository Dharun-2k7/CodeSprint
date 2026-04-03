-- Migration: Add admin system and contest metadata

-- Add admin fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_main_manager BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS otp_secret VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_expires TIMESTAMP;

-- Add contest metadata fields
ALTER TABLE contests ADD COLUMN IF NOT EXISTS writer_name VARCHAR(255);
ALTER TABLE contests ADD COLUMN IF NOT EXISTS standings_frozen BOOLEAN DEFAULT FALSE;
ALTER TABLE contests ADD COLUMN IF NOT EXISTS freeze_time TIMESTAMP;

-- Create OTP codes table for password reset and 2FA
CREATE TABLE IF NOT EXISTS otp_codes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(6) NOT NULL,
    purpose VARCHAR(50) NOT NULL, -- 'password_reset' or 'two_factor'
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for OTP codes
CREATE INDEX IF NOT EXISTS idx_otp_user_expires ON otp_codes(user_id, expires_at, used);

-- Create email queue table for sending emails (forgot password, OTP, etc.)
CREATE TABLE IF NOT EXISTS email_queue (
    id SERIAL PRIMARY KEY,
    recipient_email VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP
);

-- Index for email queue
CREATE INDEX IF NOT EXISTS idx_email_queue_sent ON email_queue(sent, created_at);

