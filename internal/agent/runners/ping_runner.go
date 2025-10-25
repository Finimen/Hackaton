package runner

import (
	"context"
	"fmt"
	"net"
	"time"
)

type PingRunner struct {
	timeout time.Duration
}

func NewPingRunner() *PingRunner {
	return &PingRunner{
		timeout: 10 * time.Second,
	}
}

func (r *PingRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	count := getIntOption(options, "count", 4)
	timeout := getDurationOption(options, "timeout", r.timeout)

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		port = "80"
	}

	var rtts []time.Duration
	packetsSent := 0
	packetsReceived := 0

	for i := 0; i < count; i++ {
		start := time.Now()

		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err != nil {
			packetsSent++
			continue
		}
		conn.Close()

		rtt := time.Since(start)
		rtts = append(rtts, rtt)
		packetsSent++
		packetsReceived++

		if i < count-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if len(rtts) == 0 {
		return nil, fmt.Errorf("all ping attempts failed")
	}

	minRTT, maxRTT, avgRTT := calculateRTTStats(rtts)
	packetLoss := float64(packetsSent-packetsReceived) / float64(packetsSent) * 100

	return map[string]interface{}{
		"packets_sent":     packetsSent,
		"packets_received": packetsReceived,
		"packet_loss":      packetLoss,
		"min_rtt":          minRTT.Seconds() * 1000,
		"max_rtt":          maxRTT.Seconds() * 1000,
		"avg_rtt":          avgRTT.Seconds() * 1000,
		"target":           target,
		"rtts":             rtts,
	}, nil
}

func calculateRTTStats(rtts []time.Duration) (min, max, avg time.Duration) {
	if len(rtts) == 0 {
		return 0, 0, 0
	}

	min = rtts[0]
	max = rtts[0]
	total := time.Duration(0)

	for _, rtt := range rtts {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		total += rtt
	}

	avg = total / time.Duration(len(rtts))
	return min, max, avg
}
