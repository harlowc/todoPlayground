package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestPostgresConnectionAndMigrations(t *testing.T) {
	if getEnv("RUN_DB_TESTS", "") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run Postgres integration tests")
	}

	cfg := loadConfig()

	db, err := openPostgresDB(cfg.postgres)
	if err != nil {
		t.Fatalf("openPostgresDB() error = %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pingPostgres(ctx, db); err != nil {
		t.Fatalf("pingPostgres() error = %v", err)
	}

	if err := runMigrations(ctx, db); err != nil {
		t.Fatalf("runMigrations() error = %v", err)
	}

	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'todos'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check todos table error = %v", err)
	}
	if !exists {
		t.Fatal("todos table does not exist after migrations")
	}

	var hasDueDate bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'todos' AND column_name = 'due_date'
		)
	`).Scan(&hasDueDate)
	if err != nil {
		t.Fatalf("check todos due_date column error = %v", err)
	}
	if !hasDueDate {
		t.Fatal("todos.due_date column does not exist after migrations")
	}

	for _, column := range []string{"category", "priority", "notes", "archived"} {
		var exists bool
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'todos' AND column_name = $1
			)
		`, column).Scan(&exists)
		if err != nil {
			t.Fatalf("check todos.%s column error = %v", column, err)
		}
		if !exists {
			t.Fatalf("todos.%s column does not exist after migrations", column)
		}
	}
}

func TestPostgresStorePersistsLifecycleAcrossReload(t *testing.T) {
	if getEnv("RUN_DB_TESTS", "") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run Postgres integration tests")
	}

	cfg := loadConfig()
	db := openTestDB(t, cfg.postgres)
	defer db.Close()

	resetPostgresTodos(t, db)

	store := newPostgresStore(db)
	created, err := store.Create(todoInput{Text: "Write docs", Priority: "normal"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Completed {
		t.Fatal("created todo is completed, want incomplete by default")
	}
	if created.DueDate != "" {
		t.Fatalf("created.DueDate = %q, want no due date by default", created.DueDate)
	}

	reloaded := newPostgresStore(db)
	todos, err := reloaded.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 1 || todos[0].Text != "Write docs" {
		t.Fatalf("List() = %#v, want one persisted todo", todos)
	}

	updated, found, err := reloaded.Update(created.ID, todoInput{Text: "Ship docs", Priority: "normal"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !found {
		t.Fatal("Update() did not find created todo")
	}
	if updated.Text != "Ship docs" {
		t.Fatalf("updated.Text = %q, want %q", updated.Text, "Ship docs")
	}

	reloadedAgain := newPostgresStore(db)
	got, found, err := reloadedAgain.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() did not find updated todo")
	}
	if got.Text != "Ship docs" {
		t.Fatalf("got.Text = %q, want %q", got.Text, "Ship docs")
	}

	completed, found, err := reloadedAgain.SetCompleted(created.ID, true)
	if err != nil {
		t.Fatalf("SetCompleted() error = %v", err)
	}
	if !found {
		t.Fatal("SetCompleted() did not find updated todo")
	}
	if !completed.Completed {
		t.Fatal("completed.Completed = false, want true")
	}

	completedAgain := newPostgresStore(db)
	got, found, err = completedAgain.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() after SetCompleted error = %v", err)
	}
	if !found {
		t.Fatal("Get() after SetCompleted did not find todo")
	}
	if !got.Completed {
		t.Fatal("got.Completed = false, want persisted true")
	}

	deleted, err := reloadedAgain.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() reported todo was not deleted")
	}

	finalStore := newPostgresStore(db)
	todos, err = finalStore.List()
	if err != nil {
		t.Fatalf("final List() error = %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("final List() = %#v, want no todos after delete", todos)
	}
}

func TestPostgresStorePersistsDueDateAcrossReload(t *testing.T) {
	if getEnv("RUN_DB_TESTS", "") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run Postgres integration tests")
	}

	cfg := loadConfig()
	db := openTestDB(t, cfg.postgres)
	defer db.Close()

	resetPostgresTodos(t, db)

	store := newPostgresStore(db)
	created, err := store.Create(todoInput{Text: "Pay rent", DueDate: "2099-01-02", Priority: "normal"})
	if err != nil {
		t.Fatalf("Create() with due date error = %v", err)
	}
	if created.DueDate != "2099-01-02" {
		t.Fatalf("created.DueDate = %q, want %q", created.DueDate, "2099-01-02")
	}

	reloaded := newPostgresStore(db)
	got, found, err := reloaded.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() after create error = %v", err)
	}
	if !found {
		t.Fatal("Get() after create did not find todo")
	}
	if got.DueDate != "2099-01-02" {
		t.Fatalf("got.DueDate = %q, want %q", got.DueDate, "2099-01-02")
	}

	updated, found, err := reloaded.Update(created.ID, todoInput{Text: "Pay rent online", DueDate: "2099-02-03", Priority: "normal"})
	if err != nil {
		t.Fatalf("Update() with due date error = %v", err)
	}
	if !found {
		t.Fatal("Update() with due date did not find todo")
	}
	if updated.DueDate != "2099-02-03" {
		t.Fatalf("updated.DueDate = %q, want %q", updated.DueDate, "2099-02-03")
	}

	withoutDueDate, found, err := reloaded.Update(created.ID, todoInput{Text: "Pay rent later", Priority: "normal"})
	if err != nil {
		t.Fatalf("Update() clearing due date error = %v", err)
	}
	if !found {
		t.Fatal("Update() clearing due date did not find todo")
	}
	if withoutDueDate.DueDate != "" {
		t.Fatalf("withoutDueDate.DueDate = %q, want no due date", withoutDueDate.DueDate)
	}
}

func TestPostgresStorePersistsCategoryPriorityAndNotesAcrossReload(t *testing.T) {
	if getEnv("RUN_DB_TESTS", "") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run Postgres integration tests")
	}

	cfg := loadConfig()
	db := openTestDB(t, cfg.postgres)
	defer db.Close()

	resetPostgresTodos(t, db)

	store := newPostgresStore(db)
	created, err := store.Create(todoInput{
		Text:     "Plan launch",
		Category: "Work",
		Priority: "high",
		Notes:    "Confirm timeline with design.",
	})
	if err != nil {
		t.Fatalf("Create() with organization fields error = %v", err)
	}
	if created.Category != "Work" || created.Priority != "high" || created.Notes != "Confirm timeline with design." {
		t.Fatalf("created = %#v, want category, priority, and notes persisted", created)
	}

	reloaded := newPostgresStore(db)
	got, found, err := reloaded.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() after create error = %v", err)
	}
	if !found {
		t.Fatal("Get() after create did not find todo")
	}
	if got.Category != "Work" || got.Priority != "high" || got.Notes != "Confirm timeline with design." {
		t.Fatalf("got = %#v, want category, priority, and notes persisted", got)
	}

	updated, found, err := reloaded.Update(created.ID, todoInput{
		Text:     "Plan launch checklist",
		Category: "Ops",
		Priority: "low",
		Notes:    "Share checklist after standup.",
	})
	if err != nil {
		t.Fatalf("Update() with organization fields error = %v", err)
	}
	if !found {
		t.Fatal("Update() with organization fields did not find todo")
	}
	if updated.Category != "Ops" || updated.Priority != "low" || updated.Notes != "Share checklist after standup." {
		t.Fatalf("updated = %#v, want updated category, priority, and notes", updated)
	}
}

func TestPostgresStorePersistsArchiveAcrossReload(t *testing.T) {
	if getEnv("RUN_DB_TESTS", "") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run Postgres integration tests")
	}

	cfg := loadConfig()
	db := openTestDB(t, cfg.postgres)
	defer db.Close()

	resetPostgresTodos(t, db)

	store := newPostgresStore(db)
	created, err := store.Create(todoInput{Text: "Clean completed list", Priority: "normal"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, found, err := store.SetCompleted(created.ID, true); err != nil {
		t.Fatalf("SetCompleted() error = %v", err)
	} else if !found {
		t.Fatal("SetCompleted() did not find created todo")
	}

	archived, found, err := store.Archive(created.ID)
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}
	if !found {
		t.Fatal("Archive() did not find completed todo")
	}
	if !archived.Archived {
		t.Fatalf("archived.Archived = false, want true")
	}

	reloaded := newPostgresStore(db)
	got, found, err := reloaded.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() after archive error = %v", err)
	}
	if !found {
		t.Fatal("Get() after archive did not find todo")
	}
	if !got.Archived {
		t.Fatalf("got.Archived = false, want true")
	}
}

func openTestDB(t *testing.T, cfg postgresConfig) *sql.DB {
	t.Helper()

	db, err := openPostgresDB(cfg)
	if err != nil {
		t.Fatalf("openPostgresDB() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pingPostgres(ctx, db); err != nil {
		db.Close()
		t.Fatalf("pingPostgres() error = %v", err)
	}
	if err := runMigrations(ctx, db); err != nil {
		db.Close()
		t.Fatalf("runMigrations() error = %v", err)
	}

	return db
}

func resetPostgresTodos(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE todos RESTART IDENTITY`); err != nil {
		t.Fatalf("reset todos error = %v", err)
	}
}
