package main

import (
	"NetScan/internal/backend/dependencies"
	"NetScan/internal/backend/server"
	"NetScan/internal/config"
	"NetScan/pkg/logger"
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	// Создаем контейнер зависимостей
	container, err := dependencies.NewContainer(ctx, cfg, log)
	if err != nil {
		log.Error("Failed to create dependency container", err)
		os.Exit(1)
	}
	defer container.Close()

	// Создаем сервер
	srv := server.New(&server.Config{
		Port: cfg.Server.Port,
		Mode: cfg.Server.Mode,
	}, container)

	// Запускаем сервер в горутине
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Server failed to start", err)
			os.Exit(1)
		}
	}()

	// Ожидаем сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed: %s", err)
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
