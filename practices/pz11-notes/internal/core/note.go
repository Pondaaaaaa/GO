package core

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("note not found")

type Note struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type Patch struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
