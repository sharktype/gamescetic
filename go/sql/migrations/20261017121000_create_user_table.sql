-- +goose Up
-- +goose StatementBegin

CREATE TABLE "user" (
    id          TEXT PRIMARY KEY NOT NULL,

    sub         TEXT NOT NULL UNIQUE,

    name        TEXT NOT NULL,
    email       TEXT NOT NULL UNIQUE,

    avatar_url  TEXT,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "user";

-- +goose StatementEnd
