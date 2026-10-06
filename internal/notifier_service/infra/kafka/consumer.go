package kafka

import (
	"fmt"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(reader *kafka.Reader) *KafkaConsumer {
	return &KafkaConsumer{
		reader: reader,
	}
}

func (k *KafkaConsumer) GetReader() *kafka.Reader {
	return k.reader
}

func (k *KafkaConsumer) Close() error {
	if err := k.reader.Close(); err != nil {
		return fmt.Errorf("failed to close kafka reader: %v", err)
	}
	return nil
}
