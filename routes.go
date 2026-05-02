package main

import (
	"html/template"
	"net/http"
	"time"
)

func newMux(store todoStore) http.Handler {
	return newMuxWithToday(store, time.Now)
}

func newMuxWithToday(store todoStore, today func() time.Time) http.Handler {
	return newMuxWithTemplates(store, template.Must(loadTemplates()), today)
}

func newMuxWithTemplates(store todoStore, templates *template.Template, today func() time.Time) http.Handler {
	app := newApp(store, templates, today)
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
