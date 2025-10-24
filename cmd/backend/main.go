package main

import (
	"NetScan/internal/config"
	"NetScan/pkg/logger"
	"log/slog"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config:", err)
	}

	// Настройка логирования
	logger.Setup(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	slog.Info("Starting NetScan backend",
		slog.String("version", cfg.App.Version),
		slog.Int("port", cfg.Server.Port),
	)

}
