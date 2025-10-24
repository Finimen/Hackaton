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

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(cfg *config.RedisConfig, log *slog.Logger) (*RedisQueue, error) {
	client := redis.NewClient(cfg.GetRedisOptions())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Error("failed to connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Connected to Redis")
	return &RedisQueue{client: client}, nil
}

// Добавляем элемент в очередь
func (r *RedisQueue) PushTask(ctx context.Context, queueName string, task interface{}) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal the task: %w", err)
	}

	return r.client.LPush(ctx, queueName, data).Err()
}

// Удаляем элемент из очереди
func (r *RedisQueue) PopTask(ctx context.Context, queueName string, timeout time.Duration) (interface{}, error) {
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
