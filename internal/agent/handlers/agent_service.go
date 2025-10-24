package handler

import (
	client "NetScan/internal/agent/clients"
	clients "NetScan/internal/agent/clients"
	runner "NetScan/internal/agent/runners"
	"context"
	"errors"
	"log/slog"
	"time"
)

type AgentService struct {
	client *clients.APIClient
	runner *runner.TaskRunner
	logger *slog.Logger
}

const (
	IDLE_DELAY  = time.Second
	ERROR_DELAY = time.Second * 5
)

func (s *AgentService) Run() {
	for {
		task, err := s.client.FetchTask(context.Background())
		if err != nil {
			if errors.Is(err, client.ErrNoTasks) {
				s.logger.Info("no tasks")
				time.Sleep(IDLE_DELAY)
				continue
			}

			s.logger.Error("Failed", err)
			time.Sleep(ERROR_DELAY)
			continue
		}

		result := s.runner.Execute(task)

		if err := s.client.SumbitResult(context.Background(), result); err != nil {
			s.logger.Error("Failed to sumbit", "error", err)
		}
	}
}
