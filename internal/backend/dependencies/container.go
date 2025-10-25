package dependencies

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"NetScan/internal/backend/services"
	"NetScan/internal/backend/storage"
	"NetScan/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Container контейнер зависимостей
type Container struct {
	// Config
	Config *config.Config

	// Logger
	Logger *slog.Logger

	// Storage
	CheckStore      storage.CheckStore
	AgentStore      storage.AgentStore
	ResultStore     storage.ResultStore
	AgentTasksStore storage.AgentTasksStore
	Queue           storage.Queue

	// Services
	CheckService *services.CheckService
	AgentService *services.AgentService
	QueueService *services.QueueService

	// Database connections
	DB    *pgxpool.Pool
	Redis *redis.Client
}

// NewContainer создает и инициализирует контейнер зависимостей
func NewContainer(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Container, error) {
	container := &Container{
		Config: cfg,
		Logger: log,
	}

	// Инициализация зависимостей
	if err := container.initDatabase(ctx); err != nil {
		return nil, err
	}

	if err := container.initRedis(); err != nil {
		return nil, err
	}

	if err := container.initStorage(); err != nil {
		return nil, err
	}

	if err := container.initServices(); err != nil {
		return nil, err
	}

	slog.Info("Dependency container initialized successfully")
	return container, nil
}

func (c *Container) initDatabase(ctx context.Context) error {
	db, err := storage.NewPostgres(ctx, &c.Config.Database, c.Logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	c.DB = db
	return nil
}

func (c *Container) initRedis() error {
	queue, err := storage.NewRedisQueue(&c.Config.Redis, c.Logger)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	c.Queue = queue
	return nil
}

func (c *Container) initStorage() error {
	c.CheckStore = storage.NewCheckStore(c.DB)
	c.AgentStore = storage.NewAgentStore(c.DB)
	c.ResultStore = storage.NewResultStore(c.DB)
	c.AgentTasksStore = storage.NewAgentTasksStore(c.DB)
	return nil
}

func (c *Container) initServices() error {
	logger := slog.Default()

	c.CheckService = services.NewCheckService(
		c.CheckStore,
		c.AgentStore,
		c.ResultStore,
		c.Queue,
		services.CheckServiceConfig{
			TaskTimeout: 30 * time.Second,
		},
		logger.With("service", "check"),
	)

	c.AgentService = services.NewAgentService(
		c.AgentStore,
		c.CheckStore,
		c.ResultStore,
		services.AgentServiceConfig{
			HeartbeatTimeout: 2 * time.Minute,
		},
		logger.With("service", "agent"),
	)

	c.QueueService = services.NewQueueService(
		c.Queue,
		c.CheckStore,
		c.AgentStore,
		c.AgentTasksStore,
		c.ResultStore,
		services.QueueServiceConfig{
			TaskTimeout:      30 * time.Second,
			PollInterval:     5 * time.Second,
			MaxRetries:       3,
			RetryDelay:       1 * time.Second,
			StuckTaskTimeout: 10 * time.Minute,
		},
		logger.With("service", "queue"),
	)

	return nil
}

// Close закрывает все соединения
func (c *Container) Close() error {
	var errors []error

	if c.DB != nil {
		c.DB.Close()
	}

	if c.Queue != nil {
		if err := c.Queue.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing dependencies: %v", errors)
	}

	return nil
}
