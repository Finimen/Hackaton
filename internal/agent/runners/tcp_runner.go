package runner

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

type TCPRunner struct {
	timeout time.Duration
}

func NewTCPRunner() *TCPRunner {
	fmt.Printf("🔧 DEBUG: Creating TCPRunner")
	return &TCPRunner{
		timeout: 10 * time.Second,
	}
}

func (r *TCPRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	port := getTCPPort(options, target)

	if port == 0 {
		port = getDefaultPort(target)
	}

	host := extractHost(target)
	address := net.JoinHostPort(host, strconv.Itoa(port))

	timeout := getDurationOption(options, "timeout", r.timeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	connectTime := time.Since(start)

	result := map[string]interface{}{
		"target":       target,
		"host":         host,
		"port":         port,
		"address":      address,
		"connect_time": connectTime.Milliseconds(),
	}

	if err != nil {
		result["port_open"] = false
		result["error"] = err.Error()

		if netErr, ok := err.(net.Error); ok {
			result["timeout"] = netErr.Timeout()
			result["temporary"] = netErr.Temporary()
		}

		return result, nil
	}
	defer conn.Close()

	result["port_open"] = true
	result["local_address"] = conn.LocalAddr().String()
	result["remote_address"] = conn.RemoteAddr().String()

	if getBoolOption(options, "banner_grab", false) {
		banner, bannerErr := r.grabBanner(ctx, conn, timeout)
		if bannerErr == nil && banner != "" {
			result["banner"] = banner
			result["banner_grabbed"] = true
		} else {
			result["banner_grabbed"] = false
			if bannerErr != nil {
				result["banner_error"] = bannerErr.Error()
			}
		}
	}

	return result, nil
}

func (r *TCPRunner) grabBanner(ctx context.Context, conn net.Conn, timeout time.Duration) (string, error) {
	readTimeout := getDurationOption(nil, "", 2*time.Second)
	conn.SetReadDeadline(time.Now().Add(readTimeout))

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	return string(buffer[:n]), nil
}

func getTCPPort(options map[string]interface{}, target string) int {
	if port, ok := options["port"].(float64); ok {
		return int(port)
	}

	if _, portStr, err := net.SplitHostPort(target); err == nil {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}

	return 0
}

func extractHost(target string) string {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		return target
	}
	return host
}

func getDefaultPort(target string) int {
	switch {
	case len(target) >= 5 && target[:5] == "https":
		return 443
	case len(target) >= 4 && target[:4] == "http":
		return 80
	case len(target) >= 3 && target[:3] == "ftp":
		return 21
	case len(target) >= 3 && target[:3] == "ssh":
		return 22
	case len(target) >= 4 && target[:4] == "smtp":
		return 25
	case len(target) >= 4 && target[:4] == "pop3":
		return 110
	case len(target) >= 4 && target[:4] == "imap":
		return 143
	case len(target) >= 4 && target[:4] == "mysql":
		return 3306
	case len(target) >= 8 && target[:8] == "postgres":
		return 5432
	case len(target) >= 4 && target[:4] == "redis":
		return 6379
	default:
		return 80
	}
}
