package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"steam-cfg-portet/internal/config_service/domain"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(writer *kafka.Writer) domain.KafkaPublisher {
	return &KafkaPublisher{writer: writer}
}

func (k *KafkaPublisher) PublishInstall(ctx context.Context, event *domain.ConfigInstalledEvent) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal config installed event: %v", err)
	}

	err = k.writer.WriteMessages(ctx, kafka.Message{
		Topic: "config-installed",
		Value: eventBytes,
	})
	if err != nil {
		return fmt.Errorf("kafka publish install failed: %v", err)
	}
	return nil
}

func (k *KafkaPublisher) PublishSaved(ctx context.Context, event *domain.ConfigSavedEvent) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal config saved event: %v", err)
	}

	err = k.writer.WriteMessages(ctx, kafka.Message{
		Topic: "config-saved",
		Value: eventBytes,
	})
	if err != nil {
		log.Printf("[KAFKA-ERROR] Failed to write message: %v", err)
		return fmt.Errorf("kafka publish saved failed: %v", err)
	}
	return nil
}

func (k *KafkaPublisher) PublishDeleted(ctx context.Context, event *domain.ConfigDeletedEvent) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal config deleted event: %v", err)
	}

	err = k.writer.WriteMessages(ctx, kafka.Message{
		Topic: "config-deleted",
		Value: eventBytes,
	})
	if err != nil {
		return fmt.Errorf("kafka publish deleted failed: %v", err)
	}
	return nil
}

func (k *KafkaPublisher) Close() error {
	return k.writer.Close()
}
