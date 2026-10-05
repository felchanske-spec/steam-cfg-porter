package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaPublisher(kafkaBroker string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Balancer: &kafka.RoundRobin{},
		Transport: &kafka.Transport{
			DialTimeout: 5 * time.Second,
		},

		MaxAttempts:  5,
		WriteTimeout: 3 * time.Second,
		Async:        false,
	}
}
