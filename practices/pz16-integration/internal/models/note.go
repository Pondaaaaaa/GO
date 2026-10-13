package models

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("note not found")
var ErrInvalid = errors.New("invalid request")

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Create struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Patch struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
