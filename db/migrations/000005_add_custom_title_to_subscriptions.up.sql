-- Add custom_title column to subscriptions table for user-specific feed naming
-- When custom_title is NULL, the original feed.title should be used
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS custom_title VARCHAR(255);


