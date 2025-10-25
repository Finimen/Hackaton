package services

import (
	"NetScan/internal/backend/models"
	"NetScan/internal/backend/storage"
	"NetScan/pkg/uuidutil"
	"NetScan/pkg/validator"
	"context"
	"fmt"
	"log/slog"
	"time"
)

type AgentService struct {
	agentStore  storage.AgentStore
	checkStore  storage.CheckStore
	resultStore storage.ResultStore
	logger      *slog.Logger
}

type AgentServiceConfig struct {
	HeartbeatTimeout time.Duration
}

func NewAgentService(
	agentStore storage.AgentStore,
	checkStore storage.CheckStore,
	resultStore storage.ResultStore,
	cfg AgentServiceConfig,
	logger *slog.Logger,
) *AgentService {

	timeout := cfg.HeartbeatTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &AgentService{
		agentStore:  agentStore,
		checkStore:  checkStore,
		resultStore: resultStore,
		logger:      logger,
	}
}

// регистрирует нового агента в системе
func (s *AgentService) RegisterAgent(ctx context.Context, req *models.RegisterRequest) (*models.Agent, string, error) {
	s.logger.Info("registering new agent",
		"name", req.Name,
		"location", req.Location,
		"capabilities", req.Capabilities,
	)

	// Валидация входных данных
	if req.Name == "" {
		s.logger.Warn("registration failed: empty agent name")
		return nil, "", fmt.Errorf("agent name is required")
	}

	if req.Location == "" {
		s.logger.Warn("registration failed: empty agent location",
			"name", req.Name,
		)
		return nil, "", fmt.Errorf("agent location is required")
	}

	// Валидация capabilities
	for _, capability := range req.Capabilities {
		if !validator.ValidateCheckType(capability) {
			s.logger.Warn("registration failed: invalid capability",
				"name", req.Name,
				"invalid_capability", capability,
			)
			return nil, "", fmt.Errorf("invalid capability: %s", capability)
		}
	}

	// Генерируем уникальный токен
	token := uuidutil.New()

	agent := &models.Agent{
		Name:         req.Name,
		Location:     req.Location,
		Token:        token,
		Capabilities: req.Capabilities,
		Status:       models.AgentStatusOffline,
	}

	if err := s.agentStore.Create(ctx, agent); err != nil {
		s.logger.Error("failed to create agent in storage",
			"error", err,
			"name", req.Name,
			"location", req.Location,
		)
		return nil, "", fmt.Errorf("failed to register agent: %w", err)
	}

	s.logger.Info("agent registered successfully",
		"agent_id", agent.ID,
		"name", agent.Name,
		"location", agent.Location,
		"capabilities_count", len(agent.Capabilities),
	)

	return agent, token, nil
}

// AuthenticateAgent аутентифицирует агента по токену
func (s *AgentService) AuthenticateAgent(ctx context.Context, token string) (*models.Agent, error) {
	s.logger.Debug("authenticating agent", "token_length", len(token))

	if token == "" {
		s.logger.Warn("authentication failed: empty token")
		return nil, fmt.Errorf("token is required")
	}

	agent, err := s.agentStore.GetByToken(ctx, token)
	if err != nil {
		s.logger.Error("failed to get agent by token",
			"error", err,
		)
		return nil, fmt.Errorf("failed to authenticate agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("authentication failed: invalid token")
		return nil, nil
	}

	s.logger.Debug("agent authenticated successfully",
		"agent_id", agent.ID,
		"name", agent.Name,
		"status", agent.Status,
	)

	return agent, nil
}

// обновляет время последней активности агента
func (s *AgentService) UpdateHeartbeat(ctx context.Context, agentID string, load int) error {
	s.logger.Debug("updating agent heartbeat",
		"agent_id", agentID,
		"load", load,
	)

	if agentID == "" {
		s.logger.Warn("heartbeat update failed: empty agent ID")
		return fmt.Errorf("agent ID is required")
	}

	// Проверяем существование агента
	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent for heartbeat",
			"error", err,
			"agent_id", agentID,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("heartbeat update failed: agent not found",
			"agent_id", agentID,
		)
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Обновляем heartbeat
	if err := s.agentStore.UpdateHeartbeat(ctx, agentID); err != nil {
		s.logger.Error("failed to update agent heartbeat in storage",
			"error", err,
			"agent_id", agentID,
		)
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	s.logger.Debug("agent heartbeat updated",
		"agent_id", agentID,
		"load", load,
		"name", agent.Name,
	)

	return nil
}

// обновляет статус агента
func (s *AgentService) UpdateAgentStatus(ctx context.Context, agentID string, status models.AgentStatus) error {
	s.logger.Info("updating agent status",
		"agent_id", agentID,
		"new_status", status,
	)

	if agentID == "" {
		s.logger.Warn("status update failed: empty agent ID")
		return fmt.Errorf("agent ID is required")
	}

	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent for status update",
			"error", err,
			"agent_id", agentID,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("status update failed: agent not found",
			"agent_id", agentID,
		)
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if err := s.agentStore.UpdateStatus(ctx, agentID, status); err != nil {
		s.logger.Error("failed to update agent status in storage",
			"error", err,
			"agent_id", agentID,
			"status", status,
		)
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	s.logger.Info("agent status updated successfully",
		"agent_id", agentID,
		"name", agent.Name,
		"old_status", agent.Status,
		"new_status", status,
	)

	return nil
}

// возвращает список онлайн агентов
func (s *AgentService) ListOnlineAgents(ctx context.Context) ([]*models.Agent, error) {
	s.logger.Debug("listing online agents")

	agents, err := s.agentStore.ListOnline(ctx)
	if err != nil {
		s.logger.Error("failed to list online agents from storage",
			"error", err,
		)
		return nil, fmt.Errorf("failed to list online agents: %w", err)
	}

	s.logger.Debug("online agents listed successfully",
		"count", len(agents),
	)

	return agents, nil
}

// возвращает агента по ID
func (s *AgentService) GetAgentByID(ctx context.Context, agentID string) (*models.Agent, error) {
	s.logger.Debug("getting agent by ID", "agent_id", agentID)

	if agentID == "" {
		s.logger.Warn("get agent failed: empty agent ID")
		return nil, fmt.Errorf("agent ID is required")
	}

	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent from storage",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Debug("agent not found", "agent_id", agentID)
		return nil, nil
	}

	s.logger.Debug("agent retrieved successfully",
		"agent_id", agentID,
		"name", agent.Name,
		"status", agent.Status,
	)

	return agent, nil
}

// возвращает статистику по агенту
func (s *AgentService) GetAgentStats(ctx context.Context, agentID string) (*models.AgentStats, error) {
	s.logger.Debug("getting agent statistics", "agent_id", agentID)

	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent for statistics",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("agent not found for statistics", "agent_id", agentID)
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Получаем последние результаты агента
	results, err := s.resultStore.GetByAgentID(ctx, agentID, 100) // последние 100 результатов
	if err != nil {
		s.logger.Error("failed to get agent results for statistics",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get agent results: %w", err)
	}

	stats := &models.AgentStats{
		Agent:         agent,
		TotalChecks:   len(results),
		LastActivity:  agent.LastHeartbeat,
		Uptime:        calculateUptime(agent.CreatedAt),
		SuccessRate:   0,
		RecentResults: results,
	}

	// Рассчитываем успешность
	if len(results) > 0 {
		successful := 0
		for _, result := range results {
			if result.Success {
				successful++
			}
		}
		stats.SuccessRate = float64(successful) / float64(len(results))
	}

	s.logger.Debug("agent statistics calculated",
		"agent_id", agentID,
		"total_checks", stats.TotalChecks,
		"success_rate", stats.SuccessRate,
		"uptime", stats.Uptime,
	)

	return stats, nil
}

// помечает неактивных агентов как офлайн
func (s *AgentService) CleanupInactiveAgents(ctx context.Context, timeout time.Duration) (int, error) {
	s.logger.Info("cleaning up inactive agents", "timeout", timeout)

	agents, err := s.agentStore.ListOnline(ctx)
	if err != nil {
		s.logger.Error("failed to list online agents for cleanup",
			"error", err,
		)
		return 0, fmt.Errorf("failed to list online agents: %w", err)
	}

	now := time.Now()
	inactiveCount := 0

	for _, agent := range agents {
		// Если агент не отправлял heartbeat дольше timeout
		if now.Sub(agent.LastHeartbeat) > timeout {
			if err := s.agentStore.UpdateStatus(ctx, agent.ID, models.AgentStatusOffline); err != nil {
				s.logger.Error("failed to mark agent as offline",
					"error", err,
					"agent_id", agent.ID,
					"name", agent.Name,
				)
				// Продолжаем с другими агентами
				continue
			}

			inactiveCount++
			s.logger.Info("marked inactive agent as offline",
				"agent_id", agent.ID,
				"name", agent.Name,
				"last_heartbeat", agent.LastHeartbeat,
				"inactive_duration", now.Sub(agent.LastHeartbeat),
			)
		}
	}

	s.logger.Info("inactive agents cleanup completed",
		"total_checked", len(agents),
		"marked_offline", inactiveCount,
	)

	return inactiveCount, nil
}

// обновляет возможности агента
func (s *AgentService) UpdateAgentCapabilities(ctx context.Context, agentID string, capabilities []string) error {
	s.logger.Info("updating agent capabilities",
		"agent_id", agentID,
		"capabilities", capabilities,
	)

	if agentID == "" {
		s.logger.Warn("capabilities update failed: empty agent ID")
		return fmt.Errorf("agent ID is required")
	}

	// Валидация capabilities
	for _, capability := range capabilities {
		if !validator.ValidateCheckType(capability) {
			s.logger.Warn("capabilities update failed: invalid capability",
				"agent_id", agentID,
				"invalid_capability", capability,
			)
			return fmt.Errorf("invalid capability: %s", capability)
		}
	}

	// Проверяем существование агента
	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent for capabilities update",
			"error", err,
			"agent_id", agentID,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("capabilities update failed: agent not found",
			"agent_id", agentID,
		)
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Обновляем capabilities
	if err := s.agentStore.UpdateCapabilities(ctx, agentID, capabilities); err != nil {
		s.logger.Error("failed to update agent capabilities in storage",
			"error", err,
			"agent_id", agentID,
			"capabilities", capabilities,
		)
		return fmt.Errorf("failed to update agent capabilities: %w", err)
	}

	s.logger.Info("agent capabilities updated successfully",
		"agent_id", agentID,
		"name", agent.Name,
		"old_capabilities", agent.Capabilities,
		"new_capabilities", capabilities,
	)

	return nil
}

// рассчитывает время работы агента
func calculateUptime(createdAt time.Time) time.Duration {
	return time.Since(createdAt)
}
