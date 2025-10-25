package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"NetScan/internal/backend/models"
	"NetScan/internal/backend/storage"
	"NetScan/pkg/validator"
)

type CheckService struct {
	checkStore  storage.CheckStore
	agentStore  storage.AgentStore
	resultStore storage.ResultStore
	queue       storage.Queue
	timeout     time.Duration
	logger      *slog.Logger
}

type CheckServiceConfig struct {
	TaskTimeout time.Duration
}

func NewCheckService(
	checkStore storage.CheckStore,
	agentStore storage.AgentStore,
	resultStore storage.ResultStore,
	queue storage.Queue,
	cfg CheckServiceConfig,
	logger *slog.Logger,
) *CheckService {

	timeout := cfg.TaskTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &CheckService{
		checkStore:  checkStore,
		agentStore:  agentStore,
		resultStore: resultStore,
		queue:       queue,
		timeout:     timeout,
		logger:      logger,
	}
}

// CreateCheck создает новую проверку и добавляет в очередь
func (s *CheckService) CreateCheck(ctx context.Context, checkType models.CheckType, target string) (*models.Check, error) {
	s.logger.Info("creating new check",
		"type", checkType,
		"target", target,
	)

	// Валидация входных данных
	if !validator.ValidateCheckType(string(checkType)) {
		s.logger.Warn("invalid check type received",
			"type", checkType,
			"target", target,
		)
		return nil, fmt.Errorf("invalid check type: %s", checkType)
	}

	if !validator.ValidateTarget(target) {
		s.logger.Warn("invalid target received",
			"type", checkType,
			"target", target,
		)
		return nil, fmt.Errorf("invalid target: %s", target)
	}

	// Создаем проверку
	check := &models.Check{
		Type:   checkType,
		Target: target,
		Status: models.CheckStatusPending,
	}

	if err := s.checkStore.Create(ctx, check); err != nil {
		s.logger.Error("failed to create check in storage",
			"error", err,
			"type", checkType,
			"target", target,
		)
		return nil, fmt.Errorf("failed to create check: %w", err)
	}

	s.logger.Debug("check created in storage",
		"check_id", check.ID,
		"type", checkType,
		"target", target,
	)

	// Получаем онлайн агентов для этой проверки
	agents, err := s.agentStore.ListOnline(ctx)
	if err != nil {
		s.logger.Error("failed to get online agents",
			"error", err,
			"check_id", check.ID,
		)
		return nil, fmt.Errorf("failed to get online agents: %w", err)
	}

	if len(agents) == 0 {
		s.logger.Warn("no online agents available for check",
			"check_id", check.ID,
		)
		return nil, fmt.Errorf("no online agents available")
	}

	s.logger.Debug("found online agents for check",
		"check_id", check.ID,
		"agents_count", len(agents),
	)

	// Создаем задачи для каждого онлайн агента
	task := models.CheckTask{
		CheckID:   check.ID,
		Type:      checkType,
		Target:    target,
		CreatedAt: time.Now(),
	}

	taskData, err := json.Marshal(task)
	if err != nil {
		s.logger.Error("failed to marshal check task",
			"error", err,
			"check_id", check.ID,
		)
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	s.logger.Debug("Task data before pushing to queue",
		"check_id", check.ID,
		"task_data", string(taskData),
		"data_type", fmt.Sprintf("%T", taskData),
	)

	// Добавляем задачу в очередь (каждый агент будет брать свою копию)
	successfulPushes := 0
	for i := range agents {
		if err := s.queue.PushTask(ctx, "check_tasks", taskData); err != nil {
			s.logger.Error("failed to push task to queue",
				"error", err,
				"check_id", check.ID,
				"agent_index", i,
			)
			// Продолжаем для других агентов, даже если один фейлится
		} else {
			successfulPushes++
		}
	}

	if successfulPushes == 0 {
		s.logger.Error("failed to push task to any agent queue",
			"check_id", check.ID,
			"total_agents", len(agents),
		)
		return nil, fmt.Errorf("failed to distribute task to any agent")
	}

	s.logger.Info("check created and queued successfully",
		"check_id", check.ID,
		"type", checkType,
		"target", target,
		"total_agents", len(agents),
		"successful_queues", successfulPushes,
	)

	return check, nil
}

// GetCheckByID возвращает проверку по ID
func (s *CheckService) GetCheckByID(ctx context.Context, id string) (*models.CheckWithResults, error) {
	s.logger.Debug("getting check by ID", "check_id", id)

	check, err := s.checkStore.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get check from storage",
			"error", err,
			"check_id", id,
		)
		return nil, fmt.Errorf("failed to get check: %w", err)
	}

	if check == nil {
		s.logger.Debug("check not found", "check_id", id)
		return nil, nil
	}

	// Получаем результаты проверки
	results, err := s.resultStore.GetByCheckID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get check results",
			"error", err,
			"check_id", id,
		)
		return nil, fmt.Errorf("failed to get check results: %w", err)
	}

	s.logger.Debug("check retrieved successfully",
		"check_id", id,
		"results_count", len(results),
		"status", check.Status,
	)

	return &models.CheckWithResults{
		Check:   check,
		Results: results,
	}, nil
}

// UpdateCheckStatus обновляет статус проверки
func (s *CheckService) UpdateCheckStatus(ctx context.Context, id string, status models.CheckStatus) error {
	s.logger.Debug("updating check status",
		"check_id", id,
		"new_status", status,
	)

	check, err := s.checkStore.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get check for status update",
			"error", err,
			"check_id", id,
		)
		return fmt.Errorf("failed to get check: %w", err)
	}

	if check == nil {
		s.logger.Warn("check not found for status update", "check_id", id)
		return fmt.Errorf("check not found: %s", id)
	}

	// Валидация перехода статусов
	if !s.isValidStatusTransition(check.Status, status) {
		s.logger.Warn("invalid status transition attempted",
			"check_id", id,
			"from_status", check.Status,
			"to_status", status,
		)
		return fmt.Errorf("invalid status transition: %s -> %s", check.Status, status)
	}

	if err := s.checkStore.UpdateStatus(ctx, id, status); err != nil {
		s.logger.Error("failed to update check status in storage",
			"error", err,
			"check_id", id,
			"status", status,
		)
		return fmt.Errorf("failed to update check status: %w", err)
	}

	s.logger.Info("check status updated",
		"check_id", id,
		"old_status", check.Status,
		"new_status", status,
	)

	return nil
}

// SubmitResult сохраняет результат проверки от агента
func (s *CheckService) SubmitResult(ctx context.Context, result *models.CheckResult) error {
	s.logger.Info("submitting check result",
		"check_id", result.CheckID,
		"agent_id", result.AgentID,
		"success", result.Success,
		"duration", result.Duration,
	)

	// Валидация
	if result.CheckID == "" {
		s.logger.Warn("empty check ID in result submission")
		return fmt.Errorf("check ID is required")
	}

	if result.AgentID == "" {
		s.logger.Warn("empty agent ID in result submission", "check_id", result.CheckID)
		return fmt.Errorf("agent ID is required")
	}

	// Проверяем существование проверки
	check, err := s.checkStore.GetByID(ctx, result.CheckID)
	if err != nil {
		s.logger.Error("failed to get check for result submission",
			"error", err,
			"check_id", result.CheckID,
		)
		return fmt.Errorf("failed to get check: %w", err)
	}

	if check == nil {
		s.logger.Warn("check not found for result submission", "check_id", result.CheckID)
		return fmt.Errorf("check not found: %s", result.CheckID)
	}

	// Проверяем существование агента
	agent, err := s.agentStore.GetByID(ctx, result.AgentID)
	if err != nil {
		s.logger.Error("failed to get agent for result submission",
			"error", err,
			"agent_id", result.AgentID,
			"check_id", result.CheckID,
		)
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("agent not found for result submission",
			"agent_id", result.AgentID,
			"check_id", result.CheckID,
		)
		return fmt.Errorf("agent not found: %s", result.AgentID)
	}

	// Сохраняем результат
	if err := s.resultStore.Create(ctx, result); err != nil {
		s.logger.Error("failed to save check result",
			"error", err,
			"check_id", result.CheckID,
			"agent_id", result.AgentID,
		)
		return fmt.Errorf("failed to create result: %w", err)
	}

	// Обновляем статус проверки если все агенты ответили
	if err := s.updateCheckCompletionStatus(ctx, result.CheckID); err != nil {
		s.logger.Warn("failed to update check completion status",
			"error", err,
			"check_id", result.CheckID,
		)
		// Не прерываем выполнение, это не критическая ошибка
	}

	s.logger.Info("check result submitted successfully",
		"check_id", result.CheckID,
		"agent_id", result.AgentID,
		"success", result.Success,
		"duration", result.Duration,
	)

	return nil
}

// ListChecks возвращает список проверок с пагинацией
func (s *CheckService) ListChecks(ctx context.Context, limit, offset int) ([]*models.Check, error) {
	s.logger.Debug("listing checks",
		"limit", limit,
		"offset", offset,
	)

	if limit <= 0 || limit > 100 {
		limit = 50 // default limit
		s.logger.Debug("adjusted limit to default", "new_limit", limit)
	}

	if offset < 0 {
		offset = 0
	}

	checks, err := s.checkStore.List(ctx, limit, offset)
	if err != nil {
		s.logger.Error("failed to list checks from storage",
			"error", err,
			"limit", limit,
			"offset", offset,
		)
		return nil, fmt.Errorf("failed to list checks: %w", err)
	}

	s.logger.Debug("checks listed successfully",
		"count", len(checks),
		"limit", limit,
		"offset", offset,
	)

	return checks, nil
}

// updateCheckCompletionStatus обновляет статус проверки когда все агенты ответили
func (s *CheckService) updateCheckCompletionStatus(ctx context.Context, checkID string) error {
	// Получаем все результаты для проверки
	results, err := s.resultStore.GetByCheckID(ctx, checkID)
	if err != nil {
		s.logger.Error("failed to get check results for completion check",
			"error", err,
			"check_id", checkID,
		)
		return err
	}

	// Получаем всех онлайн агентов
	agents, err := s.agentStore.ListOnline(ctx)
	if err != nil {
		s.logger.Error("failed to get online agents for completion check",
			"error", err,
			"check_id", checkID,
		)
		return err
	}

	// Если все онлайн агенты ответили, помечаем проверку как завершенную
	if len(results) >= len(agents) {
		if err := s.checkStore.UpdateStatus(ctx, checkID, models.CheckStatusCompleted); err != nil {
			s.logger.Error("failed to mark check as completed",
				"error", err,
				"check_id", checkID,
			)
			return err
		}

		s.logger.Info("check completed by all agents",
			"check_id", checkID,
			"results_count", len(results),
			"agents_count", len(agents),
		)
	} else {
		// Иначе помечаем как выполняющуюся
		if err := s.checkStore.UpdateStatus(ctx, checkID, models.CheckStatusRunning); err != nil {
			s.logger.Error("failed to mark check as running",
				"error", err,
				"check_id", checkID,
			)
			return err
		}

		s.logger.Debug("check still in progress",
			"check_id", checkID,
			"completed_results", len(results),
			"total_agents", len(agents),
		)
	}

	return nil
}

// isValidStatusTransition проверяет валидность перехода статусов
func (s *CheckService) isValidStatusTransition(from, to models.CheckStatus) bool {
	transitions := map[models.CheckStatus][]models.CheckStatus{
		models.CheckStatusPending:   {models.CheckStatusRunning, models.CheckStatusFailed},
		models.CheckStatusRunning:   {models.CheckStatusCompleted, models.CheckStatusFailed},
		models.CheckStatusCompleted: {},
		models.CheckStatusFailed:    {},
	}

	allowed, exists := transitions[from]
	if !exists {
		s.logger.Warn("unknown source status for transition validation",
			"from_status", from,
			"to_status", to,
		)
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}

	s.logger.Debug("invalid status transition",
		"from_status", from,
		"to_status", to,
		"allowed_transitions", allowed,
	)

	return false
}

// GetCheckStats возвращает статистику по проверке
func (s *CheckService) GetCheckStats(ctx context.Context, checkID string) (*models.CheckStats, error) {
	s.logger.Debug("getting check statistics", "check_id", checkID)

	results, err := s.resultStore.GetByCheckID(ctx, checkID)
	if err != nil {
		s.logger.Error("failed to get results for statistics",
			"error", err,
			"check_id", checkID,
		)
		return nil, err
	}

	stats := &models.CheckStats{
		TotalResults: len(results),
		Successful:   0,
		Failed:       0,
		AverageTime:  0,
		AgentResults: make(map[string]int),
	}

	var totalTime float64
	for _, result := range results {
		if result.Success {
			stats.Successful++
		} else {
			stats.Failed++
		}
		totalTime += result.Duration
		stats.AgentResults[result.AgentID]++
	}

	if len(results) > 0 {
		stats.AverageTime = totalTime / float64(len(results))
	}

	s.logger.Debug("check statistics calculated",
		"check_id", checkID,
		"total_results", stats.TotalResults,
		"successful", stats.Successful,
		"failed", stats.Failed,
		"average_time", stats.AverageTime,
	)

	return stats, nil
}
