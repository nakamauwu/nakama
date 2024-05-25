CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY, -- Random session identifier.
	data BYTEA NOT NULL, -- Gob serialized session data.
	expiry TIMESTAMPTZ NOT NULL
) WITH (ttl_expiration_expression = 'expiry');

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR NOT NULL PRIMARY KEY DEFAULT uuid_to_ulid(gen_random_ulid()),
    email VARCHAR NOT NULL UNIQUE,
    username VARCHAR NOT NULL CHECK (username != ''),
    avatar JSONB, -- { path: string, width: int, height: int, thumbhash: string }
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now() ON UPDATE now(),
    UNIQUE (LOWER(username))
);

CREATE TABLE IF NOT EXISTS posts (
    id VARCHAR NOT NULL PRIMARY KEY DEFAULT uuid_to_ulid(gen_random_ulid()),
    user_id VARCHAR NOT NULL REFERENCES users ON DELETE CASCADE ON UPDATE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now() ON UPDATE now()
);
