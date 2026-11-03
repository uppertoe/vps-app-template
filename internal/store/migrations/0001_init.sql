-- The first schema. Once this file has run anywhere, never edit it: add
-- 0002_<what>.sql instead. The kit refuses to start on an edited migration.
CREATE TABLE notes (
    id         INTEGER PRIMARY KEY,
    body       TEXT    NOT NULL CHECK (length(body) BETWEEN 1 AND 2000),
    created_at TEXT    NOT NULL -- RFC 3339, UTC
);
