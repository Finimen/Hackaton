package runner

import "context"

type PingRunner struct {
}

func (r *PingRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"packets_sent":    4,
		"packets_recived": 4,
		"avg_rtt":         45.2,
	}, nil
}
