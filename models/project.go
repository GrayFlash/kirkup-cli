package models

import "time"

type Project struct {
	Name        string
	DisplayName string
	GitRemotes  []string
	Paths       []string
	CreatedAt   time.Time
}

type ProjectStat struct {
	Name     string
	Prompts  int
	LastSeen time.Time
}
