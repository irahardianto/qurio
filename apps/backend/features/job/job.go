package job

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID        string          `json:"id"`
	SourceID  string          `json:"source_id"`
	Handler   string          `json:"handler"`
	Payload   json.RawMessage `json:"payload"`
	Error     string          `json:"error"`
	Retries   int             `json:"retries"`
	CreatedAt time.Time       `json:"created_at"`
}
