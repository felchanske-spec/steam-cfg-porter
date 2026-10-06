package usecase

import (
	"context"
	"steam-cfg-portet/internal/notifier_service/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifierUseCase_OnConfigInstalled(t *testing.T) {
	uc := NewNotifierUseCase()
	ctx := context.Background()

	t.Run("Success_LogInstalledEvent", func(t *testing.T) {
		event := &domain.ConfigInstalledEvent{
			Game:   "cs2",
			ID:     "preset-123",
			Name:   "Pro Visuals",
			UserID: 777,
		}

		err := uc.OnConfigInstalled(ctx, event)
		assert.NoError(t, err, "should not return an error when processing a valid event")
	})

	t.Run("Safe_NilEvent", func(t *testing.T) {
		err := uc.OnConfigInstalled(ctx, nil)
		assert.NoError(t, err, "should handle nil event gracefully without panic")
	})
}

func TestNotifierUseCase_OnConfigSaved(t *testing.T) {
	uc := NewNotifierUseCase()
	ctx := context.Background()

	t.Run("Success_LogSavedEvent", func(t *testing.T) {
		event := &domain.ConfigSavedEvent{
			Game: "dota2",
			ID:   "cfg-999",
			Name: "QuickCast Base",
		}

		err := uc.OnConfigSaved(ctx, event)
		assert.NoError(t, err)
	})
}

func TestNotifierUseCase_OnConfigDeleted(t *testing.T) {
	uc := NewNotifierUseCase()
	ctx := context.Background()

	t.Run("Success_LogDeletedEvent", func(t *testing.T) {
		event := &domain.ConfigDeletedEvent{
			Game:           "cs2",
			Name:           "Old Setup",
			AuthorNickName: "NaviFan",
		}

		err := uc.OnConfigDeleted(ctx, event)
		assert.NoError(t, err)
	})
}
