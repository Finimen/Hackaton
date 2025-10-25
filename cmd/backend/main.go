package main

import (
	"NetScan/internal/backend/storage"
	"NetScan/internal/config"
	"NetScan/pkg/logger"
	"context"
	"log"
	"log/slog"
	"os"
	"time"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config %s", err)
	}

	// Настройка логирования
	log := logger.Setup(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	log.Info("Starting NetScan backend",
		slog.String("Name", cfg.App.Name),
		slog.String("version", cfg.App.Version),
		slog.Int("port", cfg.Server.Port),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Подключение к БД
	db, err := storage.NewPostgres(ctx, &cfg.Database, log)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// Подключение к Redis Queue
	redisQueue, err := storage.NewRedisQueue(&cfg.Redis, log)
	if err != nil {
		log.Error("failed to connect to redis queue", slog.String("error", err.Error()))
		os.Exit(1)
	}

	_ = redisQueue
}
