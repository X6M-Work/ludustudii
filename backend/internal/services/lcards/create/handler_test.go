//go:build unit

package create

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
)

type mockStore struct {
	createFn func(ctx context.Context, p createParams) (row, error)
}

func (m *mockStore) Create(ctx context.Context, p createParams) (row, error) {
	return m.createFn(ctx, p)
}

func testContext(t *testing.T, body []byte) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/lcards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	return req, rec
}

func testHandler(t *testing.T, store func(*mockStore), setup func(*Handler)) *Handler {
	t.Helper()

	m := &mockStore{
		createFn: func(ctx context.Context, p createParams) (row, error) {
			t.Fatal("unexpected call to createFn")
			return row{}, errors.New("")
		},
	}
	if store != nil {
		store(m)
	}

	h := &Handler{
		store:    m,
		bind:     func(*http.Request, any) error { return nil },
		validate: func(request) error { return nil },
	}
	if setup != nil {
		setup(h)
	}

	return h
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func(h *Handler)
		store        func(*mockStore)
		wantCode     int
		wantResp     *response
		wantErrCode  string
		wantLocation string
	}{
		{
			name: "binding error",
			setup: func(h *Handler) {
				h.bind = func(r *http.Request, i any) error {
					return &httpkit.StatusError{Code: http.StatusBadRequest, Response: erresp.Binding(), Err: errors.New("decode failed")}
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
			name: "create error",
			store: func(m *mockStore) {
				m.createFn = func(_ context.Context, _ createParams) (row, error) {
					return row{}, errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.createFn = func(_ context.Context, p createParams) (row, error) {
					return row{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Version: 1}, nil
				}
			},
			wantCode:     http.StatusCreated,
			wantResp:     &response{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
			wantLocation: "/api/lcards/01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		},
	}

	body, _ := json.Marshal(request{Title: "Title", Description: "Description"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup request and request/response recorder
			c, rec := testContext(t, body)

			// Setup handler with mock store
			h := testHandler(t, tt.store, tt.setup)

			err := h.Handle(rec, c)
			if tt.wantErrCode != "" {
				if err == nil {
					t.Fatal("expected returned error")
				}
				if rec.Body.Len() != 0 || rec.Flushed || len(rec.Header()) != 0 {
					t.Fatalf("error response was written: %s", rec.Body.String())
				}
				switch tt.wantCode {
				case http.StatusInternalServerError:
					if httpkit.StatusCode(err) != 0 {
						t.Fatalf("expected untyped store error, got %v", err)
					}
				default:
					typed, ok := errors.AsType[*httpkit.StatusError](err)
					if !ok || typed.Code != tt.wantCode || typed.Response.Code != tt.wantErrCode {
						t.Fatalf("returned error: got %#v, want %d %s", err, tt.wantCode, tt.wantErrCode)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check the status code first
			if rec.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantCode)
			}

			if tt.wantCode == http.StatusCreated && rec.Header().Get("ETag") != `"1"` {
				t.Errorf("ETag: got %q, want quoted 1", rec.Header().Get("ETag"))
			}
			// Check the Location header (set on success only)
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("location: got %q, want %q", got, tt.wantLocation)
			}

			// Check the response body if expected response is provided
			if tt.wantResp != nil {
				var got response
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if got != *tt.wantResp {
					t.Errorf("resp: got %+v, want %+v", got, *tt.wantResp)
				}
			}
		})
	}
}

// Bind errors that carry a 413 or 415 status are returned unwritten so the
// router's error handler reports them, like all other request failures.
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

	body, _ := json.Marshal(request{Title: "Title", Description: "Description"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, rec := testContext(t, body)
			h := testHandler(t, nil, func(h *Handler) {
				h.bind = func(*http.Request, any) error { return tt.err }
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
	c, rec := testContext(t, body)

	var gotParams createParams
	m := &mockStore{
		createFn: func(_ context.Context, p createParams) (row, error) {
			gotParams = p
			return row{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Version: 1}, nil
		},
	}
	h := NewHandler(m)

	if err := h.Handle(rec, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusCreated)
	}

	wantParams := createParams{
		Title:       "Caf\u00E9 Title",       // composed é
		Description: "Caf\u00E9 Description", // composed é
	}
	if gotParams != wantParams {
		t.Errorf("params: got %+q, want %+q", gotParams, wantParams)
	}

	var gotResp response
	if err := json.Unmarshal(rec.Body.Bytes(), &gotResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	wantResp := response{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"}
	if gotResp != wantResp {
		t.Errorf("resp: got %+v, want %+v", gotResp, wantResp)
	}

	wantLocation := "/api/lcards/" + "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	if gotLocation := rec.Header().Get("Location"); gotLocation != wantLocation {
		t.Errorf("location: got %q, want %q", gotLocation, wantLocation)
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

	got := toParams(request{Title: "Title", Description: "Description"})
	want := createParams{Title: "Title", Description: "Description"}
	if got != want {
		t.Errorf("params: got %+v, want %+v", got, want)
	}
}

func TestToResp(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	got := toResp(row{ID: id, Version: 1})
	want := response{ID: id}
	if got != want {
		t.Errorf("resp: got %+v, want %+v", got, want)
	}
}
