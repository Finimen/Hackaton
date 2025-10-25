package client

import (
	domain "NetScan/internal/agent/domain"
	"context"
)

type APIClient struct {
}

// From Backend to Agent
func (a *APIClient) FetchTask(ctx context.Context) (*domain.Task, error) {

	return nil, nil
}

// From Agent to Backend
func (a *APIClient) SumbitResult(ctx context.Context, result *domain.Result) error {
	return nil
}
