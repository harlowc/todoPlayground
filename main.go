package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var tmpl = template.Must(template.ParseFiles("templates.html"))

const addErrorTarget = "#add-error"

type app struct {
	store todoStore
	today func() time.Time
}

func newAppWithToday(store todoStore, today func() time.Time) *app {
	return &app{
		store: store,
		today: today,
	}
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
		renderValidationError(w, addErrorTarget, "text is required")
		return
	}

	dueDate, ok, message := parseDueDate(r)
	if !ok {
		renderValidationError(w, addErrorTarget, message)
		return
	}
	if ok, message := validateDueDateNotPast(dueDate, a.today()); !ok {
		renderValidationError(w, addErrorTarget, message)
		return
	}

	t, err := a.store.Create(text, dueDate)
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
		renderValidationError(w, editErrorTarget(id), "text is required")
		return
	}

	dueDate, ok, message := parseDueDate(r)
	if !ok {
		renderValidationError(w, editErrorTarget(id), message)
		return
	}

	current, found, err := a.store.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if ok, message := validateDueDateNotPast(dueDate, a.today()); dueDate != current.DueDate && !ok {
		renderValidationError(w, editErrorTarget(id), message)
		return
	}

	updated, found, err := a.store.Update(id, text, dueDate)
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

func parseDueDate(r *http.Request) (string, bool, string) {
	dueDate := r.FormValue("due_date")
	if dueDate == "" {
		return "", true, ""
	}

	if _, err := time.Parse("2006-01-02", dueDate); err != nil {
		return "", false, "due date must use YYYY-MM-DD"
	}
	return dueDate, true, ""
}

func validateDueDateNotPast(dueDate string, today time.Time) (bool, string) {
	if dueDate == "" {
		return true, ""
	}

	parsedDueDate, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return false, "due date must use YYYY-MM-DD"
	}

	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, parsedDueDate.Location())
	if parsedDueDate.Before(todayDate) {
		return false, "due date cannot be before today"
	}
	return true, ""
}

func editErrorTarget(id int) string {
	return fmt.Sprintf("#edit-error-%d", id)
}

func renderValidationError(w http.ResponseWriter, target, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Retarget", target)
	w.Header().Set("HX-Reswap", "innerHTML")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, template.HTMLEscapeString(message))
}

func newMux(store todoStore) http.Handler {
	return newMuxWithToday(store, time.Now)
}

func newMuxWithToday(store todoStore, today func() time.Time) http.Handler {
	app := newAppWithToday(store, today)
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("POST /add", app.add)
	mux.HandleFunc("POST /remove/{id}", app.remove)
	mux.HandleFunc("POST /toggle/{id}", app.toggle)
	mux.HandleFunc("GET /edit/{id}", app.edit)
	mux.HandleFunc("POST /update/{id}", app.update)
	mux.HandleFunc("GET /cancel/{id}", app.cancel)
	return securityMiddleware(mux)
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		if !sameOriginRequest(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func sameOriginRequest(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	expectedScheme := "http"
	if r.TLS != nil {
		expectedScheme = "https"
	}
	return parsedOrigin.Scheme == expectedScheme && parsedOrigin.Host == r.Host
}

func newConfiguredStore(cfg config) (todoStore, error) {
	switch cfg.store {
	case "memory":
		return newMemoryStore(), nil
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
