package storage

import (
	"NetScan/internal/backend/models"
	"context"
	"time"
)

// CheckStore интерфейс для работы с проверками
type CheckStore interface {
	Create(ctx context.Context, check *models.Check) error
	GetByID(ctx context.Context, id string) (*models.Check, error)
	UpdateStatus(ctx context.Context, id string, status models.CheckStatus) error
	List(ctx context.Context, limit, offset int) ([]*models.Check, error)
	GetCountByStatus(ctx context.Context, status models.CheckStatus) (int, error)
}

// AgentStore интерфейс для работы с агентами
type AgentStore interface {
	Create(ctx context.Context, agent *models.Agent) error
	GetByToken(ctx context.Context, token string) (*models.Agent, error)
	GetByID(ctx context.Context, id string) (*models.Agent, error)
	UpdateHeartbeat(ctx context.Context, agentID string) error
	UpdateStatus(ctx context.Context, agentID string, status models.AgentStatus) error
	UpdateCapabilities(ctx context.Context, agentID string, capabilities []string) error
	ListOnline(ctx context.Context) ([]*models.Agent, error)
}

// ResultStore интерфейс для работы с результатами
type ResultStore interface {
	Create(ctx context.Context, result *models.CheckResult) error
	GetByCheckID(ctx context.Context, checkID string) ([]*models.CheckResult, error)
	GetByAgentID(ctx context.Context, agentID string, limit int) ([]*models.CheckResult, error)
	GetLatestByCheckID(ctx context.Context, checkID string, limit int) ([]*models.CheckResult, error)
	DeleteOldResults(ctx context.Context, olderThan time.Time) (int64, error)
}

// Queue интерфейс для работы с очередью
type Queue interface {
	PushTask(ctx context.Context, queueName string, task interface{}) error
	PopTask(ctx context.Context, queueName string, timeout time.Duration) ([]byte, error)
	GetQueueLength(ctx context.Context, queueName string) (int64, error)
	Publish(ctx context.Context, channel string, message interface{}) error
	Close() error
}

// AgentTasksStore интерфейс для работы с тасками
type AgentTasksStore interface {
	CreateTask(ctx context.Context, task *models.AgentTask) error
	GetStuckTasks(ctx context.Context, timeout time.Duration) ([]*models.AgentTask, error)
	DeleteTask(ctx context.Context, agentID, checkID string) error
	DeleteTasksByAgent(ctx context.Context, agentID string) error
}
