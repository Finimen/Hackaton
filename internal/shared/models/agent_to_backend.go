package shared

//Agreement of communication between agent and backend
type AgentToBackend struct {
	Backend string `json:"backend"`
	Agent   string `json:"agent"`
}
