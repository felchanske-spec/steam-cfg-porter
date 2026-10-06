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
	for {
		msg, err := h.consumer.GetReader().ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			log.Printf("[Handler] Error reading message from Kafka: %v", err)
			continue
		}

		switch msg.Topic {
		case "config-installed":
			event := &domain.ConfigInstalledEvent{}
			if err := json.Unmarshal(msg.Value, event); err != nil {
				log.Printf("[Handler] Failed to unmarshal installed event: %v", err)
				continue
			}
			_ = h.uc.OnConfigInstalled(ctx, event)

		case "config-saved":
			event := &domain.ConfigSavedEvent{}
			if err := json.Unmarshal(msg.Value, event); err != nil {
				log.Printf("[Handler] Failed to unmarshal saved event: %v", err)
				continue
			}
			_ = h.uc.OnConfigSaved(ctx, event)

		case "config-deleted":
			event := &domain.ConfigDeletedEvent{}
			if err := json.Unmarshal(msg.Value, event); err != nil {
				log.Printf("[Handler] Failed to unmarshal deleted event: %v", err)
				continue
			}
			_ = h.uc.OnConfigDeleted(ctx, event)
		}
	}
}
