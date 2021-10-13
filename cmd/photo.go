package cmd

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	ID         int
	InsertedAt time.Time
	UpdatedAt  time.Time
	RepoID     uuid.UUID
	ShaHash    string
	SourcePath string
	Path       string
	DateTaken  time.Time
	Status     string
}
