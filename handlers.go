package main

import "net/http"

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

	todos, err := a.store.List(r.Context())
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
	if err := a.templates.ExecuteTemplate(w, "base", data); err != nil {
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

	t, err := a.store.Create(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.templates.ExecuteTemplate(w, "reset-add-form", nil); err != nil {
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

	t, found, err := a.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "edit-item", t); err != nil {
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

	current, found, err := a.store.Get(r.Context(), id)
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

	updated, found, err := a.store.Update(r.Context(), id, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "todo-item", a.prepareTodo(updated)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.templates.ExecuteTemplate(w, "reset-add-form", nil); err != nil {
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

	t, found, err := a.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
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

	deleted, err := a.store.Delete(r.Context(), id)
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

	dueDate := tomorrow(a.today())
	if r.FormValue("next_weekday") == "on" {
		dueDate = nextWeekday(a.today())
	}

	recreated, found, err := a.store.CompleteAndRecreate(r.Context(), id, dueDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := a.templates.ExecuteTemplate(w, "todo-item", a.prepareTodo(recreated)); err != nil {
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
	t, found, err := a.store.SetCompleted(r.Context(), id, completed)
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
	if err := a.templates.ExecuteTemplate(w, "todo-item", a.prepareTodo(t)); err != nil {
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

	_, found, err := a.store.Archive(r.Context(), id)
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
