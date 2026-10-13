package repo

import (
	"sort"
	"sync"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/core"
)

type NoteRepoMem struct {
	mu    sync.RWMutex
	notes map[int64]core.Note
	next  int64
}

func NewNoteRepoMem() *NoteRepoMem {
	return &NoteRepoMem{notes: make(map[int64]core.Note)}
}

func (r *NoteRepoMem) Create(n core.Note) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	n.ID = r.next
	r.notes[n.ID] = n
	return n, nil
}

func (r *NoteRepoMem) List() ([]core.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.Note, 0, len(r.notes))
	for _, n := range r.notes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *NoteRepoMem) Get(id int64) (core.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	return n, nil
}

func (r *NoteRepoMem) Update(id int64, patch core.Patch, updatedAt time.Time) (core.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.notes[id]
	if !ok {
		return core.Note{}, core.ErrNotFound
	}
	if patch.Title != nil {
		n.Title = *patch.Title
	}
	if patch.Content != nil {
		n.Content = *patch.Content
	}
	n.UpdatedAt = &updatedAt
	r.notes[id] = n
	return n, nil
}

func (r *NoteRepoMem) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(r.notes, id)
	return nil
}
