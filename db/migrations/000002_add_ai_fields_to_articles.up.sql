-- Add AI processing fields to articles table
ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS summary TEXT,
    ADD COLUMN IF NOT EXISTS processing_model VARCHAR(255),
    ADD COLUMN IF NOT EXISTS processed_at TIMESTAMP WITH TIME ZONE;

-- Create index on processed_at for filtering processed articles
CREATE INDEX IF NOT EXISTS idx_articles_processed_at ON articles(processed_at);
