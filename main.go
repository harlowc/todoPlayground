package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func newConfiguredStore(cfg config) (todoStore, error) {
	switch cfg.store {
	case "postgres":
		if cfg.postgres.password == "" {
			return nil, fmt.Errorf("POSTGRES_PASSWORD is required when TODO_STORE=postgres")
		}

		db, err := openPostgresDB(cfg.postgres)
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := pingPostgres(ctx, db); err != nil {
			db.Close()
			return nil, err
		}
		if err := runMigrations(ctx, db); err != nil {
			db.Close()
			return nil, err
		}

		return newPostgresStore(db), nil
	default:
		return nil, fmt.Errorf("unsupported TODO_STORE %q; use \"postgres\"", cfg.store)
	}
}

func main() {
	cfg := loadConfig()

	store, err := newConfiguredStore(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	templates, err := loadTemplates()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server starting at http://localhost%s\n", cfg.serverAddr)
	log.Fatal(http.ListenAndServe(cfg.serverAddr, newMuxWithTemplates(store, templates, time.Now)))
}
