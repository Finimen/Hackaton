package models

import "time"

type AgentStats struct {
	Agent         *Agent         `json:"agent"`
	TotalChecks   int            `json:"total_checks"`
	SuccessRate   float64        `json:"success_rate"`
	LastActivity  time.Time      `json:"last_activity"`
	Uptime        time.Duration  `json:"uptime"`
	RecentResults []*CheckResult `json:"recent_results,omitempty"`
}
