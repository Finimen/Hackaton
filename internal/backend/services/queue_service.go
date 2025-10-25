package services

import (
	"NetScan/internal/backend/models"
	"NetScan/internal/backend/storage"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type QueueService struct {
	queue           storage.Queue
	checkStore      storage.CheckStore
	agentStore      storage.AgentStore
	agentTasksStore storage.AgentTasksStore
	resultStore     storage.ResultStore
	logger          *slog.Logger
	timeout         time.Duration
}

type QueueServiceConfig struct {
	TaskTimeout      time.Duration
	StuckTaskTimeout time.Duration
	PollInterval     time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
}

func NewQueueService(
	queue storage.Queue,
	checkStore storage.CheckStore,
	agentStore storage.AgentStore,
	agentTasksStore storage.AgentTasksStore,
	resultStore storage.ResultStore,
	cfg QueueServiceConfig,
	logger *slog.Logger,
) *QueueService {

	timeout := cfg.TaskTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	stuckTaskTimeout := cfg.StuckTaskTimeout
	if stuckTaskTimeout == 0 {
		stuckTaskTimeout = 10 * time.Minute // 10 минут по умолчанию
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &QueueService{
		queue:           queue,
		checkStore:      checkStore,
		agentStore:      agentStore,
		agentTasksStore: agentTasksStore,
		resultStore:     resultStore,
		logger:          logger,
		timeout:         timeout,
	}
}

// возвращает следующую задачу для агента
func (s *QueueService) GetNextTask(ctx context.Context, agentID string) (*models.CheckTask, error) {
	s.logger.Debug("getting next task for agent", "agent_id", agentID)

	// Проверяем существование агента
	agent, err := s.agentStore.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Error("failed to get agent for task assignment",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		s.logger.Warn("agent not found for task assignment", "agent_id", agentID)
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	if agent.Status != models.AgentStatusOnline {
		s.logger.Warn("agent is not online",
			"agent_id", agentID,
			"status", agent.Status,
		)
		return nil, fmt.Errorf("agent is not online: %s", agentID)
	}

	taskData, err := s.queue.PopTask(ctx, "check_tasks", s.timeout)
	if err != nil {
		s.logger.Error("failed to pop task from queue",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get task from queue: %w", err)
	}

	if taskData == nil {
		s.logger.Debug("no tasks available in queue", "agent_id", agentID)
		return nil, nil
	}

	s.logger.Debug("Raw task data from Redis",
		"raw_data", string(taskData),
		"agent_id", agentID,
	)

	// ИСПРАВЛЕНИЕ: Декодируем base64 из JSON строки
	var taskJson []byte

	// Данные приходят как: "\"eyJjaGVja...\""
	rawString := string(taskData)

	// Проверяем если это JSON строка с кавычками
	if len(rawString) >= 2 && rawString[0] == '"' && rawString[len(rawString)-1] == '"' {
		// Убираем внешние кавычки
		base64String := rawString[1 : len(rawString)-1]

		s.logger.Debug("Decoding base64 string",
			"base64_string", base64String,
			"agent_id", agentID,
		)

		// Декодируем base64
		decoded, err := base64.StdEncoding.DecodeString(base64String)
		if err != nil {
			s.logger.Error("Failed to decode base64 task data",
				"error", err,
				"agent_id", agentID,
				"base64_string", base64String,
			)
			return nil, fmt.Errorf("failed to decode base64 task: %w", err)
		}

		taskJson = decoded
		s.logger.Info("Successfully decoded base64 task data",
			"agent_id", agentID,
			"decoded_data", string(taskJson),
		)
	} else {
		// Если не в кавычках, используем как есть
		taskJson = taskData
		s.logger.Debug("Using raw task data (not base64 encoded)",
			"agent_id", agentID,
		)
	}

	// Парсим задачу из JSON
	var task models.CheckTask
	if err := json.Unmarshal(taskJson, &task); err != nil {
		s.logger.Error("failed to unmarshal task data",
			"error", err,
			"agent_id", agentID,
			"task_data", string(taskJson),
		)
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	s.logger.Info("Task successfully parsed",
		"agent_id", agentID,
		"check_id", task.CheckID,
		"type", task.Type,
		"target", task.Target,
	)

	// Проверяем существование проверки
	check, err := s.checkStore.GetByID(ctx, task.CheckID)
	if err != nil {
		s.logger.Error("failed to get check for task",
			"error", err,
			"check_id", task.CheckID,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get check: %w", err)
	}

	if check == nil {
		s.logger.Warn("check not found for task",
			"check_id", task.CheckID,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("check not found: %s", task.CheckID)
	}

	// Проверяем что агент поддерживает тип проверки
	if !s.agentSupportsCheckType(agent, task.Type) {
		s.logger.Warn("agent does not support check type",
			"agent_id", agentID,
			"check_type", task.Type,
			"agent_capabilities", agent.Capabilities,
		)

		// Возвращаем задачу обратно в очередь
		if err := s.requeueTask(ctx, taskData); err != nil {
			s.logger.Error("failed to requeue unsupported task",
				"error", err,
				"check_id", task.CheckID,
				"agent_id", agentID,
			)
		}

		return nil, fmt.Errorf("agent does not support check type: %s", task.Type)
	}

	// Обновляем статус проверки на "выполняется"
	if check.Status == models.CheckStatusPending {
		if err := s.checkStore.UpdateStatus(ctx, task.CheckID, models.CheckStatusRunning); err != nil {
			s.logger.Warn("failed to update check status to running",
				"error", err,
				"check_id", task.CheckID,
				"agent_id", agentID,
			)
			// Не прерываем выполнение, это не критическая ошибка
		}
	}

	s.logger.Info("task assigned to agent",
		"agent_id", agentID,
		"agent_name", agent.Name,
		"check_id", task.CheckID,
		"check_type", task.Type,
		"target", task.Target,
	)

	return &task, nil
}

// SubmitTaskResult обрабатывает результат выполнения задачи
func (s *QueueService) SubmitTaskResult(ctx context.Context, result *models.CheckResult) error {
	s.logger.Info("submitting task result",
		"check_id", result.CheckID,
		"agent_id", result.AgentID,
		"success", result.Success,
		"duration", result.Duration,
	)

	// Валидация
	if result.CheckID == "" {
		s.logger.Warn("empty check ID in task result")
		return fmt.Errorf("check ID is required")
	}

	if result.AgentID == "" {
		s.logger.Warn("empty agent ID in task result", "check_id", result.CheckID)
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
		s.logger.Error("failed to save task result",
			"error", err,
			"check_id", result.CheckID,
			"agent_id", result.AgentID,
		)
		return fmt.Errorf("failed to create result: %w", err)
	}

	// Публикуем уведомление о результате (для WebSocket)
	if err := s.publishResultNotification(ctx, result); err != nil {
		s.logger.Warn("failed to publish result notification",
			"error", err,
			"check_id", result.CheckID,
		)
		// Не прерываем выполнение, это не критическая ошибка
	}

	s.logger.Info("task result submitted successfully",
		"check_id", result.CheckID,
		"agent_id", result.AgentID,
		"success", result.Success,
		"duration", result.Duration,
	)

	return nil
}

// возвращает статистику очереди
func (s *QueueService) GetQueueStats(ctx context.Context) (*models.QueueStats, error) {
	s.logger.Debug("getting comprehensive queue statistics")

	// Получаем длину очереди
	queueLength, err := s.queue.GetQueueLength(ctx, "check_tasks")
	if err != nil {
		s.logger.Error("failed to get queue length",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get queue length: %w", err)
	}

	// Получаем онлайн агентов
	agents, err := s.agentStore.ListOnline(ctx)
	if err != nil {
		s.logger.Error("failed to get online agents for queue stats",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get online agents: %w", err)
	}

	// Получаем количество проверок по статусам
	pendingChecks, err := s.checkStore.GetCountByStatus(ctx, models.CheckStatusPending)
	if err != nil {
		s.logger.Error("failed to get pending checks count",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get pending checks count: %w", err)
	}

	runningChecks, err := s.checkStore.GetCountByStatus(ctx, models.CheckStatusRunning)
	if err != nil {
		s.logger.Error("failed to get running checks count",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get running checks count: %w", err)
	}

	completedChecks, err := s.checkStore.GetCountByStatus(ctx, models.CheckStatusCompleted)
	if err != nil {
		s.logger.Error("failed to get completed checks count",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get completed checks count: %w", err)
	}

	// Получаем количество активных задач (взятых агентами)
	activeTasksCount, err := s.getActiveTasksCount(ctx)
	if err != nil {
		s.logger.Warn("failed to get active tasks count",
			"error", err,
		)
		// Не прерываем выполнение, используем 0 как fallback
		activeTasksCount = 0
	}

	stats := &models.QueueStats{
		QueueLength:     queueLength,
		OnlineAgents:    len(agents),
		ActiveTasks:     activeTasksCount,
		PendingChecks:   pendingChecks,
		RunningChecks:   runningChecks,
		CompletedChecks: completedChecks,
		QueueThroughput: s.calculateQueueThroughput(completedChecks),
		AvgWaitTime:     float64(s.calculateAverageWaitTime(pendingChecks, queueLength, len(agents))),
		Timestamp:       time.Now(),
	}

	s.logger.Debug("comprehensive queue statistics calculated",
		"queue_length", stats.QueueLength,
		"online_agents", stats.OnlineAgents,
		"active_tasks", stats.ActiveTasks,
		"pending_checks", stats.PendingChecks,
		"running_checks", stats.RunningChecks,
		"completed_checks", stats.CompletedChecks,
		"throughput", stats.QueueThroughput,
		"avg_wait_time", stats.AvgWaitTime,
	)

	return stats, nil
}

// очищает зависшие задачи
func (s *QueueService) CleanupStuckTasks(ctx context.Context, timeout time.Duration) (int, error) {
	s.logger.Info("cleaning up stuck tasks", "timeout", timeout)

	if timeout == 0 {
		timeout = 10 * time.Minute // default timeout
	}

	// Получаем список зависших задач
	stuckTasks, err := s.agentTasksStore.GetStuckTasks(ctx, timeout)
	if err != nil {
		s.logger.Error("failed to get stuck tasks",
			"error", err,
			"timeout", timeout,
		)
		return 0, fmt.Errorf("failed to get stuck tasks: %w", err)
	}

	if len(stuckTasks) == 0 {
		s.logger.Debug("no stuck tasks found", "timeout", timeout)
		return 0, nil
	}

	cleanedCount := 0
	for _, task := range stuckTasks {
		s.logger.Warn("found stuck task",
			"task_id", task.ID,
			"agent_id", task.AgentID,
			"check_id", task.CheckID,
			"taken_at", task.TakenAt,
			"stuck_duration", time.Since(task.TakenAt),
		)

		// Возвращаем задачу в очередь
		if err := s.requeueStuckTask(ctx, task); err != nil {
			s.logger.Error("failed to requeue stuck task",
				"error", err,
				"task_id", task.ID,
				"check_id", task.CheckID,
			)
			continue
		}

		// Удаляем запись о взятой задаче
		if err := s.agentTasksStore.DeleteTask(ctx, task.AgentID, task.CheckID); err != nil {
			s.logger.Error("failed to delete stuck task record",
				"error", err,
				"task_id", task.ID,
				"agent_id", task.AgentID,
			)
			continue
		}

		// Обновляем статус проверки если нужно
		if err := s.handleStuckCheck(ctx, task.CheckID); err != nil {
			s.logger.Warn("failed to handle stuck check status",
				"error", err,
				"check_id", task.CheckID,
			)
		}

		cleanedCount++
		s.logger.Info("stuck task cleaned up successfully",
			"task_id", task.ID,
			"agent_id", task.AgentID,
			"check_id", task.CheckID,
		)
	}

	s.logger.Info("stuck tasks cleanup completed",
		"total_found", len(stuckTasks),
		"cleaned_count", cleanedCount,
		"timeout", timeout,
	)

	return cleanedCount, nil
}

// публикует прогресс выполнения задачи
func (s *QueueService) PublishTaskProgress(ctx context.Context, progress *models.TaskProgress) error {
	s.logger.Debug("publishing task progress",
		"check_id", progress.CheckID,
		"agent_id", progress.AgentID,
		"stage", progress.Stage,
		"progress", progress.Progress,
	)

	progressData, err := json.Marshal(progress)
	if err != nil {
		s.logger.Error("failed to marshal task progress",
			"error", err,
			"check_id", progress.CheckID,
		)
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	// Публикуем прогресс в канал для WebSocket
	if err := s.queue.Publish(ctx, "task_progress", progressData); err != nil {
		s.logger.Error("failed to publish task progress",
			"error", err,
			"check_id", progress.CheckID,
		)
		return fmt.Errorf("failed to publish progress: %w", err)
	}

	s.logger.Debug("task progress published successfully",
		"check_id", progress.CheckID,
		"stage", progress.Stage,
		"progress", progress.Progress,
	)

	return nil
}

// проверяет поддержку типа проверки агентом
func (s *QueueService) agentSupportsCheckType(agent *models.Agent, checkType models.CheckType) bool {
	for _, capability := range agent.Capabilities {
		if string(checkType) == capability {
			return true
		}
	}
	return false
}

// возвращает задачу обратно в очередь
func (s *QueueService) requeueTask(ctx context.Context, taskData []byte) error {
	return s.queue.PushTask(ctx, "check_tasks", taskData)
}

// публикует уведомление о результате
func (s *QueueService) publishResultNotification(ctx context.Context, result *models.CheckResult) error {
	notification := models.ResultNotification{
		CheckID:   result.CheckID,
		AgentID:   result.AgentID,
		Success:   result.Success,
		Duration:  result.Duration,
		Timestamp: time.Now(),
	}

	notificationData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal result notification: %w", err)
	}

	return s.queue.Publish(ctx, "check_results", notificationData)
}

// обрабатывает статус проверки для зависшей задачи
func (s *QueueService) handleStuckCheck(ctx context.Context, checkID string) error {
	check, err := s.checkStore.GetByID(ctx, checkID)
	if err != nil {
		return err
	}

	if check == nil {
		return fmt.Errorf("check not found: %s", checkID)
	}

	// Если проверка в статусе running и зависла, помечаем как failed
	if check.Status == models.CheckStatusRunning {
		return s.checkStore.UpdateStatus(ctx, checkID, models.CheckStatusFailed)
	}

	return nil
}

// возвращает зависшую задачу обратно в очередь
func (s *QueueService) requeueStuckTask(ctx context.Context, task *models.AgentTask) error {
	taskData, err := json.Marshal(task.TaskData)
	if err != nil {
		return fmt.Errorf("failed to marshal stuck task data: %w", err)
	}

	return s.queue.PushTask(ctx, "check_tasks", taskData)
}

// возвращает количество активных задач
func (s *QueueService) getActiveTasksCount(ctx context.Context) (int, error) {
	// В реальной реализации здесь был бы запрос к agent_tasks
	// Пока возвращаем 0 как заглушку
	return 0, nil
}

// рассчитывает пропускную способность очереди
func (s *QueueService) calculateQueueThroughput(completedChecks int) float64 {
	// Базовая реализация - можно улучшить с учетом временных интервалов
	if completedChecks > 0 {
		return float64(completedChecks) / float64(60) // checks per minute (упрощенно)
	}
	return 0.0
}

// рассчитывает среднее время ожидания
func (s *QueueService) calculateAverageWaitTime(pendingChecks int, queueLength int64, onlineAgents int) time.Duration {
	if onlineAgents == 0 || pendingChecks == 0 {
		return 0
	}

	// Упрощенный расчет: (задачи в очереди + pending) / (агенты * производительность)
	estimatedTime := time.Duration(pendingChecks+int(queueLength)) * time.Second / time.Duration(onlineAgents*2)
	return estimatedTime
}
