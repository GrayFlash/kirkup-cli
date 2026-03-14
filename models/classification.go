package models

import "time"

type Classification struct {
	ID            string
	PromptEventID string
	Category      string
	Confidence    float64
	Classifier    string
	CreatedAt     time.Time
}
