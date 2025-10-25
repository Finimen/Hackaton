package handlers

import (
	"log/slog"

	"NetScan/internal/backend/dependencies"
	"NetScan/internal/backend/services"
)

type Handlers struct {
	checkService *services.CheckService
	agentService *services.AgentService
	queueService *services.QueueService
	logger       *slog.Logger
}

func NewHandlers(container *dependencies.Container) *Handlers {
	return &Handlers{
		checkService: container.CheckService,
		agentService: container.AgentService,
		queueService: container.QueueService,
		logger:       slog.Default(),
	}
}
