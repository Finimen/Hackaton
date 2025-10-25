package storage

import (
	"NetScan/internal/backend/models"
	"NetScan/pkg/uuidutil"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAgentNotFound  = errors.New("agent not found")
	ErrInvalidAgentID = errors.New("invalid agent id")
	ErrDuplicateAgent = errors.New("agent with this token already exists")
)

type agentStore struct {
	pool *pgxpool.Pool
}

func NewAgentStore(pool *pgxpool.Pool) AgentStore {
	return &agentStore{pool: pool}
}

func (s *agentStore) Create(ctx context.Context, agent *models.Agent) error {
	agent.ID = uuidutil.New()
	agent.CreatedAt = time.Now()
	agent.Status = models.AgentStatusOffline

	query := `
		INSERT INTO agents (id, name, token, location, status, capabilities, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.pool.Exec(ctx, query,
		agent.ID,
		agent.Name,
		agent.Token,
		agent.Location,
		agent.Status,
		agent.Capabilities,
		agent.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

func (s *agentStore) GetByToken(ctx context.Context, token string) (*models.Agent, error) {
	query := `
		SELECT id, name, token, location, status, capabilities, last_heartbeat, created_at
		FROM agents 
		WHERE token = $1
	`

	var agent models.Agent
	var lastHeartbeat *time.Time

	err := s.pool.QueryRow(ctx, query, token).Scan(
		&agent.ID,
		&agent.Name,
		&agent.Token,
		&agent.Location,
		&agent.Status,
		&agent.Capabilities,
		&lastHeartbeat,
		&agent.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent by token: %w", err)
	}

	if lastHeartbeat != nil {
		agent.LastHeartbeat = *lastHeartbeat
	}

	return &agent, nil
}

func (s *agentStore) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	query := `
		SELECT id, name, token, location, status, capabilities, last_heartbeat, created_at
		FROM agents 
		WHERE id = $1
	`

	var agent models.Agent
	var lastHeartbeat *time.Time

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&agent.ID,
		&agent.Name,
		&agent.Token,
		&agent.Location,
		&agent.Status,
		&agent.Capabilities,
		&lastHeartbeat,
		&agent.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent by id %s: %w", id, err)
	}

	if lastHeartbeat != nil {
		agent.LastHeartbeat = *lastHeartbeat
	}

	return &agent, nil
}

func (s *agentStore) UpdateHeartbeat(ctx context.Context, agentID string) error {
	query := `
		UPDATE agents 
		SET last_heartbeat = $1, status = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := s.pool.Exec(ctx, query, time.Now(), models.AgentStatusOnline, time.Now(), agentID)
	if err != nil {
		return fmt.Errorf("failed to update agent heartbeat: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found with id %s", agentID)
	}

	return nil
}

func (s *agentStore) UpdateStatus(ctx context.Context, agentID string, status models.AgentStatus) error {
	query := `
		UPDATE agents 
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := s.pool.Exec(ctx, query, status, time.Now(), agentID)
	if err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found with id %s", agentID)
	}

	return nil
}

func (s *agentStore) ListOnline(ctx context.Context) ([]*models.Agent, error) {
	query := `
		SELECT id, name, location, capabilities, last_heartbeat
		FROM agents 
		WHERE status = $1
		ORDER BY last_heartbeat DESC
	`

	rows, err := s.pool.Query(ctx, query, models.AgentStatusOnline)
	if err != nil {
		return nil, fmt.Errorf("failed to query online agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var agent models.Agent
		var lastHeartbeat *time.Time

		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Location,
			&agent.Capabilities,
			&lastHeartbeat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %w", err)
		}

		if lastHeartbeat != nil {
			agent.LastHeartbeat = *lastHeartbeat
		}

		agent.Status = models.AgentStatusOnline
		agents = append(agents, &agent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agent rows: %w", err)
	}

	return agents, nil
}
