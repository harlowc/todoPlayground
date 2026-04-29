package main

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("SERVER_ADDR", "")
	t.Setenv("TODO_STORE", "")
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSLMODE", "")

	cfg := loadConfig()

	if cfg.serverAddr != ":8080" {
		t.Fatalf("serverAddr = %q, want %q", cfg.serverAddr, ":8080")
	}
	if cfg.store != "postgres" {
		t.Fatalf("store = %q, want %q", cfg.store, "postgres")
	}
	if cfg.postgres.host != "localhost" {
		t.Fatalf("host = %q, want %q", cfg.postgres.host, "localhost")
	}
	if cfg.postgres.port != "5432" {
		t.Fatalf("port = %q, want %q", cfg.postgres.port, "5432")
	}
	if cfg.postgres.name != "todo_playground" {
		t.Fatalf("name = %q, want %q", cfg.postgres.name, "todo_playground")
	}
	if cfg.postgres.user != "todo_user" {
		t.Fatalf("user = %q, want %q", cfg.postgres.user, "todo_user")
	}
	if cfg.postgres.password != "" {
		t.Fatalf("password = %q, want empty", cfg.postgres.password)
	}
	if cfg.postgres.sslMode != "disable" {
		t.Fatalf("sslMode = %q, want %q", cfg.postgres.sslMode, "disable")
	}
}

func TestConfiguredStoreOnlySupportsPostgres(t *testing.T) {
	_, err := newConfiguredStore(config{store: "memory"})
	if err == nil {
		t.Fatal("newConfiguredStore() error = nil, want unsupported store error")
	}
	if err.Error() != `unsupported TODO_STORE "memory"; use "postgres"` {
		t.Fatalf("newConfiguredStore() error = %q, want unsupported memory error", err)
	}
}

func TestLoadConfigUsesEnvironment(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":9090")
	t.Setenv("TODO_STORE", "postgres")
	t.Setenv("POSTGRES_HOST", "db")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_DB", "todos_dev")
	t.Setenv("POSTGRES_USER", "app_user")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_SSLMODE", "require")

	cfg := loadConfig()

	if cfg.serverAddr != ":9090" {
		t.Fatalf("serverAddr = %q, want %q", cfg.serverAddr, ":9090")
	}
	if cfg.store != "postgres" {
		t.Fatalf("store = %q, want %q", cfg.store, "postgres")
	}
	if cfg.postgres.host != "db" {
		t.Fatalf("host = %q, want %q", cfg.postgres.host, "db")
	}
	if cfg.postgres.port != "5433" {
		t.Fatalf("port = %q, want %q", cfg.postgres.port, "5433")
	}
	if cfg.postgres.name != "todos_dev" {
		t.Fatalf("name = %q, want %q", cfg.postgres.name, "todos_dev")
	}
	if cfg.postgres.user != "app_user" {
		t.Fatalf("user = %q, want %q", cfg.postgres.user, "app_user")
	}
	if cfg.postgres.password != "secret" {
		t.Fatalf("password = %q, want %q", cfg.postgres.password, "secret")
	}
	if cfg.postgres.sslMode != "require" {
		t.Fatalf("sslMode = %q, want %q", cfg.postgres.sslMode, "require")
	}
}

func TestPostgresConnectionString(t *testing.T) {
	cfg := postgresConfig{
		host:     "localhost",
		port:     "5432",
		name:     "todo_playground",
		user:     "todo user",
		password: "p@ss word",
		sslMode:  "disable",
	}

	got := cfg.connectionString()
	want := "postgres://todo+user:p%40ss+word@localhost:5432/todo_playground?sslmode=disable"
	if got != want {
		t.Fatalf("connectionString() = %q, want %q", got, want)
	}
}
