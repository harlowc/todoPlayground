package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	tmpl   = template.Must(template.ParseFiles("templates.html"))
	mu     sync.RWMutex
	nextID int = 1
	todos  []todo
)

type todo struct {
	ID   int
	Text string
}

func home(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	if err := tmpl.ExecuteTemplate(w, "base", todos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func add(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	mu.Lock()
	t := todo{ID: nextID, Text: text}
	todos = append(todos, t)
	nextID++
	mu.Unlock()
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "reset-add-form", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func edit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	mu.RLock()
	defer mu.RUnlock()
	for _, t := range todos {
		if t.ID == id {
			w.Header().Set("Content-Type", "text/html")
			if err := tmpl.ExecuteTemplate(w, "edit-item", t); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	mu.Lock()
	var updated todo
	for i, t := range todos {
		if t.ID == id {
			todos[i].Text = text
			updated = todos[i]
			break
		}
	}
	mu.Unlock()
	if updated.ID == 0 {
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

func cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	mu.RLock()
	var text string
	for _, t := range todos {
		if t.ID == id {
			text = t.Text
			break
		}
	}
	mu.RUnlock()
	if text == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "todo-item", todo{ID: id, Text: text}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func remove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for i, t := range todos {
		if t.ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			break
		}
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="todo-%d" hx-swap-oob="remove">#todo-%d</div>`, id, id)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	mux.HandleFunc("POST /add", add)
	mux.HandleFunc("POST /remove/{id}", remove)
	mux.HandleFunc("GET /edit/{id}", edit)
	mux.HandleFunc("POST /update/{id}", update)
	mux.HandleFunc("GET /cancel/{id}", cancel)
	return mux
}

func main() {
	fmt.Println("Server starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", newMux()))
}
