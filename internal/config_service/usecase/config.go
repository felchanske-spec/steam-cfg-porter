package usecase

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"steam-cfg-portet/internal/config_service/domain"
	"time"

	"golang.org/x/sync/singleflight"
)

type ConfigUseCase struct {
	pgRepo    domain.PostgresRepo
	redisRepo domain.RedisRepo
	kafkaRepo domain.KafkaPublisher
	sfGroup   *singleflight.Group
	cfgDir    string
}

func NewConfigUseCase(
	pgRepo domain.PostgresRepo,
	redisRepo domain.RedisRepo,
	kafkaRepo domain.KafkaPublisher,
	sf *singleflight.Group,
	cfgDir string,
) domain.UseCase {
	return &ConfigUseCase{
		pgRepo:    pgRepo,
		redisRepo: redisRepo,
		kafkaRepo: kafkaRepo,
		sfGroup:   sf,
		cfgDir:    cfgDir,
	}
}

func (uc *ConfigUseCase) GetConfig(ctx context.Context, id string, game string, authorNickname string) (*domain.Config, error) {
	localPath := filepath.Join(uc.cfgDir, game, id+".cfg")
	if _, err := os.Stat(localPath); err == nil {
		return nil, domain.ErrConfigAlreadyInstalled
	}

	cacheKey := "config:" + id

	cachedConfig, err := uc.redisRepo.GetConfig(ctx, cacheKey)
	if err == nil {
		if cachedConfig == nil {
			return nil, domain.ErrConfigNotFound
		}

		if errDir := os.MkdirAll(filepath.Dir(localPath), 0755); errDir != nil {
			return nil, errDir
		}

		if errWrite := os.WriteFile(localPath, []byte(cachedConfig.ConfigBody), 0644); errWrite != nil {
			return nil, errWrite
		}

		_ = uc.kafkaRepo.PublishInstall(ctx, &domain.ConfigInstalledEvent{
			Game:   cachedConfig.Game,
			ID:     cachedConfig.ID,
			Name:   cachedConfig.Name,
			UserID: cachedConfig.UserID,
		})
		return cachedConfig, nil
	}

	if !errors.Is(err, domain.ErrCacheMiss) {
		log.Printf("[Cache] Redis error: %v", err)
	}

	sfKey := "sf:" + cacheKey
	v, sfErr, shared := uc.sfGroup.Do(sfKey, func() (interface{}, error) {
		config, dbErr := uc.pgRepo.Get(ctx, id, game, authorNickname)
		if dbErr != nil {
			return nil, domain.ErrConfigNotFound
		}
		return config, nil
	})

	if shared {
		log.Printf("[SingleFlight] Combined concurrent calls for key: %s", sfKey)
	}

	if sfErr != nil {
		if errors.Is(sfErr, domain.ErrConfigNotFound) {
			_ = uc.redisRepo.SetConfig(ctx, cacheKey, nil, 5*time.Minute)
			return nil, domain.ErrConfigNotFound
		}
		return nil, sfErr
	}

	config := v.(*domain.Config)

	if errDir := os.MkdirAll(filepath.Dir(localPath), 0755); errDir != nil {
		return nil, errDir
	}

	if errWrite := os.WriteFile(localPath, []byte(config.ConfigBody), 0644); errWrite != nil {
		return nil, errWrite
	}

	_ = uc.redisRepo.SetConfig(ctx, cacheKey, config, 10*time.Minute)
	_ = uc.kafkaRepo.PublishInstall(ctx, &domain.ConfigInstalledEvent{
		Game:   config.Game,
		ID:     config.ID,
		Name:   config.Name,
		UserID: config.UserID,
	})

	return config, nil
}

func (uc *ConfigUseCase) SaveMyConfig(ctx context.Context, cfg *domain.Config) (string, error) {
	localPath := filepath.Join(uc.cfgDir, cfg.Game, cfg.ID+".cfg")

	fileBytes, err := os.ReadFile(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", domain.ErrCantFindPath
		}
		return "", err
	}

	cfg.ConfigBody = string(fileBytes)
	cfg.CreatedAt = time.Now()

	configID, err := uc.pgRepo.Save(ctx, cfg)
	if err != nil {
		return "", domain.ErrConfigAlreadyExists
	}

	cacheKey := "config:" + configID
	_ = uc.redisRepo.SetConfig(ctx, cacheKey, cfg, 10*time.Minute)

	_ = uc.kafkaRepo.PublishSaved(ctx, &domain.ConfigSavedEvent{
		Game: cfg.Game,
		ID:   configID,
		Name: cfg.Name,
	})

	return configID, nil
}

func (uc *ConfigUseCase) DeleteConfig(ctx context.Context, id string, userID int64) (bool, error) {
	var game, name, author string
	cacheKey := "config:" + id

	if cfg, err := uc.redisRepo.GetConfig(ctx, cacheKey); err == nil && cfg != nil {
		game, name, author = cfg.Game, cfg.Name, cfg.AuthorNickName
	} else {
		if cfgDb, errDb := uc.pgRepo.Get(ctx, id, "", ""); errDb == nil && cfgDb != nil {
			game, name, author = cfgDb.Game, cfgDb.Name, cfgDb.AuthorNickName
		}
	}

	deleted, err := uc.pgRepo.Delete(ctx, id, userID)
	if err != nil {
		return false, err
	}

	if !deleted {
		return false, nil
	}

	_ = uc.redisRepo.DeleteConfig(ctx, cacheKey)

	_ = uc.kafkaRepo.PublishDeleted(ctx, &domain.ConfigDeletedEvent{
		Game:           game,
		Name:           name,
		AuthorNickName: author,
	})

	return true, nil
}

func (uc *ConfigUseCase) ListConfigs(ctx context.Context, game string) ([]*domain.Config, error) {
	configs, err := uc.pgRepo.List(ctx, game)
	if err != nil {
		log.Printf("[UseCase] Failed to fetch configs list: %v", err)
		return nil, err
	}
	return configs, nil
}
