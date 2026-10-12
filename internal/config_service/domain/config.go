package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConfigNotFound         = errors.New("config not found")
	ErrCantFindPath           = errors.New("cannot find path of game config")
	ErrConfigAlreadyInstalled = errors.New("config already installed")
	ErrConfigAlreadyExists    = errors.New("config already exists in the database")
	ErrCacheMiss              = errors.New("cache miss")
)

type Config struct {
	ID             string    `json:"id"`
	Game           string    `json:"game"`
	Name           string    `json:"name"`
	AuthorNickName string    `json:"author_nickname"`
	IsGlobal       bool      `json:"is_global"`
	UserID         int64     `json:"user_id"`
	ConfigBody     string    `json:"config_body"`
	CreatedAt      time.Time `json:"created_at"`
}

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
	GetConfig(ctx context.Context, id string, game string, authorNickname string) (*Config, error)
	SaveMyConfig(ctx context.Context, cfg *Config) (string, error)
	DeleteConfig(ctx context.Context, id string, userID int64) (bool, error)
	ListConfigs(ctx context.Context, game string) ([]*Config, error)
}

type PostgresRepo interface {
	Get(ctx context.Context, id string, game string, authorNickname string) (*Config, error)
	Save(ctx context.Context, cfg *Config) (string, error)
	Delete(ctx context.Context, id string, userID int64) (bool, error)
	List(ctx context.Context, game string) ([]*Config, error)
}

type RedisRepo interface {
	GetConfig(ctx context.Context, key string) (*Config, error)
	SetConfig(ctx context.Context, key string, cfg *Config, ttl time.Duration) error
	DeleteConfig(ctx context.Context, key string) error
}

type KafkaPublisher interface {
	PublishInstall(ctx context.Context, event *ConfigInstalledEvent) error
	PublishSaved(ctx context.Context, event *ConfigSavedEvent) error
	PublishDeleted(ctx context.Context, event *ConfigDeletedEvent) error
	Close() error
}
