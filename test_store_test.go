package main

import (
	"context"
	"sync"
)

type memoryStore struct {
	mu     sync.RWMutex
	nextID int
	todos  []todo
}

func newMemoryStore() *memoryStore {
	return &memoryStore{nextID: 1}
}

func (s *memoryStore) List(ctx context.Context) ([]todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]todo, len(s.todos))
	copy(todos, s.todos)
	return todos, nil
}

func (s *memoryStore) Get(ctx context.Context, id int) (todo, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.todos {
		if t.ID == id {
			return t, true, nil
		}
	}
	return todo{}, false, nil
}

func (s *memoryStore) Create(ctx context.Context, input todoInput) (todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := todo{
		ID:       s.nextID,
		Text:     input.Text,
		DueDate:  input.DueDate,
		Category: input.Category,
		Priority: input.Priority,
		Notes:    input.Notes,
	}
	s.todos = append(s.todos, t)
	s.nextID++
	return t, nil
}

func (s *memoryStore) Update(ctx context.Context, id int, input todoInput) (todo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Text = input.Text
			s.todos[i].DueDate = input.DueDate
			s.todos[i].Category = input.Category
			s.todos[i].Priority = input.Priority
			s.todos[i].Notes = input.Notes
			return s.todos[i], true, nil
		}
	}
	return todo{}, false, nil
}

func (s *memoryStore) SetCompleted(ctx context.Context, id int, completed bool) (todo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Completed = completed
			return s.todos[i], true, nil
		}
	}
	return todo{}, false, nil
}

func (s *memoryStore) CompleteAndRecreate(ctx context.Context, id int, dueDate string) (todo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Completed = true
			t := todo{
				ID:       s.nextID,
				Text:     s.todos[i].Text,
				DueDate:  dueDate,
				Category: s.todos[i].Category,
				Priority: s.todos[i].Priority,
				Notes:    s.todos[i].Notes,
			}
			s.todos = append(s.todos, t)
			s.nextID++
			return t, true, nil
		}
	}
	return todo{}, false, nil
}

func (s *memoryStore) Archive(ctx context.Context, id int) (todo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Archived = true
			return s.todos[i], true, nil
		}
	}
	return todo{}, false, nil
}

func (s *memoryStore) Delete(ctx context.Context, id int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.todos {
		if t.ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (s *memoryStore) Close() error {
	return nil
}
