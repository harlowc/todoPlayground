package main

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type todo struct {
	ID        int
	Text      string
	Completed bool
	Archived  bool
	DueDate   string
	Category  string
	Priority  string
	Notes     string

	OfferNextWeekday  bool
	NextWeekdayPrompt string
}

type todoInput struct {
	Text     string
	DueDate  string
	Category string
	Priority string
	Notes    string
}

func (t todo) PriorityLabel() string {
	switch t.Priority {
	case "low":
		return "Low priority"
	case "high":
		return "High priority"
	default:
		return "Normal priority"
	}
}

type todoStore interface {
	List() ([]todo, error)
	Get(id int) (todo, bool, error)
	Create(input todoInput) (todo, error)
	Update(id int, input todoInput) (todo, bool, error)
	SetCompleted(id int, completed bool) (todo, bool, error)
	Archive(id int) (todo, bool, error)
	Delete(id int) (bool, error)
	Close() error
}

type postgresStore struct {
	db *sql.DB
}

func newPostgresStore(db *sql.DB) *postgresStore {
	return &postgresStore{db: db}
}

func (s *postgresStore) List() ([]todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `SELECT id, text, completed, archived, due_date, category, priority, notes FROM todos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *postgresStore) Get(id int) (todo, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := scanTodo(s.db.QueryRowContext(ctx, `SELECT id, text, completed, archived, due_date, category, priority, notes FROM todos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return todo{}, false, nil
	}
	if err != nil {
		return todo{}, false, err
	}
	return t, true, nil
}

func (s *postgresStore) Create(input todoInput) (todo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := scanTodo(s.db.QueryRowContext(
		ctx,
		`INSERT INTO todos (text, due_date, category, priority, notes)
		 VALUES ($1, NULLIF($2, '')::date, $3, $4, $5)
		 RETURNING id, text, completed, archived, due_date, category, priority, notes`,
		input.Text,
		input.DueDate,
		input.Category,
		input.Priority,
		input.Notes,
	))
	if err != nil {
		return todo{}, err
	}
	return t, nil
}

func (s *postgresStore) Update(id int, input todoInput) (todo, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := scanTodo(s.db.QueryRowContext(
		ctx,
		`UPDATE todos
		 SET text = $2,
		     due_date = NULLIF($3, '')::date,
		     category = $4,
		     priority = $5,
		     notes = $6,
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, text, completed, archived, due_date, category, priority, notes`,
		id,
		input.Text,
		input.DueDate,
		input.Category,
		input.Priority,
		input.Notes,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return todo{}, false, nil
	}
	if err != nil {
		return todo{}, false, err
	}
	return t, true, nil
}

func (s *postgresStore) SetCompleted(id int, completed bool) (todo, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := scanTodo(s.db.QueryRowContext(
		ctx,
		`UPDATE todos
		 SET completed = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, text, completed, archived, due_date, category, priority, notes`,
		id,
		completed,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return todo{}, false, nil
	}
	if err != nil {
		return todo{}, false, err
	}
	return t, true, nil
}

func (s *postgresStore) Archive(id int) (todo, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t, err := scanTodo(s.db.QueryRowContext(
		ctx,
		`UPDATE todos
		 SET archived = TRUE, updated_at = NOW()
		 WHERE id = $1 AND completed = TRUE
		 RETURNING id, text, completed, archived, due_date, category, priority, notes`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return todo{}, false, nil
	}
	if err != nil {
		return todo{}, false, err
	}
	return t, true, nil
}

func (s *postgresStore) Delete(id int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.db.ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (s *postgresStore) Close() error {
	return s.db.Close()
}

type todoScanner interface {
	Scan(dest ...any) error
}

func scanTodo(scanner todoScanner) (todo, error) {
	var t todo
	var dueDate sql.NullTime
	err := scanner.Scan(&t.ID, &t.Text, &t.Completed, &t.Archived, &dueDate, &t.Category, &t.Priority, &t.Notes)
	if err != nil {
		return todo{}, err
	}
	if dueDate.Valid {
		t.DueDate = dueDate.Time.Format("2006-01-02")
	}
	return t, nil
}
