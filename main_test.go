package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func resetState(t *testing.T) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	nextID = 1
	todos = nil
}

func postForm(t *testing.T, mux http.Handler, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, mux http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body.String())
	}
}

func requireContains(t *testing.T, body, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("body does not contain %q:\n%s", want, body)
	}
}

func requireNotContains(t *testing.T, body, want string) {
	t.Helper()
	if strings.Contains(body, want) {
		t.Fatalf("body unexpectedly contains %q:\n%s", want, body)
	}
}

func TestAddRendersTodoAndResetsForm(t *testing.T) {
	resetState(t)
	mux := newMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"Buy milk"}})
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, `id="todo-1"`)
	requireContains(t, body, "Buy milk")
	requireContains(t, body, `hx-get="/edit/1" hx-target="#todo-1" hx-swap="outerHTML"`)
	requireContains(t, body, `hx-swap-oob="outerHTML"`)
	requireContains(t, body, `placeholder="Add a todo..."`)

	rec = get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Buy milk")
}

func TestTodoLifecycle(t *testing.T) {
	resetState(t)
	mux := newMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"Draft plan"}})
	requireStatus(t, rec, http.StatusOK)

	rec = get(t, mux, "/edit/1")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `value="Draft plan"`)
	requireContains(t, rec.Body.String(), `hx-post="/update/1" hx-target="#todo-1" hx-swap="outerHTML"`)
	requireContains(t, rec.Body.String(), `hx-get="/cancel/1" hx-target="#todo-1" hx-swap="outerHTML"`)

	rec = postForm(t, mux, "/update/1", url.Values{"text": {"Ship plan"}})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Ship plan")
	requireContains(t, rec.Body.String(), `hx-swap-oob="outerHTML"`)

	rec = get(t, mux, "/cancel/1")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Ship plan")
	requireNotContains(t, rec.Body.String(), "Draft plan")

	rec = postForm(t, mux, "/remove/1", nil)
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `hx-swap-oob="remove"`)

	rec = get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)
	requireNotContains(t, rec.Body.String(), "Ship plan")
}

func TestTodoTextIsEscaped(t *testing.T) {
	resetState(t)
	mux := newMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {`<script>alert("x")</script>`}})
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`)
	requireNotContains(t, body, `<script>alert("x")</script>`)
}

func TestValidationAndMissingTodos(t *testing.T) {
	resetState(t)
	mux := newMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {""}})
	requireStatus(t, rec, http.StatusBadRequest)

	rec = get(t, mux, "/edit/not-a-number")
	requireStatus(t, rec, http.StatusBadRequest)

	rec = get(t, mux, "/edit/42")
	requireStatus(t, rec, http.StatusNotFound)

	rec = postForm(t, mux, "/update/42", url.Values{"text": {"Nope"}})
	requireStatus(t, rec, http.StatusNotFound)
}
