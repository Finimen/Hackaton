package models

import "time"

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

type Agent struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Token         string      `json:"-"`
	Location      string      `json:"location"`
	Status        AgentStatus `json:"status"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	CreatedAt     time.Time   `json:"created_at"`
	Capabilities  []string    `json:"capabilities"`
}

type HeartbeatRequest struct {
	AgentID string `json:"agent_id"`
	Load    int    `json:"load"` // текущая нагрузка 0-100
}

type RegisterRequest struct {
	Name         string   `json:"name"`
	Location     string   `json:"location"`
	Capabilities []string `json:"capabilities"`
}
