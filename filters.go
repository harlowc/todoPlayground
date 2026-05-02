package main

import (
	"net/http"
	"strings"
	"time"
)

const (
	viewAll       = "all"
	viewActive    = "active"
	viewCompleted = "completed"
	viewScheduled = "scheduled"
	viewToday     = "today"
	viewUpcoming  = "upcoming"
)

type pageFilters struct {
	Category string
	Priority string
	Search   string
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
