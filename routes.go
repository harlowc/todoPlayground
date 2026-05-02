package main

import (
	"html/template"
	"net/http"
	"time"
)

func newMux(todos todoRepository) http.Handler {
	return newMuxWithToday(todos, time.Now)
}

func newMuxWithToday(todos todoRepository, today func() time.Time) http.Handler {
	return newMuxWithTemplates(todos, template.Must(loadTemplates()), today)
}

func newMuxWithTemplates(todos todoRepository, templates *template.Template, today func() time.Time) http.Handler {
	app := newApp(todos, templates, today)
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("POST /add", app.add)
	mux.HandleFunc("POST /done-tomorrow/{id}", app.doneTomorrow)
	mux.HandleFunc("POST /remove/{id}", app.remove)
	mux.HandleFunc("POST /archive/{id}", app.archive)
	mux.HandleFunc("POST /completed/{id}", app.setCompleted)
	mux.HandleFunc("GET /edit/{id}", app.edit)
	mux.HandleFunc("POST /update/{id}", app.update)
	mux.HandleFunc("GET /cancel/{id}", app.cancel)
	return securityMiddleware(mux)
}
