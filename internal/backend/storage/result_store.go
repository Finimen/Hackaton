package storage

import (
	"NetScan/internal/backend/models"
	"NetScan/pkg/uuidutil"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type resultStore struct {
	pool *pgxpool.Pool
}

func NewResultStore(pool *pgxpool.Pool) ResultStore {
	return &resultStore{pool: pool}
}

func (s *resultStore) Create(ctx context.Context, result *models.CheckResult) error {
	result.ID = uuidutil.New()
	result.CreatedAt = time.Now()

	dataJSON, err := json.Marshal(result.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal result data: %w", err)
	}

	query := `
		INSERT INTO check_results (id, check_id, agent_id, success, data, error, duration, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = s.pool.Exec(ctx, query,
		result.ID,
		result.CheckID,
		result.AgentID,
		result.Success,
		dataJSON,
		result.Error,
		result.Duration,
		result.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create check result: %w", err)
	}

	return nil
}

func (s *resultStore) GetByCheckID(ctx context.Context, checkID string) ([]*models.CheckResult, error) {
	query := `
		SELECT id, check_id, agent_id, success, data, error, duration, created_at
		FROM check_results 
		WHERE check_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, checkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query check results: %w", err)
	}
	defer rows.Close()

	return s.scanResults(rows)
}

func (s *resultStore) GetByAgentID(ctx context.Context, agentID string, limit int) ([]*models.CheckResult, error) {
	query := `
		SELECT id, check_id, agent_id, success, data, error, duration, created_at
		FROM check_results 
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent results: %w", err)
	}
	defer rows.Close()

	return s.scanResults(rows)
}

// возвращает последние N результатов для конкретной проверки
func (s *resultStore) GetLatestByCheckID(ctx context.Context, checkID string, limit int) ([]*models.CheckResult, error) {
	query := `
		SELECT id, check_id, agent_id, success, data, error, duration, created_at
		FROM check_results 
		WHERE check_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, checkID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest check results: %w", err)
	}
	defer rows.Close()

	return s.scanResults(rows)
}

func (s *resultStore) DeleteOldResults(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `
		DELETE FROM check_results 
		WHERE created_at < $1
	`

	result, err := s.pool.Exec(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old results: %w", err)
	}

	return result.RowsAffected(), nil
}

// вынесенная общая логика сканирования результатов
func (s *resultStore) scanResults(rows pgx.Rows) ([]*models.CheckResult, error) {
	var results []*models.CheckResult

	for rows.Next() {
		result, err := s.scanSingleResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating result rows: %w", err)
	}

	return results, nil
}

// сканирует одну строку результата
func (s *resultStore) scanSingleResult(rows pgx.Rows) (*models.CheckResult, error) {
	var result models.CheckResult
	var dataJSON []byte

	err := rows.Scan(
		&result.ID,
		&result.CheckID,
		&result.AgentID,
		&result.Success,
		&dataJSON,
		&result.Error,
		&result.Duration,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan result row: %w", err)
	}

	// Декодируем JSON данные, только если они не пустые
	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &result.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result data: %w", err)
		}
	}

	return &result, nil
}
