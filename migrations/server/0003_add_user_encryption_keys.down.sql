ALTER TABLE users
    DROP COLUMN IF EXISTS enc_salt,
    DROP COLUMN IF EXISTS wrapped_dek;