package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"steam-cfg-portet/internal/notifier_service/delivery/events"
	"steam-cfg-portet/internal/notifier_service/infra/kafka"
	"steam-cfg-portet/internal/notifier_service/usecase"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	KafkaBroker string `env:"KAFKA_BROKER,required"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("[Main] Failed to parse config: %v", err)
	}
	log.Println("[Main] Configuration loaded successfully")

	kafkaReader := kafka.InitKafkaConsumer(cfg.KafkaBroker)
	defer kafkaReader.Close()

	kafkaRepo := kafka.NewKafkaConsumer(kafkaReader)

	notifierUseCase := usecase.NewNotifierUseCase()
	eventHandler := events.NewEventHandler(notifierUseCase, kafkaRepo)

	go func() {
		log.Println("[Main] Kafka Consumer worker started in background")
		if err := eventHandler.Run(ctx); err != nil {
			log.Printf("[Main] Consumer worker runtime error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[Main] Shutdown signal received. Starting Graceful Shutdown...")

	log.Println("[Main] Notifier service stopped safely.")
}
