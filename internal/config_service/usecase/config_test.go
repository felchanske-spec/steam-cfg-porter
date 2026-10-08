package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"steam-cfg-portet/internal/config_service/domain"
	"steam-cfg-portet/internal/config_service/domain/mocks"
	"steam-cfg-portet/internal/config_service/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/singleflight"
)

func TestConfigUseCase_GetConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "steam_configs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	testConfigID := "super-shroud-cfg"
	testGame := "cs2"
	testNickname := "shroud"

	mockConfigResult := &domain.Config{
		ID:             testConfigID,
		Game:           testGame,
		Name:           "Shroud Official Config",
		AuthorNickName: testNickname,
		UserID:         1337,
		ConfigBody:     "sensitivity 1.5\ncrosshair 1",
	}

	type mockBehavior func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher)

	tests := []struct {
		name           string
		setupLocalDisk func()
		mockBehavior   mockBehavior
		wantErr        error
		wantResult     *domain.Config
	}{
		{
			name:           "Success_FetchFromPostgresAndInstall",
			setupLocalDisk: func() {},
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {
				rm.On("GetConfig", ctx, "config:"+testConfigID).
					Return(nil, domain.ErrCacheMiss)

				pm.On("Get", ctx, testConfigID, testGame, testNickname).
					Return(mockConfigResult, nil)

				rm.On("SetConfig", ctx, "config:"+testConfigID, mockConfigResult, mock.AnythingOfType("time.Duration")).
					Return(nil)

				km.On("PublishInstall", ctx, mock.MatchedBy(func(e *domain.ConfigInstalledEvent) bool {
					return e.ID == mockConfigResult.ID && e.Game == mockConfigResult.Game
				})).Return(nil)
			},
			wantErr:    nil,
			wantResult: mockConfigResult,
		},
		{
			name: "Error_ConfigAlreadyInstalled",
			setupLocalDisk: func() {
				gameDir := filepath.Join(tmpDir, testGame)
				expectedFilePath := filepath.Join(gameDir, testConfigID+".cfg")
				_ = os.WriteFile(expectedFilePath, []byte("existing config"), 0644)
			},
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {},

			wantErr:    domain.ErrConfigAlreadyInstalled,
			wantResult: nil,
		},
		{
			name:           "Success_FetchFromRedisCache",
			setupLocalDisk: func() {},
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {
				rm.On("GetConfig", ctx, "config:"+testConfigID).
					Return(mockConfigResult, nil)

				km.On("PublishInstall", ctx, mock.MatchedBy(func(e *domain.ConfigInstalledEvent) bool {
					return e.ID == mockConfigResult.ID && e.Game == mockConfigResult.Game && e.Name == mockConfigResult.Name
				})).Return(nil)
			},
			wantErr:    nil,
			wantResult: mockConfigResult,
		},
		{
			name:           "Error_ConfigNotFoundInDB",
			setupLocalDisk: func() {},
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {
				rm.On("GetConfig", ctx, "config:"+testConfigID).
					Return(nil, domain.ErrCacheMiss)

				pm.On("Get", ctx, testConfigID, testGame, testNickname).
					Return(nil, domain.ErrConfigNotFound)

				rm.On("SetConfig", ctx, "config:"+testConfigID, (*domain.Config)(nil), mock.AnythingOfType("time.Duration")).
					Return(nil)
			},
			wantErr:    domain.ErrConfigNotFound,
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameDir := filepath.Join(tmpDir, testGame)
			_ = os.RemoveAll(gameDir)
			_ = os.MkdirAll(gameDir, 0755)

			tt.setupLocalDisk()

			mockPostgres := mocks.NewPostgresRepo(t)
			mockRedis := mocks.NewRedisRepo(t)
			mockKafka := mocks.NewKafkaPublisher(t)
			sfGroup := &singleflight.Group{}

			tt.mockBehavior(mockPostgres, mockRedis, mockKafka)

			uc := usecase.NewConfigUseCase(mockPostgres, mockRedis, mockKafka, sfGroup, tmpDir)

			res, err := uc.GetConfig(ctx, testConfigID, testGame, testNickname)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, res)

				expectedFilePath := filepath.Join(gameDir, testConfigID+".cfg")
				assert.FileExists(t, expectedFilePath, "The file must be physically saved on the user's disk")

				fileContent, errRead := os.ReadFile(expectedFilePath)
				assert.NoError(t, errRead)
				assert.Equal(t, tt.wantResult.ConfigBody, string(fileContent), "The file content must match the configuration body")
			}
		})
	}
}

func TestConfigUseCase_DeleteConfig(t *testing.T) {
	ctx := context.Background()
	testConfigID := "delete-me-cfg"
	testUserID := int64(228)
	testGame := "cs2"
	testName := "Trash Config"
	testAuthor := "simple"

	mockConfig := &domain.Config{
		ID:             testConfigID,
		Game:           testGame,
		Name:           testName,
		AuthorNickName: testAuthor,
		UserID:         testUserID,
	}

	type mockBehavior func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher)

	tests := []struct {
		name         string
		mockBehavior mockBehavior
		wantDeleted  bool
		wantErr      error
	}{
		{
			name: "Success_DeleteConfig",
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {
				rm.On("GetConfig", ctx, "config:"+testConfigID).
					Return(mockConfig, nil)

				pm.On("Delete", ctx, testConfigID, testUserID).
					Return(true, nil)

				rm.On("DeleteConfig", ctx, "config:"+testConfigID).
					Return(nil)

				km.On("PublishDeleted", ctx, &domain.ConfigDeletedEvent{
					Game:           testGame,
					Name:           testName,
					AuthorNickName: testAuthor,
				}).Return(nil)
			},
			wantDeleted: true,
			wantErr:     nil,
		},
		{
			name: "Error_DeleteForeignConfig",
			mockBehavior: func(pm *mocks.PostgresRepo, rm *mocks.RedisRepo, km *mocks.KafkaPublisher) {
				rm.On("GetConfig", ctx, "config:"+testConfigID).
					Return(mockConfig, nil)

				pm.On("Delete", ctx, testConfigID, testUserID).
					Return(false, nil)
			},
			wantDeleted: false,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := mocks.NewPostgresRepo(t)
			mockRedis := mocks.NewRedisRepo(t)
			mockKafka := mocks.NewKafkaPublisher(t)
			sfGroup := &singleflight.Group{}

			tt.mockBehavior(mockPostgres, mockRedis, mockKafka)

			uc := usecase.NewConfigUseCase(mockPostgres, mockRedis, mockKafka, sfGroup, "")

			deleted, err := uc.DeleteConfig(ctx, testConfigID, testUserID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantDeleted, deleted)
			}
		})
	}
}

func TestConfigUseCase_ListConfigs(t *testing.T) {
	ctx := context.Background()
	gameFilter := "dota2"
	dbErr := errors.New("connection pool exhausted")

	expectedConfigs := []*domain.Config{
		{
			ID:             "dota2-quickcast-standard",
			Game:           "dota2",
			Name:           "Standard Preset",
			AuthorNickName: "System",
			ConfigBody:     "unbindall",
		},
	}

	type mockBehavior func(pm *mocks.PostgresRepo)

	tests := []struct {
		name         string
		mockBehavior mockBehavior
		wantConfigs  []*domain.Config
		wantErr      error
	}{
		{
			name: "Success_ListConfigs",
			mockBehavior: func(pm *mocks.PostgresRepo) {
				pm.On("List", ctx, gameFilter).
					Return(expectedConfigs, nil)
			},
			wantConfigs: expectedConfigs,
			wantErr:     nil,
		},
		{
			name: "Error_DatabaseFailure",
			mockBehavior: func(pm *mocks.PostgresRepo) {
				pm.On("List", ctx, gameFilter).
					Return(nil, dbErr)
			},
			wantConfigs: nil,
			wantErr:     dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := mocks.NewPostgresRepo(t)
			// Redis и Kafka в ListConfigs не используются, но нужны для конструктора
			mockRedis := mocks.NewRedisRepo(t)
			mockKafka := mocks.NewKafkaPublisher(t)
			sfGroup := &singleflight.Group{}

			tt.mockBehavior(mockPostgres)

			uc := usecase.NewConfigUseCase(mockPostgres, mockRedis, mockKafka, sfGroup, "")

			configs, err := uc.ListConfigs(ctx, gameFilter)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, configs)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantConfigs, configs)
			}
		})
	}
}
