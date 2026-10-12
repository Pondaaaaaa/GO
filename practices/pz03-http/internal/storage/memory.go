package storage

import (
	"errors"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type MemoryStore struct {
	mu     sync.RWMutex
	nextID int64
	tasks  map[int64]Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[int64]Task)}
}

func (s *MemoryStore) Create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	task := Task{ID: s.nextID, Title: title}
	s.tasks[task.ID] = task
	return task
}

func (s *MemoryStore) Get(id int64) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return task, nil
}

func (s *MemoryStore) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}
