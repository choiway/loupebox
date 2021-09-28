package cmd

import "time"

type Photo struct {
	ID         int
	InsertedAt time.Time
	UpdatedAt  time.Time
	ShaHash    string
	SourcePath string
	Path       string
	DateTaken  time.Time
	Status     string
}
