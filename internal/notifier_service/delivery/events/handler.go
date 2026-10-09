package events

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"steam-cfg-portet/internal/notifier_service/domain"
	"steam-cfg-portet/internal/notifier_service/infra/kafka"
)

type EventHandler struct {
	uc       domain.UseCase
	consumer *kafka.KafkaConsumer
}

func NewEventHandler(uc domain.UseCase, consumer *kafka.KafkaConsumer) *EventHandler {
	return &EventHandler{
		uc:       uc,
		consumer: consumer,
	}
}

func (h *EventHandler) Run(ctx context.Context) error {
	log.Println("[Handler] Multi-topic worker engine started with individual manual-commit guarantees")

	go h.runSavedLoop(ctx)
	go h.runInstalledLoop(ctx)
	go h.runDeletedLoop(ctx)

	<-ctx.Done()
	return nil
}

func (h *EventHandler) runSavedLoop(ctx context.Context) {
	reader := h.consumer.SavedReader
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[Saved-Loop] Fetch error: %v", err)
			continue
		}

		event := &domain.ConfigSavedEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Saved-Loop] Unmarshal error: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		if err := h.uc.OnConfigSaved(ctx, event); err != nil {
			log.Printf("[Saved-Loop] Notification delivery failed, retrying: %v", err)
			continue
		}

		_ = reader.CommitMessages(ctx, msg)
	}
}

func (h *EventHandler) runInstalledLoop(ctx context.Context) {
	reader := h.consumer.InstalledReader
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[Installed-Loop] Fetch error: %v", err)
			continue
		}

		event := &domain.ConfigInstalledEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Installed-Loop] Unmarshal error: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		if err := h.uc.OnConfigInstalled(ctx, event); err != nil {
			log.Printf("[Installed-Loop] Notification delivery failed, retrying: %v", err)
			continue
		}

		_ = reader.CommitMessages(ctx, msg)
	}
}

func (h *EventHandler) runDeletedLoop(ctx context.Context) {
	reader := h.consumer.DeletedReader
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[Deleted-Loop] Fetch error: %v", err)
			continue
		}

		event := &domain.ConfigDeletedEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Deleted-Loop] Unmarshal error: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		if err := h.uc.OnConfigDeleted(ctx, event); err != nil {
			log.Printf("[Deleted-Loop] Notification delivery failed, retrying: %v", err)
			continue
		}

		_ = reader.CommitMessages(ctx, msg)
	}
}
