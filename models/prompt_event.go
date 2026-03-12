package models

import "time"

type PromptEvent struct {
	ID         string
	Timestamp  time.Time
	Agent      string
	SessionID  string
	Prompt     string
	WorkingDir string
}
