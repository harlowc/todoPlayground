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
	created, err := store.Create("Write docs")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Completed {
		t.Fatal("created todo is completed, want incomplete by default")
	}

	reloaded := newPostgresStore(db)
	todos, err := reloaded.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 1 || todos[0].Text != "Write docs" {
		t.Fatalf("List() = %#v, want one persisted todo", todos)
	}

	updated, found, err := reloaded.Update(created.ID, "Ship docs")
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
