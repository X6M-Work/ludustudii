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
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	createFn func(ctx context.Context, p createParams) (string, error)
}

func (m *mockStore) Create(ctx context.Context, p createParams) (string, error) {
	return m.createFn(ctx, p)
}

func testContext(t *testing.T, body []byte) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/lcards", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func testHandler(t *testing.T, store func(*mockStore), setup func(*Handler)) *Handler {
	t.Helper()

	m := &mockStore{
		createFn: func(ctx context.Context, p createParams) (string, error) {
			t.Fatal("unexpected call to createFn")
			return "", errors.New("")
		},
	}
	if store != nil {
		store(m)
	}

	h := &Handler{
		store:    m,
		bind:     func(*echo.Context, any) error { return nil },
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
		name        string
		setup       func(h *Handler)
		store       func(*mockStore)
		wantCode    int
		wantResp    *response
		wantErrCode string
	}{
		{
			name: "binding error",
			setup: func(h *Handler) {
				h.bind = func(c *echo.Context, i any) error {
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
			name: "create error",
			store: func(m *mockStore) {
				m.createFn = func(_ context.Context, _ createParams) (string, error) {
					return "", errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.createFn = func(_ context.Context, p createParams) (string, error) {
					return "01a1730e-199d-7f9f-b59d-3bcd6f84066c", nil
				}
			},
			wantCode: http.StatusCreated,
			wantResp: &response{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		},
	}

	body, _ := json.Marshal(request{Title: "Title", Description: "Description"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup Echo context and request/response recorder
			c, rec := testContext(t, body)

			// Setup handler with mock store
			h := testHandler(t, tt.store, tt.setup)

			// Call the handler and check the response
			if err := h.Handle(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check the status code first
			if rec.Code != tt.wantCode {
				t.Fatalf("status: got %d, want %d", rec.Code, tt.wantCode)
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

func TestHandler_Wiring(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(request{
		Title:       "\t Cafe\u0301 Title \n",       // decomposed é = e + U+0301
		Description: "\t Cafe\u0301 Description \n", // decomposed é = e + U+0301
	})
	c, rec := testContext(t, body)

	var gotParams createParams
	m := &mockStore{
		createFn: func(_ context.Context, p createParams) (string, error) {
			gotParams = p
			return "01a1730e-199d-7f9f-b59d-3bcd6f84066c", nil
		},
	}
	h := NewHandler(m)

	if err := h.Handle(c); err != nil {
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
	got := toResp(id)
	want := response{ID: id}
	if got != want {
		t.Errorf("resp: got %+v, want %+v", got, want)
	}
}
