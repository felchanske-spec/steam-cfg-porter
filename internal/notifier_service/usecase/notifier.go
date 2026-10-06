package usecase

import (
	"context"
	"log"
	"steam-cfg-portet/internal/notifier_service/domain"
)

type NotifierUseCase struct{}

func NewNotifierUseCase() domain.UseCase {
	return &NotifierUseCase{}
}

func (n *NotifierUseCase) OnConfigInstalled(ctx context.Context, event *domain.ConfigInstalledEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] Config Installed! User ID: %d | Game: %s | Config ID: %s | Name: %s",
		event.UserID, event.Game, event.ID, event.Name)

	return nil
}

func (n *NotifierUseCase) OnConfigSaved(ctx context.Context, event *domain.ConfigSavedEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] New Config Saved in Cloud! Game: %s | Config ID: %s | Name: %s",
		event.Game, event.ID, event.Name)

	return nil
}

func (n *NotifierUseCase) OnConfigDeleted(ctx context.Context, event *domain.ConfigDeletedEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] Config Deleted by Author! Game: %s | Name: %s | Author: %s",
		event.Game, event.Name, event.AuthorNickName)

	return nil
}
