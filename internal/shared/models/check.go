package models

import "time"

type CheckType string

const (
	CheckTypeHTTP       CheckType = "http"
	CheckTypePingCheck  CheckType = "ping"
	CheckTypeDNS        CheckType = "dns"
	CheckTypeTCP        CheckType = "tcp"
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
}
