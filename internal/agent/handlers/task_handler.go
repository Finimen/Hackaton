package handler

import (
	"NetScan/internal/agent/domain"
	runner "NetScan/internal/agent/runners"
	"context"
	"log/slog"
	"time"
)

type TaskHandler struct {
	runnerFactory *runner.Factory
	logger        *slog.Logger
}

func NewTaskHandler(runnerFactory *runner.Factory, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		runnerFactory: runnerFactory,
		logger:        logger,
	}
}

func (t *TaskHandler) ExecuteTask(ctx context.Context, task *domain.Task) *domain.Result {
	runner, err := t.runnerFactory.GetRunner(task.Type)
	if err != nil {
		return domain.NewErrorResult(task.ID, task.AgentID, err)
	}

	start := time.Now()

	data, err := runner.Execute(ctx, task.Target, task.Options)

	responseTime := time.Since(start).Microseconds()
	return domain.NewSuccessResult(task.ID, task.AgentID, int(responseTime), data)
}
