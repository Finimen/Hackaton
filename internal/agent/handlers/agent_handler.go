package handler

import (
	client "NetScan/internal/agent/clients"
	clients "NetScan/internal/agent/clients"
	"context"
	"errors"
	"log/slog"
	"time"
)

type AgentHandler struct {
	api    *clients.APIClient
	runner *TaskHandler
	logger *slog.Logger
}

const (
	IDLE_DELAY  = time.Second
	ERROR_DELAY = time.Second * 5
)

func NewAgentHandler(logger *slog.Logger, clients *clients.APIClient, runner *TaskHandler) *AgentHandler {
	return &AgentHandler{
		api:    clients,
		runner: runner,
		logger: logger,
	}
}

func (s *AgentHandler) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping agent handler due to context cancellation")
			return
		default:
			task, err := s.api.FetchTask(ctx)
			if err != nil {
				if errors.Is(err, client.ErrNoTasks) {
					s.logger.Info("no tasks")
					time.Sleep(IDLE_DELAY)
					continue
				}

				s.logger.Error("Failed to fetch task", "error", err)
				time.Sleep(ERROR_DELAY)
				continue
			}

			result := s.runner.ExecuteTask(ctx, task)

			if err := s.api.SubmitResult(ctx, result); err != nil {
				s.logger.Error("Failed to sumbit", "error", err)
			}
		}
	}
}
