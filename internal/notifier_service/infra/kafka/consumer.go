package kafka

import (
	"fmt"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	SavedReader     *kafka.Reader
	InstalledReader *kafka.Reader
	DeletedReader   *kafka.Reader
}

func NewKafkaConsumer(saved, installed, deleted *kafka.Reader) *KafkaConsumer {
	return &KafkaConsumer{
		SavedReader:     saved,
		InstalledReader: installed,
		DeletedReader:   deleted,
	}
}

func (k *KafkaConsumer) Close() error {
	if err := k.SavedReader.Close(); err != nil {
		return fmt.Errorf("failed to close saved reader: %v", err)
	}
	if err := k.InstalledReader.Close(); err != nil {
		return fmt.Errorf("failed to close installed reader: %v", err)
	}
	if err := k.DeletedReader.Close(); err != nil {
		return fmt.Errorf("failed to close deleted reader: %v", err)
	}
	return nil
}
