CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    ticket_id VARCHAR(50) UNIQUE NOT NULL,
    title TEXT NOT NULL,
    job_type_id INT NOT NULL REFERENCES job_types(id),
    primary_contact_id INT REFERENCES contacts(id) ON DELETE SET NULL,
    assigned_to_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    custom_fields JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON jobs (job_type_id);
CREATE INDEX ON jobs (status);
CREATE INDEX ON jobs (assigned_to_user_id);
CREATE INDEX ON jobs (primary_contact_id);

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON jobs
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
