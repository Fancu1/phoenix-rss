DROP INDEX IF EXISTS idx_articles_last_checked_at;
DROP INDEX IF EXISTS idx_articles_published_at;

ALTER TABLE articles
    DROP COLUMN IF EXISTS http_last_modified,
    DROP COLUMN IF EXISTS http_etag,
    DROP COLUMN IF EXISTS last_checked_at;
