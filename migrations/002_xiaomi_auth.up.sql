-- Add Xiaomi auth method tracking and encrypted auth data storage
ALTER TABLE mifit_accounts ADD COLUMN IF NOT EXISTS auth_method VARCHAR(20) DEFAULT 'zepp';
ALTER TABLE mifit_accounts ADD COLUMN IF NOT EXISTS xiaomi_auth_data BYTEA;
