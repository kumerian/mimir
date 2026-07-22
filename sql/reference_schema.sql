CREATE TABLE IF NOT EXISTS metadata_sources (
    source_name TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    location TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
