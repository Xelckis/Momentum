CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE job_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id INT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    author_user_id INT NOT NULL REFERENCES users(id),
    content JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON job_updates (job_id);
CREATE INDEX ON job_updates (author_user_id);
