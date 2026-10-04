-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS configs (
    id VARCHAR(255) PRIMARY KEY,
    game VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    author_nickname VARCHAR(100) NOT NULL,
    is_global BOOLEAN NOT NULL DEFAULT FALSE,
    user_id BIGINT NOT NULL,
    config_body TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_configs_game_author ON configs(game, author_nickname);
CREATE INDEX IF NOT EXISTS idx_configs_user_id ON configs(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS configs;
-- +goose StatementEnd