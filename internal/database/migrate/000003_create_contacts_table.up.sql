CREATE TABLE contacts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50),
    custom_fields JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
