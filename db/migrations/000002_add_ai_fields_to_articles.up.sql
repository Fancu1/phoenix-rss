-- Add AI processing fields to articles table
ALTER TABLE articles ADD COLUMN summary TEXT;
ALTER TABLE articles ADD COLUMN processing_model VARCHAR(255);
ALTER TABLE articles ADD COLUMN processed_at TIMESTAMP WITH TIME ZONE;

-- Create index on processed_at for filtering processed articles
CREATE INDEX idx_articles_processed_at ON articles(processed_at);
