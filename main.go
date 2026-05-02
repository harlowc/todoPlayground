package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func newPostgresRepositoryFromConfig(cfg config) (*postgresRepository, error) {
	if cfg.postgres.password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
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

	return newPostgresRepository(db), nil
}

func main() {
	cfg := loadConfig()

	repo, err := newPostgresRepositoryFromConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	templates, err := loadTemplates()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server starting at http://localhost%s\n", cfg.serverAddr)
	log.Fatal(http.ListenAndServe(cfg.serverAddr, newMuxWithTemplates(repo, templates, time.Now)))
}
