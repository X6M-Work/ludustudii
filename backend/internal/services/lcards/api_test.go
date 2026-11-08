//go:build integration

package lcards_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/app/api"
	"github.com/apotourlyan/ludustudii/internal/app/router"
	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

func newServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	db := testkit.NewDB(t)
	log := slog.New(slog.DiscardHandler)
	onError := router.ErrorHandler(log)
	srv := httptest.NewServer(router.New(log, onError, api.New(db, onError)))
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
		`INSERT INTO learning_cards (id, title, description, created_at, updated_at) VALUES ($1, $2, $3, now(), now())`,
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c/extra",
		})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
	})

	t.Run("HEAD /api/lcards returns 200 with JSON content-type", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodHead,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("content-type: got %q, want JSON", ct)
		}
	})

	// AutoHandleHEAD runs the GET handler and suppresses its body for both
	// successful lookups and errors. Both API paths document HEAD explicitly.
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(fmt.Sprintf("HEAD /api/lcards/{id} returns %d", status), func(t *testing.T) {
			t.Parallel()
			srv, db := newServer(t)
			const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
			if status == http.StatusOK {
				insertCard(t, db, id, "Title", "Description")
			}
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodHead, Path: "/api/lcards/" + id})
			testkit.AssertStatus(t, resp, status)
			if len(resp.Body) != 0 {
				t.Errorf("body: got %q, want empty", resp.Body)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("content-type: got %q, want JSON", ct)
			}
		})
	}

	t.Run("PATCH /api/lcards returns 405 with Allow header", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPatch,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, http.StatusMethodNotAllowed)
		if got := resp.Header.Get("Allow"); got != "GET, HEAD, POST" {
			t.Errorf("Allow: got %q, want %q", got, "GET, HEAD, POST")
		}
		testkit.AssertJSON(t, resp.Body, `{"code":"method_not_allowed"}`)
	})

	t.Run("PATCH /api/lcards/{id} returns 405 with Allow header", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPatch,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, http.StatusMethodNotAllowed)
		if got := resp.Header.Get("Allow"); got != "DELETE, GET, HEAD, PUT" {
			t.Errorf("Allow: got %q, want %q", got, "DELETE, GET, HEAD, PUT")
		}
		testkit.AssertJSON(t, resp.Body, `{"code":"method_not_allowed"}`)
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("empty DB", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		testkit.AssertJSON(t, resp.Body, `{"items":[],"next_cursor":null}`)
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		testkit.AssertJSON(t, resp.Body, `{
			"items": [
				{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066b","version":1,"title":"Newer Title","description":"Newer Description"},
				{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066a","version":1,"title":"Older Title","description":"Older Description"}
			],
			"next_cursor": null
		}`)
	})

	t.Run("walk pages with limit 1", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const (
			idA = "01a1730e-199d-7f9f-b59d-3bcd6f84066a"
			idB = "01a1730e-199d-7f9f-b59d-3bcd6f84066b"
			idC = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		)
		insertCard(t, db, idA, "Title A", "Description A")
		insertCard(t, db, idB, "Title B", "Description B")
		insertCard(t, db, idC, "Title C", "Description C")

		wantPages := []string{
			`{"items":[{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066c","version":1,"title":"Title C","description":"Description C"}],
				"next_cursor":"01a1730e-199d-7f9f-b59d-3bcd6f84066c"}`,
			`{"items":[{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066b","version":1,"title":"Title B","description":"Description B"}],
				"next_cursor":"01a1730e-199d-7f9f-b59d-3bcd6f84066b"}`,
			`{"items":[{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066a","version":1,"title":"Title A","description":"Description A"}],
				"next_cursor":null}`,
		}

		path := "/api/lcards?limit=1"
		var pages int
		for {
			if pages == len(wantPages) {
				t.Fatalf("pages: got more than %d, want %d", pages, len(wantPages))
			}

			resp := testkit.Do(t, srv, testkit.DoOpts{
				Method: http.MethodGet,
				Path:   path,
			})
			testkit.AssertStatus(t, resp, http.StatusOK)
			testkit.AssertJSON(t, resp.Body, wantPages[pages])
			pages++

			var got struct {
				NextCursor *string `json:"next_cursor"`
			}
			if err := json.Unmarshal(resp.Body, &got); err != nil {
				t.Fatalf("unmarshal response: %v\n%s", err, resp.Body)
			}
			if got.NextCursor == nil {
				break
			}
			path = "/api/lcards?limit=1&cursor=" + *got.NextCursor
		}
		if pages != len(wantPages) {
			t.Errorf("pages: got %v, want %v", pages, len(wantPages))
		}
	})

	t.Run("uppercase cursor", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const (
			idA = "01a1730e-199d-7f9f-b59d-3bcd6f84066a"
			idB = "01a1730e-199d-7f9f-b59d-3bcd6f84066b"
		)
		insertCard(t, db, idA, "Title A", "Description A")
		insertCard(t, db, idB, "Title B", "Description B")

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards?cursor=" + strings.ToUpper(idB),
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		testkit.AssertJSON(t, resp.Body, `{
			"items": [{"id":"01a1730e-199d-7f9f-b59d-3bcd6f84066a","version":1,"title":"Title A","description":"Description A"}],
			"next_cursor": null
		}`)
	})

	t.Run("default limit", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		for range 21 {
			insertCard(t, db, uuid.Must(uuid.NewV7()).String(), "Title", "Description")
		}

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards",
		})
		testkit.AssertStatus(t, resp, http.StatusOK)

		var got struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			NextCursor *string `json:"next_cursor"`
		}
		if err := json.Unmarshal(resp.Body, &got); err != nil {
			t.Fatalf("unmarshal response: %v\n%s", err, resp.Body)
		}
		if len(got.Items) != 20 {
			t.Fatalf("items: got %v, want %v", len(got.Items), 20)
		}
		// The cursor must be the last item on the page, not just any non-null value.
		if got.NextCursor == nil {
			t.Fatalf("next_cursor: got %v, want %v", got.NextCursor, got.Items[19].ID)
		}
		if *got.NextCursor != got.Items[19].ID {
			t.Errorf("next_cursor: got %v, want %v", *got.NextCursor, got.Items[19].ID)
		}
	})

	invalid := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "limit not an integer",
			query: "?limit=abc",
			want:  `{"code":"validation_error","errors":[{"field":"limit","code":"invalid_integer"}]}`,
		},
		{
			name:  "limit not canonical",
			query: "?limit=05",
			want:  `{"code":"validation_error","errors":[{"field":"limit","code":"invalid_integer"}]}`,
		},
		{
			name:  "limit overflow",
			query: "?limit=99999999999999999999999999999",
			want:  `{"code":"validation_error","errors":[{"field":"limit","code":"invalid_integer"}]}`,
		},
		{
			name:  "limit below min",
			query: "?limit=0",
			want:  `{"code":"validation_error","errors":[{"field":"limit","code":"out_of_range"}]}`,
		},
		{
			name:  "limit above max",
			query: "?limit=101",
			want:  `{"code":"validation_error","errors":[{"field":"limit","code":"out_of_range"}]}`,
		},
		{
			name:  "cursor not a uuid",
			query: "?cursor=not-a-uuid",
			want:  `{"code":"validation_error","errors":[{"field":"cursor","code":"invalid_cursor"}]}`,
		},
		{
			name:  "limit and cursor invalid",
			query: "?limit=0&cursor=bad",
			want: `{"code":"validation_error","errors":[
				{"field":"limit","code":"out_of_range"},
				{"field":"cursor","code":"invalid_cursor"}
			]}`,
		},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newServer(t)

			resp := testkit.Do(t, srv, testkit.DoOpts{
				Method: http.MethodGet,
				Path:   "/api/lcards" + tt.query,
			})
			testkit.AssertStatus(t, resp, http.StatusBadRequest)
			testkit.AssertJSON(t, resp.Body, tt.want)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("seeded row", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		insertCard(t, db, id, "Title", "Description")

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		testkit.AssertJSON(t, resp.Body, `{
			"id": "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
 "version": 1,
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/" + upperID,
		})
		testkit.AssertStatus(t, resp, http.StatusOK)
		testkit.AssertJSON(t, resp.Body, `{
			"id": "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
 "version": 1,
			"title": "Title",
			"description": "Description"
		}`)
	})

	t.Run("missing UUIDv7", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{"code":"validation_error","errors":[{"field":"id","code":"invalid_uuid7"}]}`)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/json",
			Body:        `{"title":"\t Cafe\u0301 Title \n","description":"\t Cafe\u0301 Description \n"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusCreated)

		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(resp.Body, &got); err != nil {
			t.Fatalf("unmarshal response: %v\n%s", err, resp.Body)
		}
		testkit.AssertJSON(t, resp.Body, fmt.Sprintf(`{"id":%q}`, got.ID))

		wantLocation := "/api/lcards/" + got.ID
		if location := resp.Header.Get("Location"); location != wantLocation {
			t.Errorf("location: got %q, want %q", location, wantLocation)
		}

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

	t.Run("mixed case JSON media type", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "Application/JSON; Charset=UTF-8",
			Body:        `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusCreated)
	})

	t.Run("no Content-Type with a body", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodPost,
			Path:   "/api/lcards",
			Body:   `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusUnsupportedMediaType)
		testkit.AssertJSON(t, resp.Body, `{"code":"unsupported_media_type"}`)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/json",
			Body:        `{"title":`,
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{"code":"bind_error"}`)
	})

	// An empty body skips the media-type check (app/binder.go) and BindBody returns
	// early, so the request reaches validation with zero-value fields.
	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/json",
			Body:        "",
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/xml",
			Body:        `<request><Title>Title</Title><Description>Description</Description></request>`,
		})
		testkit.AssertStatus(t, resp, http.StatusUnsupportedMediaType)
		testkit.AssertJSON(t, resp.Body, `{"code":"unsupported_media_type"}`)
	})

	t.Run("body over MaxBodyBytes", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/json",
			Body:        strings.Repeat("a", router.MaxBodyBytes+1),
		})
		testkit.AssertStatus(t, resp, http.StatusRequestEntityTooLarge)
		testkit.AssertJSON(t, resp.Body, `{"code":"payload_too_large"}`)
	})

	// Without a Content-Length the body-limit middleware cannot reject the
	// request up front; the limit is hit while the handler binds the body.
	t.Run("chunked body over MaxBodyBytes", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPost,
			Path:        "/api/lcards",
			ContentType: "application/json",
			Body:        strings.Repeat("a", router.MaxBodyBytes+1),
			Chunked:     true,
		})
		testkit.AssertStatus(t, resp, http.StatusRequestEntityTooLarge)
		testkit.AssertJSON(t, resp.Body, `{"code":"payload_too_large"}`)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		insertCard(t, db, id, "Old Title", "Old Description")

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/" + id,
			ContentType: "application/json",
			Body:        `{"title":"\t Cafe\u0301 New Title \n","description":"\t Cafe\u0301 New Description \n"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusNoContent)
		if len(resp.Body) != 0 {
			t.Errorf("body: got %q, want empty", resp.Body)
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			ContentType: "application/json",
			Body:        `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			ContentType: "application/json",
			Body:        `{"title":"Title","description":"Description"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{"code":"validation_error","errors":[{"field":"id","code":"invalid_uuid7"}]}`)
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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/" + id,
			ContentType: "application/xml",
			Body:        `<request><Title>New Title</Title><Description>New Description</Description></request>`,
		})
		testkit.AssertStatus(t, resp, http.StatusUnsupportedMediaType)
		testkit.AssertJSON(t, resp.Body, `{"code":"unsupported_media_type"}`)

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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/" + id,
			ContentType: "application/json",
			Body:        `{"title":"","description":"New Description"}`,
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{
			"code": "validation_error",
			"errors": [
				{"field": "title", "code": "required"}
			]
		}`)

		if row := getCard(t, db, id); row != original {
			t.Errorf("stored row: got %+q, want %+q", row, original)
		}
	})

	t.Run("chunked body over MaxBodyBytes", func(t *testing.T) {
		t.Parallel()
		srv, db := newServer(t)

		const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
		original := cardRow{
			Title:       "Original Title",
			Description: "Original Description",
		}
		insertCard(t, db, id, original.Title, original.Description)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:      http.MethodPut,
			Headers:     http.Header{"If-Match": {`"1"`}},
			Path:        "/api/lcards/" + id,
			ContentType: "application/json",
			Body:        strings.Repeat("a", router.MaxBodyBytes+1),
			Chunked:     true,
		})
		testkit.AssertStatus(t, resp, http.StatusRequestEntityTooLarge)
		testkit.AssertJSON(t, resp.Body, `{"code":"payload_too_large"}`)

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

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:  http.MethodDelete,
			Headers: http.Header{"If-Match": {`"1"`}},
			Path:    "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, http.StatusNoContent)
		if len(resp.Body) != 0 {
			t.Errorf("first delete body: got %q, want empty", resp.Body)
		}
		if cardExists(t, db, id) {
			t.Errorf("card %s still exists after delete", id)
		}

		resp = testkit.Do(t, srv, testkit.DoOpts{
			Method:  http.MethodDelete,
			Headers: http.Header{"If-Match": {`"1"`}},
			Path:    "/api/lcards/" + id,
		})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
	})

	t.Run("urn:uuid prefix", func(t *testing.T) {
		t.Parallel()
		srv, _ := newServer(t)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:  http.MethodDelete,
			Headers: http.Header{"If-Match": {`"1"`}},
			Path:    "/api/lcards/urn:uuid:01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		})
		testkit.AssertStatus(t, resp, http.StatusBadRequest)
		testkit.AssertJSON(t, resp.Body, `{"code":"validation_error","errors":[{"field":"id","code":"invalid_uuid7"}]}`)
	})
}

func TestLifecycle(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t)

	// 1. POST
	resp := testkit.Do(t, srv, testkit.DoOpts{
		Method:      http.MethodPost,
		Path:        "/api/lcards",
		ContentType: "application/json",
		Body:        `{"title":"Initial Title","description":"Initial Description"}`,
	})
	testkit.AssertStatus(t, resp, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		t.Fatalf("unmarshal POST response: %v\n%s", err, resp.Body)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatal("POST response: missing Location header")
	}

	path := "/api/lcards/" + created.ID

	// 2. GET after create, following the Location header
	resp = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   location,
	})
	testkit.AssertStatus(t, resp, http.StatusOK)
	testkit.AssertJSON(t, resp.Body, fmt.Sprintf(
		`{"id":%q,"version":1,"title":"Initial Title","description":"Initial Description"}`,
		created.ID,
	))

	// 3. PUT
	resp = testkit.Do(t, srv, testkit.DoOpts{
		Method:      http.MethodPut,
		Headers:     http.Header{"If-Match": {`"1"`}},
		Path:        path,
		ContentType: "application/json",
		Body:        `{"title":"Updated Title","description":"Updated Description"}`,
	})
	testkit.AssertStatus(t, resp, http.StatusNoContent)
	if len(resp.Body) != 0 {
		t.Errorf("PUT body: got %q, want empty", resp.Body)
	}

	// 4. GET after update
	resp = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, http.StatusOK)
	testkit.AssertJSON(t, resp.Body, fmt.Sprintf(
		`{"id":%q,"version":2,"title":"Updated Title","description":"Updated Description"}`,
		created.ID,
	))

	// 5. DELETE
	resp = testkit.Do(t, srv, testkit.DoOpts{
		Method:  http.MethodDelete,
		Headers: http.Header{"If-Match": {`"2"`}},
		Path:    path,
	})
	testkit.AssertStatus(t, resp, http.StatusNoContent)
	if len(resp.Body) != 0 {
		t.Errorf("DELETE body: got %q, want empty", resp.Body)
	}

	// 6. GET after delete -> 404
	resp = testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodGet,
		Path:   path,
	})
	testkit.AssertStatus(t, resp, http.StatusNotFound)
	testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
}

func TestPreconditions(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t)
	created := testkit.Do(t, srv, testkit.DoOpts{
		Method: http.MethodPost, Path: "/api/lcards", ContentType: "application/json",
		Body: `{"title":"Original","description":"Original description"}`,
	})
	testkit.AssertStatus(t, created, http.StatusCreated)
	if got := created.Header.Get("ETag"); got != `"1"` {
		t.Fatalf("POST ETag: %q", got)
	}
	path := created.Header.Get("Location")
	get := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
	testkit.AssertStatus(t, get, http.StatusOK)
	var card struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(get.Body, &card); err != nil {
		t.Fatal(err)
	}
	if card.Version != 1 || get.Header.Get("ETag") != fmt.Sprintf(`"%d"`, card.Version) {
		t.Fatalf("GET version: %+v, ETag %q", card, get.Header.Get("ETag"))
	}
	head := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodHead, Path: path})
	testkit.AssertStatus(t, head, http.StatusOK)
	if head.Header.Get("ETag") != get.Header.Get("ETag") {
		t.Fatalf("HEAD ETag: %q", head.Header.Get("ETag"))
	}
	body := `{"title":"Updated","description":"Updated description"}`
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		for _, value := range []string{"", "*", `W/"1"`, `"1", "2"`, "1", `"0"`, `"-1"`, `"9223372036854775808"`} {
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path, Headers: http.Header{"If-Match": {value}}, ContentType: "application/json", Body: body})
			testkit.AssertStatus(t, resp, http.StatusPreconditionRequired)
			testkit.AssertJSON(t, resp.Body, `{"code":"precondition_required"}`)
		}
		resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path, Headers: http.Header{"If-Match": {`"1"`, `"1"`}}, ContentType: "application/json", Body: body})
		testkit.AssertStatus(t, resp, http.StatusPreconditionRequired)
	}
	put := testkit.DoOpts{Method: http.MethodPut, Path: path, Headers: http.Header{"If-Match": {get.Header.Get("ETag")}}, ContentType: "application/json", Body: body}
	updated := testkit.Do(t, srv, put)
	testkit.AssertStatus(t, updated, http.StatusNoContent)
	if got := updated.Header.Get("ETag"); got != `"2"` {
		t.Fatalf("PUT ETag: %q", got)
	}
	stale := testkit.Do(t, srv, put)
	testkit.AssertStatus(t, stale, http.StatusPreconditionFailed)
	testkit.AssertJSON(t, stale.Body, `{"code":"precondition_failed"}`)
	staleDelete := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodDelete, Path: path, Headers: http.Header{"If-Match": {`"1"`}}})
	testkit.AssertStatus(t, staleDelete, http.StatusPreconditionFailed)
	testkit.AssertJSON(t, staleDelete.Body, `{"code":"precondition_failed"}`)
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		unknown := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066d", Headers: http.Header{"If-Match": {`"99"`}}, ContentType: "application/json", Body: body})
		testkit.AssertStatus(t, unknown, http.StatusNotFound)
	}
	final := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
	testkit.AssertStatus(t, final, http.StatusOK)
	if got := final.Header.Get("ETag"); got != `"2"` {
		t.Fatalf("version after stale writes: %q", got)
	}
	deleted := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodDelete, Path: path, Headers: http.Header{"If-Match": {`"2"`}}})
	testkit.AssertStatus(t, deleted, http.StatusNoContent)
}

func TestConcurrentUpdates(t *testing.T) {
	t.Parallel()
	srv, db := newServer(t)
	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	insertCard(t, db, id, "Original", "Original description")
	const n = 12
	start := make(chan struct{})
	results := make(chan int, n)
	var workers sync.WaitGroup
	for i := range n {
		workers.Go(func() {
			<-start
			resp := testkit.Do(t, srv, testkit.DoOpts{
				Method: http.MethodPut, Path: "/api/lcards/" + id,
				Headers: http.Header{"If-Match": {`"1"`}}, ContentType: "application/json",
				Body: fmt.Sprintf(`{"title":"Writer %d","description":"Description"}`, i),
			})
			results <- resp.StatusCode
		})
	}
	close(start)
	workers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for status := range results {
		switch status {
		case http.StatusNoContent:
			successes++
		case http.StatusPreconditionFailed:
			conflicts++
		default:
			t.Errorf("unexpected status: %d", status)
		}
	}
	if successes != 1 || conflicts != n-1 {
		t.Fatalf("successes %d, conflicts %d; want 1 and %d", successes, conflicts, n-1)
	}
	final := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/lcards/" + id})
	testkit.AssertStatus(t, final, http.StatusOK)
	var card struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(final.Body, &card); err != nil {
		t.Fatal(err)
	}
	if card.Version != 2 || final.Header.Get("ETag") != `"2"` {
		t.Fatalf("final version %d, ETag %q", card.Version, final.Header.Get("ETag"))
	}
}

func TestUnknownRequestFields(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			srv, db := newServer(t)
			const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
			original := cardRow{Title: "Original", Description: "Original description"}
			insertCard(t, db, id, original.Title, original.Description)
			path := "/api/lcards"
			if method == http.MethodPut {
				path += "/" + id
			}
			resp := testkit.Do(t, srv, testkit.DoOpts{
				Method:      method,
				Path:        path,
				Headers:     http.Header{"If-Match": {`"1"`}},
				ContentType: "application/json",
				Body:        `{"title":"Changed","description":"Changed description","extra":true}`,
			})
			testkit.AssertStatus(t, resp, http.StatusBadRequest)
			testkit.AssertJSON(t, resp.Body, `{"code":"bind_error"}`)
			if row := getCard(t, db, id); row != original {
				t.Errorf("stored row: got %+q, want %+q", row, original)
			}
			var count int
			if err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM learning_cards`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Errorf("card count: got %d, want 1", count)
			}
		})
	}
}
