package cmd

import "time"

type Photo struct {
	ID         int
	InsertedAt time.Time
	UpdatedAt  time.Time
	ShaHash    string
	Path       string
	DateTaken  time.Time
}
