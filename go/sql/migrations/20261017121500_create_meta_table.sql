-- +goose Up
-- +goose StatementBegin

CREATE TABLE "meta" (
    key     TEXT PRIMARY KEY NOT NULL,
    value   TEXT NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "meta";

-- +goose StatementEnd
