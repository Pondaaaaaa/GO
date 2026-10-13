package core

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("note not found")

type Note struct {
	ID        int64      `json:"id" example:"1" binding:"required"`
	Title     string     `json:"title" example:"First note" binding:"required"`
	Content   string     `json:"content" example:"Test body" binding:"required"`
	CreatedAt time.Time  `json:"createdAt" example:"2026-10-13T12:00:00Z" binding:"required"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" example:"2026-10-13T12:01:00Z"`
}

type NoteCreate struct {
	// Title is trimmed and must contain a non-whitespace character.
	Title   string `json:"title" example:"First note" binding:"required" minLength:"1"`
	Content string `json:"content" example:"Test body"`
}

// NoteUpdate requires at least one non-null field. Empty content is allowed.
type NoteUpdate struct {
	Title   *string `json:"title" example:"Updated note" minLength:"1"`
	Content *string `json:"content" example:"Changed body"`
}

type ErrorResponse struct {
	// Error is invalid request, note not found, or internal server error.
	Error string `json:"error" binding:"required"`
}
