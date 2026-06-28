ALTER TABLE session ADD COLUMN enc_salt    BLOB;
ALTER TABLE session ADD COLUMN wrapped_dek BLOB;
