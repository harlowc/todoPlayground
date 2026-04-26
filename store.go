package main

import "sync"

type todo struct {
	ID   int
	Text string
}

type todoStore interface {
	List() []todo
	Get(id int) (todo, bool)
	Create(text string) todo
	Update(id int, text string) (todo, bool)
	Delete(id int) bool
}

type memoryStore struct {
	mu     sync.RWMutex
	nextID int
	todos  []todo
}

func newMemoryStore() *memoryStore {
	return &memoryStore{nextID: 1}
}

func (s *memoryStore) List() []todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]todo, len(s.todos))
	copy(todos, s.todos)
	return todos
}

func (s *memoryStore) Get(id int) (todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.todos {
		if t.ID == id {
			return t, true
		}
	}
	return todo{}, false
}

func (s *memoryStore) Create(text string) todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := todo{ID: s.nextID, Text: text}
	s.todos = append(s.todos, t)
	s.nextID++
	return t
}

func (s *memoryStore) Update(id int, text string) (todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.todos {
		if t.ID == id {
			s.todos[i].Text = text
			return s.todos[i], true
		}
	}
	return todo{}, false
}

func (s *memoryStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.todos {
		if t.ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return true
		}
	}
	return false
}
