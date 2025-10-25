package models

import "time"

type CheckType string

const (
	CheckTypeHTTP       CheckType = "http"
	CheckTypeHTTPS      CheckType = "https"
	CheckTypePing       CheckType = "ping"
	CheckTypeTCP        CheckType = "tcp"
	CheckTypeDNS        CheckType = "dns"
	CheckTypeTraceroute CheckType = "traceroute"
)

type CheckStatus string

const (
	CheckStatusPending   CheckStatus = "pending"
	CheckStatusRunning   CheckStatus = "running"
	CheckStatusCompleted CheckStatus = "completed"
	CheckStatusFailed    CheckStatus = "failed"
)

type Check struct {
	ID        string      `json:"id"`
	Type      CheckType   `json:"type"`
	Target    string      `json:"target"`
	Status    CheckStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type CheckResult struct {
	ID        string                 `json:"id"`
	CheckID   string                 `json:"check_id"`
	AgentID   string                 `json:"agent_id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
	Duration  float64                `json:"duration"` // в секундах
	CreatedAt time.Time              `json:"created_at"`
}
