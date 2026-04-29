package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestMux() http.Handler {
	return newMux(newMemoryStore())
}

func newTestMuxWithToday(store todoStore, today time.Time) http.Handler {
	return newMuxWithToday(store, func() time.Time {
		return today
	})
}

func postForm(t *testing.T, mux http.Handler, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func postFormWithHeaders(t *testing.T, mux http.Handler, path string, values url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
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
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"Buy milk"}})
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, `id="todo-1"`)
	requireContains(t, body, "Buy milk")
	requireContains(t, body, `hx-post="/toggle/1"`)
	requireContains(t, body, `hx-get="/edit/1" hx-target="#todo-1" hx-swap="outerHTML"`)
	requireContains(t, body, `hx-swap-oob="outerHTML"`)
	requireContains(t, body, `placeholder="Add a todo..."`)
	requireContains(t, body, `id="add-error"`)

	rec = get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Buy milk")
}

func TestHomeAllowsHTMXValidationErrorsToRender(t *testing.T) {
	mux := newTestMux()

	rec := get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, "htmx:beforeSwap")
	requireContains(t, body, "document.addEventListener")
	requireContains(t, body, `getResponseHeader("HX-Retarget")`)
	requireContains(t, body, "shouldSwap = true")
}

func TestHomeUsesLocalHTMXAsset(t *testing.T) {
	mux := newTestMux()

	rec := get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, `<script src="/static/htmx.min.js"></script>`)
	requireNotContains(t, body, "unpkg.com")

	rec = get(t, mux, "/static/htmx.min.js")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "htmx")
}

func TestSecurityHeadersAreSet(t *testing.T) {
	mux := newTestMux()

	rec := get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
}

func TestCrossOriginPostsAreRejected(t *testing.T) {
	mux := newTestMux()

	rec := postFormWithHeaders(t, mux, "/add", url.Values{"text": {"Sneaky"}}, map[string]string{
		"Origin": "http://evil.example",
	})
	requireStatus(t, rec, http.StatusForbidden)

	rec = postFormWithHeaders(t, mux, "/add", url.Values{"text": {"Same origin"}}, map[string]string{
		"Origin": "http://example.com",
	})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Same origin")
}

func TestTodoLifecycle(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"Draft plan"}})
	requireStatus(t, rec, http.StatusOK)

	rec = get(t, mux, "/edit/1")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `value="Draft plan"`)
	requireContains(t, rec.Body.String(), `hx-post="/update/1" hx-target="#todo-1" hx-swap="outerHTML"`)
	requireContains(t, rec.Body.String(), `hx-get="/cancel/1" hx-target="#todo-1" hx-swap="outerHTML"`)
	requireContains(t, rec.Body.String(), `id="edit-error-1"`)

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

func TestTodoCompletionToggle(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"Read docs"}})
	requireStatus(t, rec, http.StatusOK)
	requireNotContains(t, rec.Body.String(), "checked")
	requireNotContains(t, rec.Body.String(), `class="todo-item completed"`)

	rec = postForm(t, mux, "/toggle/1", url.Values{"completed": {"on"}})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `class="todo-item completed"`)
	requireContains(t, rec.Body.String(), "checked")

	rec = get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `class="todo-item completed"`)

	rec = postForm(t, mux, "/toggle/1", nil)
	requireStatus(t, rec, http.StatusOK)
	requireNotContains(t, rec.Body.String(), `class="todo-item completed"`)
	requireNotContains(t, rec.Body.String(), "checked")
}

func TestTodoDueDateLifecycle(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{
		"text":     {"Pay rent"},
		"due_date": {"2099-01-02"},
	})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `datetime="2099-01-02"`)
	requireContains(t, rec.Body.String(), "Due 2099-01-02")

	rec = get(t, mux, "/edit/1")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), `type="date" name="due_date" value="2099-01-02"`)

	rec = postForm(t, mux, "/update/1", url.Values{
		"text":     {"Pay rent online"},
		"due_date": {"2099-02-03"},
	})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Pay rent online")
	requireContains(t, rec.Body.String(), `datetime="2099-02-03"`)
	requireContains(t, rec.Body.String(), "Due 2099-02-03")

	rec = get(t, mux, "/")
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Pay rent online")
	requireContains(t, rec.Body.String(), "Due 2099-02-03")
}

func TestTodoDueDateIsOptionalAndValidated(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {"No date needed"}})
	requireStatus(t, rec, http.StatusOK)
	requireNotContains(t, rec.Body.String(), "todo-due-date")

	rec = postForm(t, mux, "/add", url.Values{
		"text":     {"Bad date"},
		"due_date": {"next Friday"},
	})
	requireStatus(t, rec, http.StatusBadRequest)
	requireContains(t, rec.Body.String(), "due date must use YYYY-MM-DD")
	if got := rec.Header().Get("HX-Retarget"); got != "#add-error" {
		t.Fatalf("HX-Retarget = %q, want #add-error", got)
	}
}

func TestAddRejectsPastDueDate(t *testing.T) {
	today := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	mux := newTestMuxWithToday(newMemoryStore(), today)

	rec := postForm(t, mux, "/add", url.Values{
		"text":     {"File report"},
		"due_date": {"2026-04-28"},
	})
	requireStatus(t, rec, http.StatusBadRequest)
	requireContains(t, rec.Body.String(), "due date cannot be before today")
	if got := rec.Header().Get("HX-Retarget"); got != "#add-error" {
		t.Fatalf("HX-Retarget = %q, want #add-error", got)
	}
	if got := rec.Header().Get("HX-Reswap"); got != "innerHTML" {
		t.Fatalf("HX-Reswap = %q, want innerHTML", got)
	}

	rec = postForm(t, mux, "/add", url.Values{
		"text":     {"File report"},
		"due_date": {"2026-04-29"},
	})
	requireStatus(t, rec, http.StatusOK)
}

func TestUpdateOnlyRejectsPastDueDateWhenDateChanges(t *testing.T) {
	today := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	created, err := store.Create("Renew permit", "2026-04-28")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	mux := newTestMuxWithToday(store, today)

	rec := postForm(t, mux, "/update/1", url.Values{
		"text":     {"Renew permit online"},
		"due_date": {created.DueDate},
	})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Renew permit online")
	requireContains(t, rec.Body.String(), "Due 2026-04-28")

	rec = postForm(t, mux, "/update/1", url.Values{
		"text":     {"Renew permit soon"},
		"due_date": {"2026-04-27"},
	})
	requireStatus(t, rec, http.StatusBadRequest)
	requireContains(t, rec.Body.String(), "due date cannot be before today")
	if got := rec.Header().Get("HX-Retarget"); got != "#edit-error-1" {
		t.Fatalf("HX-Retarget = %q, want #edit-error-1", got)
	}

	rec = postForm(t, mux, "/update/1", url.Values{
		"text":     {"Renew permit today"},
		"due_date": {"2026-04-29"},
	})
	requireStatus(t, rec, http.StatusOK)
	requireContains(t, rec.Body.String(), "Due 2026-04-29")
}

func TestTodoTextIsEscaped(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {`<script>alert("x")</script>`}})
	requireStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	requireContains(t, body, `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`)
	requireNotContains(t, body, `<script>alert("x")</script>`)
}

func TestValidationAndMissingTodos(t *testing.T) {
	mux := newTestMux()

	rec := postForm(t, mux, "/add", url.Values{"text": {""}})
	requireStatus(t, rec, http.StatusBadRequest)

	rec = get(t, mux, "/edit/not-a-number")
	requireStatus(t, rec, http.StatusBadRequest)

	rec = get(t, mux, "/edit/42")
	requireStatus(t, rec, http.StatusNotFound)

	rec = postForm(t, mux, "/update/42", url.Values{"text": {"Nope"}})
	requireStatus(t, rec, http.StatusNotFound)

	rec = postForm(t, mux, "/toggle/42", url.Values{"completed": {"on"}})
	requireStatus(t, rec, http.StatusNotFound)

	rec = postForm(t, mux, "/remove/42", nil)
	requireStatus(t, rec, http.StatusNotFound)
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	mux := newTestMux()

	rec := get(t, mux, "/missing")
	requireStatus(t, rec, http.StatusNotFound)
}

func TestPostgresStoreRequiresPassword(t *testing.T) {
	_, err := newConfiguredStore(config{
		store: "postgres",
		postgres: postgresConfig{
			host:    "localhost",
			port:    "5432",
			name:    "todo_playground",
			user:    "todo_user",
			sslMode: "disable",
		},
	})
	if err == nil {
		t.Fatal("newConfiguredStore() error = nil, want POSTGRES_PASSWORD error")
	}
	if !strings.Contains(err.Error(), "POSTGRES_PASSWORD is required") {
		t.Fatalf("newConfiguredStore() error = %q, want POSTGRES_PASSWORD error", err)
	}
}
