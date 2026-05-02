package main

import (
	"net/http"
	"strconv"
	"time"
)

const addErrorTarget = "#add-error"

func parseTodoID(w http.ResponseWriter, r *http.Request) (int, bool) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
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
