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
	KafkaBroker      string `env:"KAFKA_BROKER,required"`
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramChatID   string `env:"TELEGRAM_CHAT_ID,required"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("[Main] Failed to parse config: %v", err)
	}
	log.Println("[Main] Configuration loaded successfully")

	savedReader := kafka.InitSavedConsumer(cfg.KafkaBroker)
	installedReader := kafka.InitInstalledConsumer(cfg.KafkaBroker)
	deletedReader := kafka.InitDeletedConsumer(cfg.KafkaBroker)

	defer savedReader.Close()
	defer installedReader.Close()
	defer deletedReader.Close()

	kafkaRepo := kafka.NewKafkaConsumer(savedReader, installedReader, deletedReader)

	notifierUseCase := usecase.NewNotifierUseCase(cfg.TelegramBotToken, cfg.TelegramChatID)
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
