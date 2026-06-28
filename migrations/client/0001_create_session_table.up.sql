CREATE TABLE IF NOT EXISTS session (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    login         TEXT    NOT NULL,
    access_token  TEXT    NOT NULL,
    refresh_token TEXT    NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);