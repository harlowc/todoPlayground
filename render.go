package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func renderRemoveTodo(w http.ResponseWriter, id int) {
	fmt.Fprintf(w, `<div id="todo-%d" hx-swap-oob="remove">#todo-%d</div>`, id, id)
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
