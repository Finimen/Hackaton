package runner

import (
	"context"
	"net/http"
)

type HTTPRunner struct {
	client *http.Client
}

func (r *HTTPRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	resp, err := r.client.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
	}, nil
}
