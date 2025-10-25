package main

import (
	handler "NetScan/internal/agent/handlers"
	"log/slog"
)

type Container struct {
	Logger  *slog.Logger
	Handler *handler.AgentHandler
}

func GetContainer() *Container {
	container := Container{}

	container.Logger = slog.New(slog.Default().Handler())
	container.Handler = handler.NewAgentHandler(container.Logger, nil, nil)

	return &container
}
