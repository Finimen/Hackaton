package handler

import (
	client "NetScan/internal/agent/clients"
	clients "NetScan/internal/agent/clients"
	"NetScan/internal/agent/domain"
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
)

type AgentHandler struct {
	api    *clients.APIClient
	runner *TaskHandler
	logger *slog.Logger
}

const (
	INITIAL_DELAY     = 5 * time.Second
	MAX_INITIAL_DELAY = 30 * time.Second
	IDLE_DELAY        = time.Second
	ERROR_DELAY       = time.Second * 5
	HEALTH_CHECK_URL  = "/health"
)

func NewAgentHandler(logger *slog.Logger, clients *clients.APIClient, runner *TaskHandler) *AgentHandler {
	return &AgentHandler{
		api:    clients,
		runner: runner,
		logger: logger,
	}
}

func (s *AgentHandler) Run(ctx context.Context) {
	if err := s.waitForBackendStability(ctx); err != nil {
		s.logger.Error("Backend not available, stopping agent", "error", err)
		return
	}

	s.logger.Info("Backend is stable, starting task processing")
	s.processTasks(ctx)
}

func (s *AgentHandler) waitForBackendStability(ctx context.Context) error {
	s.logger.Info("Waiting for backend to stabilize...")

	timeout := time.After(2 * time.Minute)
	backoff := INITIAL_DELAY

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("backend not ready within timeout")
		default:
			if s.isBackendHealthy(ctx) {
				s.logger.Info("Backend health check passed")
				return nil
			}

			s.logger.Warn("Backend not ready yet, retrying...",
				"next_attempt", backoff,
			)

			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > MAX_INITIAL_DELAY {
				backoff = MAX_INITIAL_DELAY
			}
		}
	}
}

func (s *AgentHandler) isBackendHealthy(ctx context.Context) bool {
	_, err := s.api.FetchTask(ctx)
	if err == nil {
		return true
	}

	if errors.Is(err, client.ErrNoTasks) {
		return true
	}

	// Добавить проверку на 401
	if strings.Contains(err.Error(), "status 401") {
		return false
	}

	if s.isServerError(err) {
		return false
	}

	if errors.Is(err, client.ErrNoTasks) {
		return true
	}

	if s.isServerError(err) {
		return false
	}

	return false
}

func (s *AgentHandler) isServerError(err error) bool {
	errorStr := err.Error()
	return contains(errorStr, "status 500") ||
		contains(errorStr, "internal server error") ||
		contains(errorStr, "service unavailable")
}

func (s *AgentHandler) processTasks(ctx context.Context) {
	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping agent handler due to context cancellation")
			return
		default:
			task, err := s.api.FetchTask(ctx)
			if err != nil {
				s.handleFetchError(err, &consecutiveErrors, maxConsecutiveErrors)
				continue
			}

			consecutiveErrors = 0

			s.processSingleTask(ctx, task)
		}
	}
}

func (s *AgentHandler) handleFetchError(err error, consecutiveErrors *int, max int) {
	*consecutiveErrors++

	if errors.Is(err, client.ErrNoTasks) {
		s.logger.Debug("No tasks available", "delay", IDLE_DELAY)
		time.Sleep(IDLE_DELAY)
		return
	}

	s.logger.Error("Failed to fetch task",
		"error", err,
		"consecutive_errors", *consecutiveErrors,
	)

	delay := ERROR_DELAY
	if *consecutiveErrors > 3 {
		delay = time.Duration(*consecutiveErrors) * ERROR_DELAY
		s.logger.Warn("Multiple consecutive errors, increasing delay",
			"delay", delay,
		)
	}

	time.Sleep(delay)

	if *consecutiveErrors >= max {
		s.logger.Error("Too many consecutive errors, agent might be unstable")
	}
}

func (s *AgentHandler) processSingleTask(ctx context.Context, task *domain.Task) {
	s.logger.Info("Executing task",
		"task_id", task.ID,
		"type", task.Type,
		"target", task.Target,
	)

	result := s.runner.ExecuteTask(ctx, task)

	if err := s.api.SubmitResult(ctx, result); err != nil {
		s.logger.Error("Failed to submit result",
			"error", err,
			"task_id", task.ID,
		)
	} else {
		s.logger.Info("Result submitted successfully",
			"task_id", task.ID,
			"success", result.Success,
			"duration_ms", result.ResponseTime,
		)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 &&
			(s[0:len(substr)] == substr ||
				contains(s[1:], substr)))
}
