BEGIN TRANSACTION;

CREATE TYPE data_type AS ENUM (
    'PASS',
    'TEXT',
    'BIN',
    'CARD'
);

COMMIT;