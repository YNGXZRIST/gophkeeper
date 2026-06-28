CREATE TABLE IF NOT EXISTS files (
                                         id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                         user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
                                         meta        BYTEA NOT NULL,
                                         chunk_count INT NOT NULL,
                                         version     BIGINT NOT NULL DEFAULT 1,
                                         created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS files_user_id_idx ON files (user_id);
