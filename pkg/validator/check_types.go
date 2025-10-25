package validator

func ValidateCheckType(checkType string) bool {
	validTypes := map[string]bool{
		"http":       true,
		"https":      true,
		"ping":       true,
		"tcp":        true,
		"dns":        true,
		"traceroute": true,
	}
	return validTypes[checkType]
}
