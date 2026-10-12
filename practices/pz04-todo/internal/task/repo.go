package task

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("task not found")

type Repository struct {
	mu     sync.RWMutex
	tasks  map[int64]Task
	nextID int64
}

func NewRepository() *Repository {
	return &Repository{tasks: make(map[int64]Task), nextID: 1}
}

func (r *Repository) List() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]Task, 0, len(r.tasks))
	for _, item := range r.tasks {
		tasks = append(tasks, item)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}

func (r *Repository) Get(id int64) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return item, nil
}

func (r *Repository) Create(title string, done bool) Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	item := Task{ID: r.nextID, Title: title, Done: done, CreatedAt: now, UpdatedAt: now}
	r.tasks[item.ID] = item
	r.nextID++
	return item
}

func (r *Repository) Update(id int64, title string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	item.Title = title
	item.Done = done
	item.UpdatedAt = time.Now().UTC()
	r.tasks[id] = item
	return item, nil
}

func (r *Repository) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(r.tasks, id)
	return nil
}
