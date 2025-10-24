package shared

type AgentToBackend struct {
	Backend string `json:"backend"`
	Agent   string `json:"agent"`
}
