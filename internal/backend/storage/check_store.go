package storage

import (
	"NetScan/internal/backend/models"
	"NetScan/pkg/uuidutil"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type checkStore struct {
	pool *pgxpool.Pool
}

func NewCheckStore(pool *pgxpool.Pool) CheckStore {
	return &checkStore{pool: pool}
}

// Создаем новую проверку
func (s *checkStore) Create(ctx context.Context, check *models.Check) error {
	check.ID = uuidutil.New()
	check.CreatedAt = time.Now()
	check.UpdatedAt = time.Now()

	query := `INSERT INTO checks (id, type, target, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.pool.Exec(ctx, query,
		check.ID,
		check.Type,
		check.Target,
		check.Status,
		check.CreatedAt,
		check.UpdatedAt,
	)

	return err
}

// Возвращает по ID
func (s *checkStore) GetByID(ctx context.Context, id string) (*models.Check, error) {
	query := `SELECT id, type, target, status, created_at, updated_at
		FROM checks WHERE id = $1`

	var check models.Check
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&check.ID,
		&check.Type,
		&check.Target,
		&check.Status,
		&check.CreatedAt,
		&check.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return &check, err
}

// Обновляет статус проверки
func (s *checkStore) UpdateStatus(ctx context.Context, id string, status models.CheckStatus) error {
	query := `UPDATE checks SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := s.pool.Exec(ctx, query, status, time.Now(), id)
	return err
}

// Возвращаем список проверок
func (s *checkStore) List(ctx context.Context, limit, offset int) ([]*models.Check, error) {
	query := `
		SELECT id, type, target, status, created_at, updated_at
		FROM checks 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list checks: failed to query checks (limit=%d, offset=%d): %w", limit, offset, err)
	}
	defer rows.Close()

	var checks []*models.Check
	for rows.Next() {
		var check models.Check
		err := rows.Scan(
			&check.ID,
			&check.Type,
			&check.Target,
			&check.Status,
			&check.CreatedAt,
			&check.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("list checks: failed to scan row: %w", err)
		}
		checks = append(checks, &check)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list checks: row iteration error: %w", err)
	}

	return checks, nil

}

// возвращает количество проверок по статусу
func (s *checkStore) GetCountByStatus(ctx context.Context, status models.CheckStatus) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM checks 
		WHERE status = $1
	`

	var count int
	err := s.pool.QueryRow(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get check count by status %s: %w", status, err)
	}

	return count, nil
}
