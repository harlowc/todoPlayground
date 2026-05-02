package main

import (
	"html/template"
	"time"
)

type app struct {
	todos     todoRepository
	templates *template.Template
	today     func() time.Time
}

type pageData struct {
	Todos          []todo
	View           string
	CategoryFilter string
	PriorityFilter string
	Search         string
}

func newAppWithToday(todos todoRepository, today func() time.Time) *app {
	return newApp(todos, template.Must(loadTemplates()), today)
}

func newApp(todos todoRepository, templates *template.Template, today func() time.Time) *app {
	return &app{
		todos:     todos,
		templates: templates,
		today:     today,
	}
}

func loadTemplates() (*template.Template, error) {
	return template.ParseFiles("templates.html")
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
