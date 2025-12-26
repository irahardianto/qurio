CREATE TABLE failed_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    handler TEXT NOT NULL,
    payload JSONB NOT NULL,
    error TEXT NOT NULL,
    retries INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
