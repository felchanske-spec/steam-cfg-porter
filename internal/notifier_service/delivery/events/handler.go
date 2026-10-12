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
	log.Println("[Handler] Multi-topic worker engine initialized successfully in strict manual-commit mode")
	log.Println("[Handler] Spawning 3 independent background routines for topic streams...")

	go h.runSavedLoop(ctx)
	go h.runInstalledLoop(ctx)
	go h.runDeletedLoop(ctx)

	<-ctx.Done()
	log.Println("[Handler] Context canceled. Stopping event handler engine...")
	return nil
}

func (h *EventHandler) runSavedLoop(ctx context.Context) {
	reader := h.consumer.SavedReader
	log.Println("[Saved-Loop] Persistent goroutine actively listening to topic 'config-saved'...")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("[Saved-Loop] Reader connection closed gracefully due to context cancellation")
				return
			}
			log.Printf("[Saved-Loop-ERROR] Failed to fetch message from Kafka partition: %v", err)
			continue
		}

		log.Printf("[Saved-Loop] 📥 Caught raw message event from Kafka! Partition: %d | Offset: %d", msg.Partition, msg.Offset)

		event := &domain.ConfigSavedEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Saved-Loop-ERROR] JSON Unmarshal crashed. Skipping message corrupt byte-payload: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[NOTIFICATION] New Config Saved in Cloud! Game: %s | Config ID: %s | Name: %s",
			event.Game, event.ID, event.Name)

		log.Println("[Saved-Loop] Invoking UseCase to compile Markdown and dispatch HTTP POST to Telegram API...")
		if err := h.uc.OnConfigSaved(ctx, event); err != nil {
			log.Printf("[Saved-Loop-ERROR] Telegram dispatch failed: %v. [CRITICAL] Skipping Kafka commit to trigger auto-retry boundary!", err)
			continue
		}

		log.Printf("[Saved-Loop] Telegram API returned 200 OK! Push delivered successfully. Executing manual offset commit...")
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[Saved-Loop-ERROR] Critical: Failed to synchronize offset with Kafka broker: %v", err)
		} else {
			log.Printf("[Saved-Loop] Offset %d successfully synchronized and committed to Kafka!", msg.Offset)
		}
	}
}

func (h *EventHandler) runInstalledLoop(ctx context.Context) {
	reader := h.consumer.InstalledReader
	log.Println("[Installed-Loop] Persistent goroutine actively listening to topic 'config-installed'...")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("[Installed-Loop] Reader connection closed gracefully")
				return
			}
			log.Printf("[Installed-Loop-ERROR] Fetch error: %v", err)
			continue
		}

		log.Printf("[Installed-Loop] Caught raw message event from Kafka! Partition: %d | Offset: %d", msg.Partition, msg.Offset)

		event := &domain.ConfigInstalledEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Installed-Loop-ERROR] Unmarshal error: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[NOTIFICATION] Config Installed! User ID: %d | Game: %s | Preset Name: %s",
			event.UserID, event.Game, event.Name)

		log.Println("[Installed-Loop] Dispatching HTTP POST to Telegram API...")
		if err := h.uc.OnConfigInstalled(ctx, event); err != nil {
			log.Printf("[Installed-Loop-ERROR] Telegram dispatch failed: %v. Skipping commit!", err)
			continue
		}

		log.Printf("[Installed-Loop] Telegram API returned 200 OK! Executing manual offset commit...")
		_ = reader.CommitMessages(ctx, msg)
		log.Printf("[Installed-Loop] Offset %d committed to Kafka!", msg.Offset)
	}
}

func (h *EventHandler) runDeletedLoop(ctx context.Context) {
	reader := h.consumer.DeletedReader
	log.Println("[Deleted-Loop] Persistent goroutine actively listening to topic 'config-deleted'...")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("[Deleted-Loop] Reader connection closed gracefully")
				return
			}
			log.Printf("[Deleted-Loop-ERROR] Fetch error: %v", err)
			continue
		}

		log.Printf("[Deleted-Loop] Caught raw message event from Kafka! Partition: %d | Offset: %d", msg.Partition, msg.Offset)

		event := &domain.ConfigDeletedEvent{}
		if err := json.Unmarshal(msg.Value, event); err != nil {
			log.Printf("[Deleted-Loop-ERROR] Unmarshal error: %v", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[NOTIFICATION] Config Deleted from Cloud! Game: %s | Name: %s | Author: %s",
			event.Game, event.Name, event.AuthorNickName)

		log.Println("[Deleted-Loop] Dispatching HTTP POST to Telegram API...")
		if err := h.uc.OnConfigDeleted(ctx, event); err != nil {
			log.Printf("[Deleted-Loop-ERROR] Telegram dispatch failed: %v. Skipping commit!", err)
			continue
		}

		log.Printf("[Deleted-Loop] Telegram API returned 200 OK! Executing manual offset commit...")
		_ = reader.CommitMessages(ctx, msg)
		log.Printf("[Deleted-Loop] Offset %d committed to Kafka!", msg.Offset)
	}
}
