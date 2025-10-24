package client

import (
	runner "NetScan/internal/agent/runners"
	"context"
)

type APIClient struct {
}

func (a *APIClient) FetchTask(ctx context.Context) (*runner.Task, error) {
	return nil, nil
}

func (a *APIClient) SumbitResult(ctx context.Context, result *interface{}) error {
	return nil
}
