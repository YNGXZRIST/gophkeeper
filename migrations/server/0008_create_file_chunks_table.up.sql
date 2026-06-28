CREATE TABLE IF NOT EXISTS file_chunks (
                                         file_id UUID NOT NULL REFERENCES files (id) ON DELETE CASCADE,
                                         idx     INT NOT NULL,
                                         data    BYTEA NOT NULL,
                                         PRIMARY KEY (file_id, idx)
);
