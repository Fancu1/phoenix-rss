-- Remove AI processing fields from articles table
DROP INDEX IF EXISTS idx_articles_processed_at;

ALTER TABLE articles DROP COLUMN IF EXISTS processed_at;
ALTER TABLE articles DROP COLUMN IF EXISTS processing_model;
ALTER TABLE articles DROP COLUMN IF EXISTS summary;
