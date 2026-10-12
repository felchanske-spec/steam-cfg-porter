package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"steam-cfg-portet/internal/config_service/delivery/grpc"
	deliveryHTTP "steam-cfg-portet/internal/config_service/delivery/http"
	"steam-cfg-portet/internal/config_service/infra/db"
	"steam-cfg-portet/internal/config_service/infra/kafka"
	"steam-cfg-portet/internal/config_service/usecase"

	pb "steam-cfg-portet/api/pb"

	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/singleflight"
	googleGRPC "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Config struct {
	GRPCPort    string `env:"CONFIG_SERVICE_GRPC_PORT" envDefault:":50051"`
	HTTPPort    string `env:"CONFIG_SERVICE_HTTP_PORT" envDefault:":8080"`
	ConfigDir   string `env:"CONFIG_SERVICE_DIR" envDefault:"./tmp/steam_configs"`
	PostgresDSN string `env:"POSTGRES_DSN,required"`
	RedisAddr   string `env:"REDIS_ADDR,required"`
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

	pool, err := db.InitPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("[Main] Postgres init failed: %v", err)
	}
	defer pool.Close()

	redisClient, err := db.InitRedis(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatalf("[Main] Redis init failed: %v", err)
	}
	defer redisClient.Close()

	kafkaWriter := kafka.InitKafkaPublisher(cfg.KafkaBroker)
	defer kafkaWriter.Close()

	pgRepo := db.NewPostgresRepo(pool)
	redisRepo := db.NewRedisCache(redisClient)
	kafkaRepo := kafka.NewKafkaPublisher(kafkaWriter)
	sfGroup := &singleflight.Group{}

	cfgUseCase := usecase.NewConfigUseCase(pgRepo, redisRepo, kafkaRepo, sfGroup, cfg.ConfigDir)

	wg := &sync.WaitGroup{}
	grpcServerImpl := grpc.NewConfigGRPCServer(cfgUseCase, wg)
	grpcServer := googleGRPC.NewServer()
	pb.RegisterConfigServiceServer(grpcServer, grpcServerImpl)
	reflection.Register(grpcServer)

	healthHandler := deliveryHTTP.NewHealthHandler(pool, redisClient)
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health/liveness", healthHandler.Liveness)
	httpMux.HandleFunc("/health/readiness", healthHandler.Readiness)
	httpServer := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: httpMux,
	}

	go func() {
		log.Printf("[Main] HTTP Health Server is running on port %s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[Main] HTTP Server error: %v", err)
		}
	}()

	go func() {
		listener, err := net.Listen("tcp", cfg.GRPCPort)
		if err != nil {
			log.Fatalf("[Main] Failed to listen TCP port %s: %v", cfg.GRPCPort, err)
		}
		log.Printf("[Main] gRPC Server is running on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("[Main] gRPC Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[Main] Shutdown signal received. Starting Graceful Shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)

	grpcServer.GracefulStop()

	wg.Wait()

	log.Println("[Main] All active requests finished. Service stopped safely.")
}
