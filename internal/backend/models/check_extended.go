package models

type CheckWithResults struct {
	*Check
	Results []*CheckResult `json:"results"`
}

type CheckStats struct {
	TotalResults int            `json:"total_results"`
	Successful   int            `json:"successful"`
	Failed       int            `json:"failed"`
	AverageTime  float64        `json:"average_time"`
	AgentResults map[string]int `json:"agent_results"`
}
