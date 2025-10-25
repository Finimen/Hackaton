package runner

import "context"

type Runner interface {
	Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error)
}
