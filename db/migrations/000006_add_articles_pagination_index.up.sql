-- Add composite index for efficient pagination queries on articles
-- This index supports: WHERE feed_id = ? ORDER BY published_at DESC LIMIT ? OFFSET ?
-- The index ordering (DESC) matches the query pattern, avoiding filesort operations.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_articles_feed_published 
    ON articles (feed_id, published_at DESC);


