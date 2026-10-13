package service

import (
	"context"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/models"
	"strings"
)

type Repository interface {
	Create(context.Context, models.Create) (models.Note, error)
	Get(context.Context, int64) (models.Note, error)
	List(context.Context) ([]models.Note, error)
	Update(context.Context, int64, models.Patch) (models.Note, error)
	Delete(context.Context, int64) error
}
type Service struct{ Notes Repository }

func (s Service) Create(ctx context.Context, in models.Create) (models.Note, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return models.Note{}, models.ErrInvalid
	}
	return s.Notes.Create(ctx, in)
}
func (s Service) Get(ctx context.Context, id int64) (models.Note, error) { return s.Notes.Get(ctx, id) }
func (s Service) List(ctx context.Context) ([]models.Note, error)        { return s.Notes.List(ctx) }
func (s Service) Update(ctx context.Context, id int64, p models.Patch) (models.Note, error) {
	if p.Title == nil && p.Content == nil {
		return models.Note{}, models.ErrInvalid
	}
	if p.Title != nil {
		trimmed := strings.TrimSpace(*p.Title)
		if trimmed == "" {
			return models.Note{}, models.ErrInvalid
		}
		p.Title = &trimmed
	}
	return s.Notes.Update(ctx, id, p)
}
func (s Service) Delete(ctx context.Context, id int64) error { return s.Notes.Delete(ctx, id) }
