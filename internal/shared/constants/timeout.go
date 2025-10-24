package constants

import "time"

const (
	HTTPTimeout       = 30 * time.Second
	PingTimeout       = 30 * time.Second
	DNSTimeout        = 5 * time.Second
	TCPTimeout        = 15 * time.Second
	QueuePollInterval = 5 * time.Millisecond
)
