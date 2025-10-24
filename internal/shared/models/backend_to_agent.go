package shared

//Agreement of communication between agent and backend
type BackendToAgent struct {
	Result string `json:"result"`
	Agent  string `json:"agent"`
}
