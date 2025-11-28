-- Remove status column from feeds table
DROP INDEX IF EXISTS idx_feeds_status;
ALTER TABLE feeds DROP COLUMN IF EXISTS status;


