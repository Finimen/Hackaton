package client

import (
	"errors"
)

var ErrNoTasks = errors.New("no tasks available")
var ErrNotRegistered = errors.New("agent not registered")
var ErrBackendDown = errors.New("backend unavailable")
