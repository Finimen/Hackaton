package storage

import (
	"NetScan/internal/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisQueue struct {
	client *redis.Client
}

func NewRedisQueue(cfg *config.RedisConfig, log *slog.Logger) (Queue, error) {
	client := redis.NewClient(cfg.GetRedisOptions())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Error("failed to connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Connected to Redis")
	return &redisQueue{client: client}, nil
}

// Добавляем элемент в очередь
func (r *redisQueue) PushTask(ctx context.Context, queueName string, task interface{}) error {
	var data []byte
	var err error

	switch v := task.(type) {
	case []byte:
		// Если уже получили []byte, используем как есть
		data = v
		slog.Debug("Pushing raw bytes to Redis",
			"queue", queueName,
			"data", string(v),
			"length", len(v),
		)
	case string:
		// Если строка, конвертируем в []byte
		data = []byte(v)
		slog.Debug("Pushing string to Redis",
			"queue", queueName,
			"data", v,
			"length", len(v),
		)
	default:
		// Для других типов маршалим в JSON
		data, err = json.Marshal(task)
		if err != nil {
			return fmt.Errorf("failed to marshal the task: %w", err)
		}
		slog.Debug("Pushing object to Redis",
			"queue", queueName,
			"data", string(data),
			"length", len(data),
		)
	}

	// ВАЖНО: передаем как []byte, чтобы Redis не разбирал структуру
	return r.client.LPush(ctx, queueName, data).Err()
}

// Удаляем элемент из очереди
func (r *redisQueue) PopTask(ctx context.Context, queueName string, timeout time.Duration) ([]byte, error) {
	result, err := r.client.BRPop(ctx, time.Second, queueName).Result()

	if err != nil {
		// Проверяем ошибки контекста
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}

		// Очередь пуста
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, fmt.Errorf("redis BRPop failed: %w", err)
	}

	// Проверяем результат на корректность
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid BRPop result: expected 2 elements, got %d", len(result))
	}

	return []byte(result[1]), nil
}

func (r *redisQueue) Close() error {
	return r.client.Close()
}

func (r *redisQueue) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	return r.client.LLen(ctx, queueName).Result()
}

func (r *redisQueue) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, channel, data).Err()
}
