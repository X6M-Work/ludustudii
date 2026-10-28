//go:build integration

package lcards_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/app"
	"github.com/apotourlyan/ludustudii/internal/spa"
	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

func newServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	db := testkit.NewDB(t)
	srv := httptest.NewServer(app.NewRouter(clientApp, db))
	t.Cleanup(srv.Close)
	return srv, db
}

type cardRow struct {
	Title       string
	Description string
}

func insertCard(t *testing.T, db *sql.DB, id, title, description string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO learning_cards (id, title, description) VALUES ($1, $2, $3)`,
		id, title, description,
	)
	if err != nil {
		t.Fatalf("insert card: %v", err)
	}
}

func getCard(t *testing.T, db *sql.DB, id string) cardRow {
	t.Helper()
	var got cardRow
	err := db.QueryRowContext(t.Context(),
		`SELECT title, description FROM learning_cards WHERE id = $1`, id,
	).Scan(&got.Title, &got.Description)
	if err != nil {
		t.Fatalf("query card: %v", err)
	}
	return got
}

func cardExists(t *testing.T, db *sql.DB, id string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(t.Context(),
		`SELECT EXISTS (SELECT 1 FROM learning_cards WHERE id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check card exists: %v", err)
	}
	return exists
}

func TestRouting(t *testing.T) {
	t.Parallel()

	t.Run("GET extra segment under card ID returns 404", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c/extra",
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})

	t.Run("HEAD /api/lcards returns 200 with JSON content-type", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodHead,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
			t.Errorf("content-type: got %q, want JSON", ct)
		}
	})

	t.Run("PATCH /api/lcards returns 405 with Allow header", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPatch,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, body, http.StatusMethodNotAllowed)
		if got := resp.Header.Get(echo.HeaderAllow); got != "GET, HEAD, POST" {
			t.Errorf("Allow: got %q, want %q", got, "GET, HEAD, POST")
		}
		testkit.AssertJSON(t, body, `{"code":"method_not_allowed"}`)
	})

	t.Run("PATCH /api/lcards/{id} returns 405 with Allow header", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPatch,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, body, http.StatusMethodNotAllowed)
		if got := resp.Header.Get(echo.HeaderAllow); got != "GET, HEAD, PUT, DELETE" {
			t.Errorf("Allow: got %q, want %q", got, "GET, HEAD, PUT, DELETE")
		}
		testkit.AssertJSON(t, body, `{"code":"method_not_allowed"}`)
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("empty DB", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		testkit.AssertJSON(t, body, `[]`)
	})

	t.Run("two seeded rows newest first", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const (
			olderID = "01a1730e-199d-7f9f-b59d-3bcd6f84066a"
			newerID = "01a1730e-199d-7f9f-b59d-3bcd6f84066b"
		)
		insertCard(t, db, olderID, "Older Title", "Older Description")
		insertCard(t, db, newerID, "Newer Title", "Newer Description")

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		testkit.AssertJSON(t, body, `[
			{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066b","title":"Newer Title","description":"Newer Description"},
			{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066a","title":"Older Title","description":"Older Description"}
		]`)
	})
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("seeded row", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		insertCard(t, db, id, "Title", "Description")

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		testkit.AssertJSON(t, body, `{
			"id": "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			"title": "Title",
			"description": "Description"
		}`)
	})

	t.Run("uppercase ID", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const (
			lowerID = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
			upperID = "01A1730E-199D-7F9F-B59D-3BCD6F84066C"
		)
		insertCard(t, db, lowerID, "Title", "Description")

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/" + upperID,
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		testkit.AssertJSON(t, body, `{
			"id": "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			"title": "Title",
			"description": "Description"
		}`)
	})

	t.Run("missing UUIDv7", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"param_error"}`)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":"\t Cafe\u0301 Title \n","description":"\t Cafe\u0301 Description \n"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusCreated)

		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal response: %v\n%s", err, body)
		}
		testkit.AssertJSON(t, body, fmt.Sprintf(`{"id":%q}`, got.ID))

		u, err := uuid.Parse(got.ID)
		if err != nil {
			t.Fatalf("parse id %q: %v", got.ID, err)
		}
		if u.Version() != 7 {
			t.Errorf("uuid version: got %d, want 7", u.Version())
		}

		wantRow := cardRow{
			Title:       "Caf\u00E9 Title",
			Description: "Caf\u00E9 Description",
		}
		if row := getCard(t, db, got.ID); row != wantRow {
			t.Errorf("stored row: got %+q, want %+q", row, wantRow)
		}
	})

	t.Run("no Content-Type with a body", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPost,
			Path:   "/api/lcards",
			Body:   `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"bind_error"}`)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"bind_error"}`)
	})

	// An empty body skips the media-type check (app/binder.go) and BindBody returns
	// early, so the request reaches validation with zero-value fields.
	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: echo.MIMEApplicationJSON,
			Body:        "",
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{
			"code": "validation_error",
			"errors": [
				{"field": "title", "code": "required"},
				{"field": "description", "code": "required"}
			]
		}`)
	})

	t.Run("XML body", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: echo.MIMEApplicationXML,
			Body:        `<request><Title>Title</Title><Description>Description</Description></request>`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"bind_error"}`)
	})

	t.Run("body over MaxBodyBytes", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: echo.MIMEApplicationJSON,
			Body:        strings.Repeat("a", app.MaxBodyBytes+1),
		})
		testkit.AssertStatus(t, resp, body, http.StatusRequestEntityTooLarge)
		testkit.AssertJSON(t, body, `{"code":"payload_too_large"}`)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		insertCard(t, db, id, "Old Title", "Old Description")

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Path:        "/api/lcards/" + id,
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":"\t Cafe\u0301 New Title \n","description":"\t Cafe\u0301 New Description \n"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusNoContent)
		if len(body) != 0 {
			t.Errorf("body: got %q, want empty", body)
		}

		wantRow := cardRow{
			Title:       "Caf\u00E9 New Title",
			Description: "Caf\u00E9 New Description",
		}
		if row := getCard(t, db, id); row != wantRow {
			t.Errorf("stored row: got %+q, want %+q", row, wantRow)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Path:        "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Path:        "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"param_error"}`)
	})

	t.Run("XML body", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		original := cardRow{
			Title:       "Original Title",
			Description: "Original Description",
		}
		insertCard(t, db, id, original.Title, original.Description)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Path:        "/api/lcards/" + id,
			ContentType: echo.MIMEApplicationXML,
			Body:        `<request><Title>New Title</Title><Description>New Description</Description></request>`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"bind_error"}`)

		if row := getCard(t, db, id); row != original {
			t.Errorf("stored row: got %+q, want %+q", row, original)
		}
	})

	t.Run("invalid field leaves row unchanged", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		original := cardRow{
			Title:       "Original Title",
			Description: "Original Description",
		}
		insertCard(t, db, id, original.Title, original.Description)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Path:        "/api/lcards/" + id,
			ContentType: echo.MIMEApplicationJSON,
			Body:        `{"title":"","description":"New Description"}`,
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{
			"code": "validation_error",
			"errors": [
				{"field": "title", "code": "required"}
			]
		}`)

		if row := getCard(t, db, id); row != original {
			t.Errorf("stored row: got %+q, want %+q", row, original)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("seeded row then second delete returns 404", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		insertCard(t, db, id, "Title", "Description")

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodDelete,
			Path:   "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, body, http.StatusNoContent)
		if len(body) != 0 {
			t.Errorf("first delete body: got %q, want empty", body)
		}
		if cardExists(t, db, id) {
			t.Errorf("card %s still exists after delete", id)
		}

		resp, body = testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodDelete,
			Path:   "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodDelete,
			Path:   "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, body, http.StatusBadRequest)
		testkit.AssertJSON(t, body, `{"code":"param_error"}`)
	})
}

func TestLifecycle(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t)

	// 1. POST
	resp, body := testkit.Do(t, srv, testkit.DoOpts{
		Method:      http.MethodPost,
		Path:        "/api/lcards",
		ContentType: echo.MIMEApplicationJSON,
		Body:        `{"title":"Initial Title","description":"Initial Description"}`,
	})
	testkit.AssertStatus(t, resp, body, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal POST response: %v\n%s", err, body)
	}

	path := "/api/lcards/" + created.ID

	// 2. GET after create
	resp, body = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, body, http.StatusOK)
	testkit.AssertJSON(t, body, fmt.Sprintf(
		`{"id":%q,"title":"Initial Title","description":"Initial Description"}`,
		created.ID,
	))

	// 3. PUT
	resp, body = testkit.Do(t, srv, testkit.DoOpts{
		Method:      http.MethodPut,
		Path:        path,
		ContentType: echo.MIMEApplicationJSON,
		Body:        `{"title":"Updated Title","description":"Updated Description"}`,
	})
	testkit.AssertStatus(t, resp, body, http.StatusNoContent)
	if len(body) != 0 {
		t.Errorf("PUT body: got %q, want empty", body)
	}

	// 4. GET after update
	resp, body = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, body, http.StatusOK)
	testkit.AssertJSON(t, body, fmt.Sprintf(
		`{"id":%q,"title":"Updated Title","description":"Updated Description"}`,
		created.ID,
	))

	// 5. DELETE
	resp, body = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodDelete,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, body, http.StatusNoContent)
	if len(body) != 0 {
		t.Errorf("DELETE body: got %q, want empty", body)
	}

	// 6. GET after delete -> 404
	resp, body = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, body, http.StatusNotFound)
	testkit.AssertJSON(t, body, `{"code":"not_found"}`)
}
