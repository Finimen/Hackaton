package runner

import (
	"strconv"
	"time"
)

func getStringOption(options map[string]interface{}, key, defaultValue string) string {
	if value, ok := options[key].(string); ok && value != "" {
		return value
	}
	return defaultValue
}

func getIntOption(options map[string]interface{}, key string, defaultValue int) int {
	if value, ok := options[key].(float64); ok {
		return int(value)
	}
	if value, ok := options[key].(int); ok {
		return value
	}
	return defaultValue
}

func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := options[key].(bool); ok {
		return value
	}
	return defaultValue
}

func getDurationOption(options map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if value, ok := options[key].(float64); ok {
		return time.Duration(value) * time.Second
	}
	if value, ok := options[key].(string); ok {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getHeadersOption(options map[string]interface{}) map[string]string {
	headers := make(map[string]string)

	if headersOpt, ok := options["headers"].(map[string]interface{}); ok {
		for key, value := range headersOpt {
			if strValue, ok := value.(string); ok {
				headers[key] = strValue
			}
		}
	}

	return headers
}

func getFloatOption(options map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := options[key].(float64); ok {
		return value
	}
	if value, ok := options[key].(int); ok {
		return float64(value)
	}
	return defaultValue
}

func getStringSliceOption(options map[string]interface{}, key string) []string {
	var result []string

	if slice, ok := options[key].([]interface{}); ok {
		for _, item := range slice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}

	return result
}

func parsePort(port interface{}) (int, bool) {
	switch v := port.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case string:
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 65535 {
			return p, true
		}
	}
	return 0, false
}
