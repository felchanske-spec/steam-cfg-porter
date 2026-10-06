package domain

import "context"

type ConfigInstalledEvent struct {
	Game   string `json:"game"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	UserID int64  `json:"user_id"`
}

type ConfigSavedEvent struct {
	Game string `json:"game"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ConfigDeletedEvent struct {
	Game           string `json:"game"`
	Name           string `json:"name"`
	AuthorNickName string `json:"author_nickname"`
}

type UseCase interface {
	OnConfigInstalled(ctx context.Context, event *ConfigInstalledEvent) error
	OnConfigSaved(ctx context.Context, event *ConfigSavedEvent) error
	OnConfigDeleted(ctx context.Context, event *ConfigDeletedEvent) error
}

type KafkaConsumer interface {
	Consume(ctx context.Context) error
}
