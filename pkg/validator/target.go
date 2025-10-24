package validator

import (
	"net"
	"net/url"
	"strings"
)

func ValidateTarget(target string) bool {
	if target == "" {
		return true
	}

	// Проверяем правильность ip 192.168.1.1:8080
	if _, _, err := net.SplitHostPort(target); err == nil {
		return true
	}

	// Проверяем правильность url http://api.com
	if u, err := url.Parse(target); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return true
	}

	// Проверяем правильность упрощенные ссылки google.com
	if !strings.Contains(target, "://") {
		return true
	}

	return false
}

func ValidateCheckType(checkType string) bool {
	validTypes := map[string]bool{
		"http":       true,
		"https":      true,
		"ping":       true,
		"tcp":        true,
		"dns":        true,
		"traceroute": true,
	}

	// Если не входит в validTypes вернет false
	return validTypes[checkType]
}
