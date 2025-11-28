-- Add status column to feeds table for async subscription flow
-- Status values: 'pending' (awaiting first fetch), 'active' (synced), 'error' (fetch failed)
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Create index for efficient filtering by status
CREATE INDEX IF NOT EXISTS idx_feeds_status ON feeds (status);

-- Update existing feeds to 'active' status (they were already synced)
UPDATE feeds SET status = 'active' WHERE status = '';


