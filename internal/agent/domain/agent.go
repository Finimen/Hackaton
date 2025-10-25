package domain

import "time"

type AgentStatus string

const (
	AgentStatusRegistered AgentStatus = "registered"
	AgentStatusActive     AgentStatus = "active"
	AgentStatusOffline    AgentStatus = "offline"
	AgentStatusError      AgentStatus = "error"
)

type Agent struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Location     string        `json:"location"`
	Status       AgentStatus   `json:"status"`
	Token        string        `json:"-"`
	Version      string        `json:"version"`
	Metadata     AgentMetadata `json:"metadata"`
	Capabilities []CheckType   `json:"capabilities"`
	LastSeen     time.Time     `json:"last_seen"`
	CreatedAt    time.Time     `json:"created_at"`
}

type AgentMetadata struct {
	IPAddress string `json:"ip_address"`
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUCount  int    `json:"cpu_count"`
	MemoryMB  int64  `json:"memory_mb"`
	GoVersion string `json:"go_version"`
}

type Heartbeat struct {
	AgentID   string      `json:"agent_id"`
	Status    AgentStatus `json:"status"`
	Load      SystemLoad  `json:"load"`
	Timestamp time.Time   `json:"timestamp"`
}

type SystemLoad struct {
	CPUUsage    float64 `json:"cpu_usage"`    // 0.0 - 1.0
	MemoryUsage float64 `json:"memory_usage"` // 0.0 - 1.0
	DiskUsage   float64 `json:"disk_usage"`   // 0.0 - 1.0
	ActiveJobs  int     `json:"active_jobs"`
}

func NewAgent(name, location, token string) *Agent {
	return &Agent{
		ID:       generateUUID(),
		Name:     name,
		Location: location,
		Token:    token,
		Status:   AgentStatusRegistered,
		Version:  "1.0.0",
		Capabilities: []CheckType{
			HTTPCheck,
			PingCheck,
			DNSCheck,
			TCPCheck,
		},
		Metadata:  AgentMetadata{},
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
}

func (a *Agent) UpdateMetadata(metadata AgentMetadata) {
	a.Metadata = metadata
}

func (a *Agent) UpdateStatus(status AgentStatus) {
	a.Status = status
	a.LastSeen = time.Now()
}

func (a *Agent) IsCapable(checkType CheckType) bool {
	for _, capability := range a.Capabilities {
		if capability == checkType {
			return true
		}
	}
	return false
}

func (a *Agent) AddCapability(checkType CheckType) {
	if !a.IsCapable(checkType) {
		a.Capabilities = append(a.Capabilities, checkType)
	}
}
