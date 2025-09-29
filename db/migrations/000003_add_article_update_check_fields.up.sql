ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS http_etag TEXT NULL,
    ADD COLUMN IF NOT EXISTS http_last_modified TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles (published_at DESC);
CREATE INDEX IF NOT EXISTS idx_articles_last_checked_at ON articles (last_checked_at);
