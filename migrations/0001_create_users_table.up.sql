CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login      TEXT        NOT NULL,
    password   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_login_unique UNIQUE (login)
);