package models

import "time"

type Project struct {
	Name        string
	DisplayName string
	GitRemotes  []string
	Paths       []string
	CreatedAt   time.Time
}
