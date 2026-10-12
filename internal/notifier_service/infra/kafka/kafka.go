package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func newReaderForTopic(kafkaBroker, groupID, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		GroupID: groupID,
		Topic:   topic,

		MaxBytes: 10e6,
		Dialer: &kafka.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
		},
		QueueCapacity: 100,
	})
}

func InitSavedConsumer(kafkaBroker string) *kafka.Reader {
	return newReaderForTopic(kafkaBroker, "notifier-saved-group", "config-saved")
}

func InitInstalledConsumer(kafkaBroker string) *kafka.Reader {
	return newReaderForTopic(kafkaBroker, "notifier-installed-group", "config-installed")
}

func InitDeletedConsumer(kafkaBroker string) *kafka.Reader {
	return newReaderForTopic(kafkaBroker, "notifier-deleted-group", "config-deleted")
}
