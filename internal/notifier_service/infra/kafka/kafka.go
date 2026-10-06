package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaConsumer(kafkaBroker string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		GroupID: "notifie-group",
		GroupTopics: []string{
			"config-installed",
			"config-saved",
			"config-deleted",
		},

		MaxBytes: 10e6,
		Dialer: &kafka.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
		},
		QueueCapacity:  100,
		CommitInterval: time.Second,
	})
}
