package domain

import (
	"NetScan/pkg/uuidutil"
	"time"
)

type CheckType string

const (
	HTTPCheck       CheckType = "http"
	HTTPSCheck      CheckType = "https"
	PingCheck       CheckType = "ping"
	DNSCheck        CheckType = "dns"
	TCPCheck        CheckType = "tcp"
	TracerouteCheck CheckType = "traceroute"
)

type DNSType string

const (
	DNSRecordA     DNSType = "A"
	DNSRecordAAAA  DNSType = "AAAA"
	DNSRecordMX    DNSType = "MX"
	DNSRecordNS    DNSType = "NS"
	DNSRecordTXT   DNSType = "TXT"
	DNSRecordCNAME DNSType = "CNAME"
)

type Task struct {
	ID        string                 `json:"id"`
	Type      CheckType              `json:"type"`
	Target    string                 `json:"target"`
	Options   map[string]interface{} `json:"options"`
	CreatedAt time.Time              `json:"created_at"`
	AgentID   string                 `json:"agent_id"`
}

func NewHTTPTask(target string, timeout int) *Task {
	return &Task{
		ID:     generateUUID(),
		Type:   HTTPCheck,
		Target: target,
		Options: map[string]interface{}{
			"timeout": timeout,
			"method":  "GET",
		},
		CreatedAt: time.Now(),
	}
}

func NewPingTask(target string, count int) *Task {
	return &Task{
		ID:     generateUUID(),
		Type:   PingCheck,
		Target: target,
		Options: map[string]interface{}{
			"count": count,
		},
		CreatedAt: time.Now(),
	}
}

func NewDNSTask(target string, recordType DNSType) *Task {
	return &Task{
		ID:     generateUUID(),
		Type:   DNSCheck,
		Target: target,
		Options: map[string]interface{}{
			"type": recordType,
		},
		CreatedAt: time.Now(),
	}
}

func NewTCPCheck(target string, port int) *Task {
	return &Task{
		ID:     generateUUID(),
		Type:   TCPCheck,
		Target: target,
		Options: map[string]interface{}{
			"port": port,
		},
		CreatedAt: time.Now(),
	}
}

func generateUUID() string {
	return uuidutil.New()
}
