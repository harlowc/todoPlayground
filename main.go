package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

var tmpl = template.Must(template.ParseFiles("templates.html"))

type app struct {
	store todoStore
}

func newApp(store todoStore) *app {
	return &app{store: store}
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	todos, err := a.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", todos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) add(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}

	t, err := a.store.Create(text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "reset-add-form", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) edit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	t, found, err := a.store.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "edit-item", t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}

	updated, found, err := a.store.Update(id, text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "reset-add-form", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	t, found, err := a.store.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) remove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	deleted, err := a.store.Delete(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="todo-%d" hx-swap-oob="remove">#todo-%d</div>`, id, id)
}

func (a *app) toggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	completed := r.FormValue("completed") == "on"
	t, found, err := a.store.SetCompleted(id, completed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseTodoID(w http.ResponseWriter, r *http.Request) (int, bool) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func newMux(store todoStore) *http.ServeMux {
	app := newApp(store)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("POST /add", app.add)
	mux.HandleFunc("POST /remove/{id}", app.remove)
	mux.HandleFunc("POST /toggle/{id}", app.toggle)
	mux.HandleFunc("GET /edit/{id}", app.edit)
	mux.HandleFunc("POST /update/{id}", app.update)
	mux.HandleFunc("GET /cancel/{id}", app.cancel)
	return mux
}

func newConfiguredStore(cfg config) (todoStore, error) {
	switch cfg.store {
	case "memory":
		return newMemoryStore(), nil
	case "postgres":
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
		return nil, fmt.Errorf("unsupported TODO_STORE %q", cfg.store)
	}
}

func main() {
	cfg := loadConfig()

	store, err := newConfiguredStore(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	fmt.Printf("Server starting at http://localhost%s\n", cfg.serverAddr)
	log.Fatal(http.ListenAndServe(cfg.serverAddr, newMux(store)))
}
