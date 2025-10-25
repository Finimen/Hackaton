package storage

import (
	"NetScan/internal/backend/models"
	"NetScan/pkg/uuidutil"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type CheckStore struct {
	db *sql.DB
}

func NewCheckStore(db *sql.DB) *CheckStore {
	return &CheckStore{db: db}
}

// Создаем новую проверку
func (s *CheckStore) Create(ctx context.Context, check *models.Check) error {
	check.ID = uuidutil.New()
	check.CreatedAt = time.Now()
	check.UpdatedAt = time.Now()

	query := `INSERT INTO checks (id, type, target, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.ExecContext(ctx, query,
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
func (s *CheckStore) GetByID(ctx context.Context, id string) (*models.Check, error) {
	query := `SELECT id, type, target, status, created_at, updated_at
		FROM checks WHERE id = $1`

	var check models.Check
	err := s.db.QueryRowContext(ctx, query, id).Scan(
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
func (s *CheckStore) UpdateStatus(ctx context.Context, id string, status models.CheckStatus) error {
	query := `UPDATE checks SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// Возвращаем список проверок
func (s *CheckStore) List(ctx context.Context, limit, offset int) ([]*models.Check, error) {
	query := `
		SELECT id, type, target, status, created_at, updated_at
		FROM checks 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
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
