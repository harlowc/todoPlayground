package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var tmpl = template.Must(template.ParseFiles("templates.html"))

const addErrorTarget = "#add-error"

const (
	viewAll       = "all"
	viewActive    = "active"
	viewCompleted = "completed"
	viewScheduled = "scheduled"
	viewToday     = "today"
	viewUpcoming  = "upcoming"
)

type app struct {
	store todoStore
	today func() time.Time
}

type pageData struct {
	Todos          []todo
	View           string
	CategoryFilter string
	PriorityFilter string
	Search         string
}

type pageFilters struct {
	Category string
	Priority string
	Search   string
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

	view, ok := parseView(r)
	if !ok {
		http.Error(w, "unknown view", http.StatusBadRequest)
		return
	}
	filters, ok, message := parsePageFilters(r)
	if !ok {
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	todos, err := a.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := pageData{
		Todos:          a.prepareTodos(filterTodos(todos, view, filters, a.today())),
		View:           view,
		CategoryFilter: filters.Category,
		PriorityFilter: filters.Priority,
		Search:         filters.Search,
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) add(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	input, ok, message := parseTodoInput(r)
	if !ok {
		renderValidationError(w, addErrorTarget, message)
		return
	}
	if ok, message := validateDueDateNotPast(input.DueDate, a.today()); !ok {
		renderValidationError(w, addErrorTarget, message)
		return
	}

	t, err := a.store.Create(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
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

	input, ok, message := parseTodoInput(r)
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
	if ok, message := validateDueDateNotPast(input.DueDate, a.today()); input.DueDate != current.DueDate && !ok {
		renderValidationError(w, editErrorTarget(id), message)
		return
	}

	updated, found, err := a.store.Update(id, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", a.prepareTodo(updated)); err != nil {
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
	if err := tmpl.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
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
	renderRemoveTodo(w, id)
}

func (a *app) doneTomorrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
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

	if _, found, err := a.store.SetCompleted(id, true); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	dueDate := tomorrow(a.today())
	if r.FormValue("next_weekday") == "on" {
		dueDate = nextWeekday(a.today())
	}

	recreated, err := a.store.Create(todoInput{
		Text:     current.Text,
		DueDate:  dueDate,
		Category: current.Category,
		Priority: current.Priority,
		Notes:    current.Notes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", a.prepareTodo(recreated)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) setCompleted(w http.ResponseWriter, r *http.Request) {
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
	if completed {
		renderRemoveTodo(w, id)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) archive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, ok := parseTodoID(w, r)
	if !ok {
		return
	}

	_, found, err := a.store.Archive(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderRemoveTodo(w, id)
}

func renderRemoveTodo(w http.ResponseWriter, id int) {
	fmt.Fprintf(w, `<div id="todo-%d" hx-swap-oob="remove">#todo-%d</div>`, id, id)
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

func parseView(r *http.Request) (string, bool) {
	view := r.URL.Query().Get("view")
	if view == "" {
		return viewAll, true
	}
	switch view {
	case viewAll, viewActive, viewCompleted, viewScheduled, viewToday, viewUpcoming:
		return view, true
	default:
		return "", false
	}
}

func parsePageFilters(r *http.Request) (pageFilters, bool, string) {
	filters := pageFilters{
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
		Priority: strings.TrimSpace(r.URL.Query().Get("priority")),
		Search:   strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if filters.Priority != "" && !validPriority(filters.Priority) {
		return pageFilters{}, false, "priority filter must be low, normal, or high"
	}
	return filters, true, ""
}

func filterTodos(todos []todo, view string, filters pageFilters, today time.Time) []todo {
	filtered := make([]todo, 0, len(todos))
	for _, t := range todos {
		if t.Archived {
			continue
		}
		if !matchesView(t, view, today) {
			continue
		}
		if !matchesFilters(t, filters) {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func matchesView(t todo, view string, today time.Time) bool {
	switch view {
	case viewAll:
		return true
	case viewActive:
		return !t.Completed
	case viewCompleted:
		return t.Completed
	case viewScheduled:
		return !t.Completed && t.DueDate != ""
	case viewToday:
		return !t.Completed && sameDate(t.DueDate, today)
	case viewUpcoming:
		return !t.Completed && futureDate(t.DueDate, today)
	default:
		return false
	}
}

func matchesFilters(t todo, filters pageFilters) bool {
	if filters.Category != "" && !strings.EqualFold(t.Category, filters.Category) {
		return false
	}
	if filters.Priority != "" && t.Priority != filters.Priority {
		return false
	}
	if filters.Search != "" {
		query := strings.ToLower(filters.Search)
		searchable := strings.ToLower(strings.Join([]string{t.Text, t.Category, t.Notes}, " "))
		if !strings.Contains(searchable, query) {
			return false
		}
	}
	return true
}

func sameDate(dueDate string, today time.Time) bool {
	parsed, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return false
	}
	return parsed.Format("2006-01-02") == today.Format("2006-01-02")
}

func futureDate(dueDate string, today time.Time) bool {
	parsed, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return false
	}
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, parsed.Location())
	return parsed.After(todayDate)
}

func parseTodoInput(r *http.Request) (todoInput, bool, string) {
	input := todoInput{
		Text:     r.FormValue("text"),
		Category: r.FormValue("category"),
		Priority: r.FormValue("priority"),
		Notes:    r.FormValue("notes"),
	}
	if input.Text == "" {
		return todoInput{}, false, "text is required"
	}

	dueDate, ok, message := parseDueDate(r)
	if !ok {
		return todoInput{}, false, message
	}
	input.DueDate = dueDate

	if input.Priority == "" {
		input.Priority = "normal"
	}
	if !validPriority(input.Priority) {
		return todoInput{}, false, "priority must be low, normal, or high"
	}
	return input, true, ""
}

func validPriority(priority string) bool {
	switch priority {
	case "low", "normal", "high":
		return true
	default:
		return false
	}
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

func tomorrow(today time.Time) string {
	return today.AddDate(0, 0, 1).Format("2006-01-02")
}

func nextWeekday(today time.Time) string {
	next := today.AddDate(0, 0, 1)
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}
	return next.Format("2006-01-02")
}

func tomorrowFallsOnWeekend(today time.Time) bool {
	next := today.AddDate(0, 0, 1)
	return next.Weekday() == time.Saturday || next.Weekday() == time.Sunday
}

func nextWeekdayPrompt(today time.Time) string {
	tomorrowDate := today.AddDate(0, 0, 1)
	next, err := time.Parse("2006-01-02", nextWeekday(today))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Tomorrow is %s. Recreate for %s instead?", tomorrowDate.Weekday(), next.Weekday())
}

func (a *app) prepareTodos(todos []todo) []todo {
	prepared := make([]todo, len(todos))
	for i, t := range todos {
		prepared[i] = a.prepareTodo(t)
	}
	return prepared
}

func (a *app) prepareTodo(t todo) todo {
	if !t.Completed && tomorrowFallsOnWeekend(a.today()) {
		t.OfferNextWeekday = true
		t.NextWeekdayPrompt = nextWeekdayPrompt(a.today())
	}
	return t
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
	mux.HandleFunc("POST /done-tomorrow/{id}", app.doneTomorrow)
	mux.HandleFunc("POST /remove/{id}", app.remove)
	mux.HandleFunc("POST /archive/{id}", app.archive)
	mux.HandleFunc("POST /completed/{id}", app.setCompleted)
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

	fmt.Printf("Server starting at http://localhost%s\n", cfg.serverAddr)
	log.Fatal(http.ListenAndServe(cfg.serverAddr, newMux(store)))
}
