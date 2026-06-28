CREATE TABLE IF NOT EXISTS notes (
                                         id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                         user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
                                         data       BYTEA NOT NULL,
                                         version    BIGINT NOT NULL DEFAULT 1,
                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notes_user_id_idx ON notes (user_id);
