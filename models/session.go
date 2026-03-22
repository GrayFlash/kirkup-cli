package models

import "time"

type Session struct {
	ID                  string
	Project             string
	Agent               string
	StartedAt           time.Time
	EndedAt             time.Time
	PromptCount         int
	GapThresholdMinutes int
}
