-- Remove custom_title column from subscriptions table
ALTER TABLE subscriptions DROP COLUMN IF EXISTS custom_title;

