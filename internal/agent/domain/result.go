package domain

import "time"

type Result struct {
	TaskID       string                 `json:"task_id"`
	AgentID      string                 `json:"agent_id"`
	Success      bool                   `json:"success"`
	ResponseTime int                    `json:"response_time"`
	Error        string                 `json:"error,omitempty"`
	Data         map[string]interface{} `json:"data"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type HTTPResult struct {
	StatusCode    int               `json:"status_code"`
	Headers       map[string]string `json:"headers"`
	BodyPreview   string            `json:"body_preview,omitempty"`
	ContentLength int64             `json:"content_length"`
	SSL           *SSLInfo          `json:"ssl,omitempty"`
}

type PingResult struct {
	PacketsSent     int     `json:"packets_sent"`
	PacketsReceived int     `json:"packets_recived"`
	PacketLoss      float64 `json:"packet_loss"`
	MinRTT          float64 `json:"min_rtt"`
	MaxRTT          float64 `json:"max_rtt"`
	AvgRTT          float64 `json:"avg_rtt"`
}

type DNSResult struct {
	Records []string `json:"records"`
	Server  string   `json:"server"`
	TTL     int      `json:"ttl,omitempty"`
}

type TCPResult struct {
	PortOpen     bool    `json:"port_open"`
	ConntentTime float64 `json:"conntent_time"`
}

type SSLInfo struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at"`
	Issuer    string    `json:"issuer"`
}

func NewSuccessResult(taskID, agentID string, responseTime int, data map[string]interface{}) *Result {
	return &Result{
		TaskID:       taskID,
		AgentID:      agentID,
		Success:      true,
		ResponseTime: responseTime,
		Data:         data,
		Timestamp:    time.Now(),
		Metadata:     map[string]interface{}{},
	}
}

func NewErrorResult(taskID, agentID string, error error) *Result {
	return &Result{
		TaskID:    taskID,
		AgentID:   agentID,
		Success:   false,
		Error:     error.Error(),
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{},
	}
}
