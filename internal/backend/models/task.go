package models

import "time"

type CheckTask struct {
	CheckID   string                 `json:"check_id"`
	Type      CheckType              `json:"type"`
	Target    string                 `json:"target"`
	Options   map[string]interface{} `json:"options,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}
