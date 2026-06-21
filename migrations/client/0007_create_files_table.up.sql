CREATE TABLE IF NOT EXISTS files (
    id             TEXT PRIMARY KEY,
    meta           BLOB NOT NULL,
    chunk_count    INTEGER NOT NULL DEFAULT 0,
    version        INTEGER NOT NULL DEFAULT 1,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dirty          INTEGER NOT NULL DEFAULT 0,
    deleted        INTEGER NOT NULL DEFAULT 0,
    base_version   INTEGER NOT NULL DEFAULT 0,
    conflict       INTEGER NOT NULL DEFAULT 0,
    server_blob    BLOB,
    server_version INTEGER
);
