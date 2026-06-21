CREATE TABLE IF NOT EXISTS sync_state (
    entity TEXT PRIMARY KEY,
    cursor TEXT NOT NULL DEFAULT ''
);
