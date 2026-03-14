package models

import "time"

type PromptEvent struct {
	ID         string
	Timestamp  time.Time
	Agent      string
	SessionID  string
	Prompt     string
	Project    string
	GitBranch  string
	GitRemote  string
	WorkingDir string
	RawSource  string
	CreatedAt  time.Time
}
