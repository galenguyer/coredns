CREATE TABLE IF NOT EXISTS queries (
    ts TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
    ip INET NOT NULL,
    qname TEXT NOT NULL,
    qtype TEXT NOT NULL,
    rcode TEXT NOT NULL,
    duration_us INTEGER NOT NULL,
    host TEXT NOT NULL
);

SELECT create_hypertable('queries', 'ts');
