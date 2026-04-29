package main

import (
	"fmt"
	"net/url"
	"os"
)

type config struct {
	serverAddr string
	store      string
	postgres   postgresConfig
}

type postgresConfig struct {
	host     string
	port     string
	name     string
	user     string
	password string
	sslMode  string
}

func loadConfig() config {
	return config{
		serverAddr: getEnv("SERVER_ADDR", ":8080"),
		store:      getEnv("TODO_STORE", "postgres"),
		postgres: postgresConfig{
			host:     getEnv("POSTGRES_HOST", "localhost"),
			port:     getEnv("POSTGRES_PORT", "5432"),
			name:     getEnv("POSTGRES_DB", "todo_playground"),
			user:     getEnv("POSTGRES_USER", "todo_user"),
			password: getEnv("POSTGRES_PASSWORD", ""),
			sslMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func (c postgresConfig) connectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(c.user),
		url.QueryEscape(c.password),
		c.host,
		c.port,
		c.name,
		url.QueryEscape(c.sslMode),
	)
}
