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

var (
	ErrAgentNotFound  = errors.New("agent not found")
	ErrInvalidAgentID = errors.New("invalid agent id")
	ErrDuplicateAgent = errors.New("agent with this token already exists")
)

type AgentStore struct {
	db *sql.DB
}

func NewAgentStore(db *sql.DB) *AgentStore {
	return &AgentStore{db: db}
}

func (s *AgentStore) Create(ctx context.Context, agent *models.Agent) error {
	if agent == nil {
		return fmt.Errorf("agent is nil")
	}

	agent.ID = uuidutil.New()
	agent.CreatedAt = time.Now()
	agent.Status = models.AgentStatusOffline

	query := `
		INSERT INTO agents (id, name, token, location, status, capabilities, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.db.ExecContext(ctx, query,
		agent.ID,
		agent.Name,
		agent.Token,
		agent.Location,
		agent.Status,
		agent.Capabilities,
		agent.CreatedAt,
	)
	if err != nil {

	}

}
