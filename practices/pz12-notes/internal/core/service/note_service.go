package service

import (
	"errors"
	"strings"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core"
)

var ErrInvalid = errors.New("invalid note")

type Repository interface {
	Create(core.Note) (core.Note, error)
	List() ([]core.Note, error)
	Get(int64) (core.Note, error)
	Update(int64, core.NoteUpdate, time.Time) (core.Note, error)
	Delete(int64) error
}

type NoteService struct{ repo Repository }

func NewNoteService(r Repository) *NoteService { return &NoteService{repo: r} }

func (s *NoteService) Create(title, content string) (core.Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return core.Note{}, ErrInvalid
	}
	return s.repo.Create(core.Note{Title: title, Content: content, CreatedAt: time.Now().UTC()})
}

func (s *NoteService) List() ([]core.Note, error)      { return s.repo.List() }
func (s *NoteService) Get(id int64) (core.Note, error) { return s.repo.Get(id) }

func (s *NoteService) Update(id int64, patch core.NoteUpdate) (core.Note, error) {
	if patch.Title == nil && patch.Content == nil {
		return core.Note{}, ErrInvalid
	}
	if patch.Title != nil {
		trimmed := strings.TrimSpace(*patch.Title)
		if trimmed == "" {
			return core.Note{}, ErrInvalid
		}
		patch.Title = &trimmed
	}
	now := time.Now().UTC()
	return s.repo.Update(id, patch, now)
}

func (s *NoteService) Delete(id int64) error { return s.repo.Delete(id) }
