CREATE TABLE deadlines (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL,
    text         TEXT         NOT NULL,
    deadline_at  TIMESTAMPTZ  NOT NULL,
    reminded_24h BOOLEAN      DEFAULT FALSE,
    reminded_12h BOOLEAN      DEFAULT FALSE,
    reminded_6h  BOOLEAN      DEFAULT FALSE,
    reminded_3h  BOOLEAN      DEFAULT FALSE,
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);
