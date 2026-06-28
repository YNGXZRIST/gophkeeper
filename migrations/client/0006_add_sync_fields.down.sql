ALTER TABLE notes DROP COLUMN server_version;
ALTER TABLE notes DROP COLUMN server_blob;
ALTER TABLE notes DROP COLUMN conflict;
ALTER TABLE notes DROP COLUMN base_version;
ALTER TABLE notes DROP COLUMN deleted;
ALTER TABLE notes DROP COLUMN dirty;

ALTER TABLE passwords DROP COLUMN server_version;
ALTER TABLE passwords DROP COLUMN server_blob;
ALTER TABLE passwords DROP COLUMN conflict;
ALTER TABLE passwords DROP COLUMN base_version;
ALTER TABLE passwords DROP COLUMN deleted;
ALTER TABLE passwords DROP COLUMN dirty;

ALTER TABLE cards DROP COLUMN server_version;
ALTER TABLE cards DROP COLUMN server_blob;
ALTER TABLE cards DROP COLUMN conflict;
ALTER TABLE cards DROP COLUMN base_version;
ALTER TABLE cards DROP COLUMN deleted;
ALTER TABLE cards DROP COLUMN dirty;
