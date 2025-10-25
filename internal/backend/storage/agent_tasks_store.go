package storage

import (
	"NetScan/internal/backend/models"
	"NetScan/pkg/uuidutil"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type agentTasksStore struct {
	pool *pgxpool.Pool
}

func NewAgentTasksStore(pool *pgxpool.Pool) AgentTasksStore {
	return &agentTasksStore{pool: pool}
}

func (s *agentTasksStore) CreateTask(ctx context.Context, task *models.AgentTask) error {
	task.ID = uuidutil.New()

	query := `
		INSERT INTO agent_tasks (id, agent_id, check_id, task_data, taken_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, query,
		task.ID,
		task.AgentID,
		task.CheckID,
		task.TaskData,
		task.TakenAt,
		task.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent task: %w", err)
	}

	return nil
}

func (s *agentTasksStore) GetStuckTasks(ctx context.Context, timeout time.Duration) ([]*models.AgentTask, error) {
	query := `
		SELECT id, agent_id, check_id, task_data, taken_at, status, created_at
		FROM agent_tasks 
		WHERE taken_at < $1 AND status = 'processing'
		ORDER BY taken_at ASC
	`

	cutoffTime := time.Now().Add(-timeout)
	rows, err := s.pool.Query(ctx, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query stuck tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.AgentTask
	for rows.Next() {
		var task models.AgentTask
		err := rows.Scan(
			&task.ID,
			&task.AgentID,
			&task.CheckID,
			&task.TaskData,
			&task.TakenAt,
			&task.Status,
			&task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent task row: %w", err)
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agent task rows: %w", err)
	}

	return tasks, nil
}

func (s *agentTasksStore) DeleteTask(ctx context.Context, agentID, checkID string) error {
	query := `
		DELETE FROM agent_tasks 
		WHERE agent_id = $1 AND check_id = $2
	`

	result, err := s.pool.Exec(ctx, query, agentID, checkID)
	if err != nil {
		return fmt.Errorf("failed to delete agent task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent task not found: agent=%s, check=%s", agentID, checkID)
	}

	return nil
}

func (s *agentTasksStore) DeleteTasksByAgent(ctx context.Context, agentID string) error {
	query := `
		DELETE FROM agent_tasks 
		WHERE agent_id = $1
	`

	_, err := s.pool.Exec(ctx, query, agentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent tasks: %w", err)
	}

	return nil
}
