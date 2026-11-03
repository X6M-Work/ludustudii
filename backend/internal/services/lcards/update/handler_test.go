//go:build unit

package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

type mockStore struct {
	updateFn func(ctx context.Context, p updateParams) (int64, error)
}

func (m *mockStore) Update(ctx context.Context, p updateParams) (int64, error) {
	return m.updateFn(ctx, p)
}

func testContext(t *testing.T, id string, body []byte) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/lcards/"+id, bytes.NewReader(body))
	req.Header.Set("If-Match", `"1"`)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	req.SetPathValue("id", id)
	return req, rec
}

func testHandler(t *testing.T, store func(*mockStore), setup func(*Handler)) *Handler {
	t.Helper()

	m := &mockStore{
		updateFn: func(ctx context.Context, p updateParams) (int64, error) {
			t.Fatal("unexpected call to updateFn")
			return 0, errors.New("")
		},
	}
	if store != nil {
		store(m)
	}

	h := &Handler{
		store:           m,
		parseVersion:    func(string) (int64, error) { return 1, nil },
		bind:            func(http.ResponseWriter, *http.Request, any) error { return nil },
		validate:        func(request) error { return nil },
		validateParamID: func(id string) error { return nil },
	}
	if setup != nil {
		setup(h)
	}

	return h
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(h *Handler)
		store       func(*mockStore)
		wantCode    int
		wantErrCode string
	}{
		{
			name: "param error",
			setup: func(h *Handler) {
				h.validateParamID = func(id string) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusBadRequest,
			wantErrCode: erresp.CodeParam,
		},
		{
			name: "binding error",
			setup: func(h *Handler) {
				h.bind = func(w http.ResponseWriter, r *http.Request, i any) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusBadRequest,
			wantErrCode: erresp.CodeBinding,
		},
		{
			name: "validation error",
			setup: func(h *Handler) {
				h.validate = func(req request) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusBadRequest,
			wantErrCode: erresp.CodeValidation,
		},
		{
			name:        "parser error",
			setup:       func(h *Handler) { h.parseVersion = func(string) (int64, error) { return 0, errors.New("invalid") } },
			wantCode:    http.StatusPreconditionRequired,
			wantErrCode: erresp.CodePreconditionRequired,
		},
		{
			name: "version conflict",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) (int64, error) { return 0, domain.ErrVersionConflict }
			},
			wantCode:    http.StatusPreconditionFailed,
			wantErrCode: erresp.CodePreconditionFailed,
		},
		{
			name: "not found error",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) (int64, error) {
					return 0, domain.ErrNotFound
				}
			},
			wantCode:    http.StatusNotFound,
			wantErrCode: erresp.CodeNotFound,
		},
		{
			name: "update error",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) (int64, error) {
					return 0, errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) (int64, error) {
					return 2, nil
				}
			},
			wantCode: http.StatusNoContent,
		},
	}

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	body, _ := json.Marshal(request{Title: "Title", Description: "Description"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup request and request/response recorder
			c, rec := testContext(t, id, body)

			// Setup handler with mock store
			h := testHandler(t, tt.store, tt.setup)

			// Call the handler and check the response
			if err := h.Handle(rec, c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check the status code first
			if rec.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantCode)
			}

			if tt.wantCode == http.StatusNoContent && rec.Header().Get("ETag") != `"2"` {
				t.Errorf("ETag: got %q, want quoted 2", rec.Header().Get("ETag"))
			}
			// Check the error message if expected error message is provided
			if tt.wantErrCode != "" {
				var got erresp.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal error response: %v", err)
				}
				if got.Code != tt.wantErrCode {
					t.Errorf("err code: got %q, want %q", got.Code, tt.wantErrCode)
				}
			}
		})
	}
}

// Bind errors that carry a 413 or 415 status are returned unwritten so the
// router's error handler reports them; other bind errors stay bind_error.
func TestHandler_HandleBindStatusError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "payload too large", err: httpkit.ErrPayloadTooLarge, wantCode: http.StatusRequestEntityTooLarge},
		{name: "unsupported media type", err: httpkit.ErrUnsupportedMediaType, wantCode: http.StatusUnsupportedMediaType},
	}

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	body, _ := json.Marshal(request{Title: "Title", Description: "Description"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, rec := testContext(t, id, body)
			h := testHandler(t, nil, func(h *Handler) {
				h.bind = func(http.ResponseWriter, *http.Request, any) error { return tt.err }
			})

			err := h.Handle(rec, c)
			if got := httpkit.StatusCode(err); got != tt.wantCode {
				t.Fatalf("returned error status: got %d (%v), want %d", got, err, tt.wantCode)
			}
			if rec.Body.Len() != 0 {
				t.Errorf("body: got %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestHandler_Wiring(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(request{
		Title:       "\t Cafe\u0301 Title \n",       // decomposed é = e + U+0301
		Description: "\t Cafe\u0301 Description \n", // decomposed é = e + U+0301
	})
	c, rec := testContext(t, "01A1730E-199D-7F9F-B59D-3BCD6F84066C", body)

	var gotParams updateParams
	m := &mockStore{
		updateFn: func(_ context.Context, p updateParams) (int64, error) {
			gotParams = p
			return 2, nil
		},
	}
	h := NewHandler(m)
	c.Header.Set("If-Match", `"7"`)

	if err := h.Handle(rec, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}

	wantParams := updateParams{
		Version:     7,
		ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c", // lowercased
		Title:       "Caf\u00E9 Title",                      // composed é
		Description: "Caf\u00E9 Description",                // composed é
	}
	if gotParams != wantParams {
		t.Errorf("params: got %+v, want %+v", gotParams, wantParams)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body: got %q, want empty", rec.Body.String())
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	type fieldErr struct{ field, code string }

	tests := []struct {
		name     string
		req      request
		wantErrs []fieldErr
	}{
		{
			name: "valid",
			req:  request{Title: "Title", Description: "Description"},
		},
		{
			name: "invalid",
			req:  request{},
			wantErrs: []fieldErr{
				{"title", validator.CodeRequired},
				{"description", validator.CodeRequired},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validate(tt.req)
			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			got := erresp.Validation(err).Errors
			if len(got) != len(tt.wantErrs) {
				t.Fatalf("errors: got %d, want %d (%v)", len(got), len(tt.wantErrs), err)
			}
			for i, want := range tt.wantErrs {
				if got[i].Field != want.field || got[i].Code != want.code {
					t.Errorf("errors[%d]: got %s/%s, want %s/%s",
						i, got[i].Field, got[i].Code, want.field, want.code)
				}
			}
		})
	}
}

func TestToParams(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	got := toParams(request{Title: "Title", Description: "Description"}, id, 7)
	want := updateParams{Version: 7, ID: id, Title: "Title", Description: "Description"}
	if got != want {
		t.Errorf("params: got %+v, want %+v", got, want)
	}
}
