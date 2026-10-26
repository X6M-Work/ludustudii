package get

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	getFn func(ctx context.Context, id string) (row, error)
}

func (m *mockStore) Get(ctx context.Context, id string) (row, error) {
	return m.getFn(ctx, id)
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*Handler)
		store       func(*mockStore)
		wantCode    int
		wantResp    *response
		wantErrCode string
	}{
		{
			name: "param error",
			setup: func(h *Handler) {
				h.validateParam = func(id string) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusBadRequest,
			wantErrCode: erresp.CodeParam,
		},
		{
			name: "not found error",
			store: func(m *mockStore) {
				m.getFn = func(_ context.Context, _ string) (row, error) {
					return row{}, domain.ErrNotFound
				}
			},
			wantCode:    http.StatusNotFound,
			wantErrCode: erresp.CodeNotFound,
		},
		{
			name: "get error",
			store: func(m *mockStore) {
				m.getFn = func(_ context.Context, _ string) (row, error) {
					return row{}, errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.getFn = func(_ context.Context, _ string) (row, error) {
					return row{
						ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
						Title:       "Title",
						Description: "Description",
					}, nil
				}
			},
			wantCode: http.StatusOK,
			wantResp: &response{
				ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
				Title:       "Title",
				Description: "Description",
			},
		},
	}

	id := "01a1730e-199d-7f9f-b59d-3bcd6f84066c"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup Echo context and request/response recorder
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/lcards/"+id, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

			// Setup handler with mock store
			m := &mockStore{}
			if tt.store != nil {
				tt.store(m)
			}
			h := NewHandler(m)
			if tt.setup != nil {
				tt.setup(h)
			}

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

func TestValidateParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid uuid v7", id: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		{name: "empty", id: "", wantErr: true},
		{name: "not a uuid", id: "not-a-uuid", wantErr: true},
		{name: "uuid v4", id: "9b2f6f4e-3c1a-4d2b-8e5f-7a6b5c4d3e2f", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateParam(tt.id)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			fe, ok := errors.AsType[*validator.FieldError](err)
			if !ok {
				t.Fatalf("error: got %v, want *validator.FieldError", err)
			}
			if fe.Field != "id" || fe.Code != validator.CodeInvalidUUID7 {
				t.Errorf("field error: got %s/%s, want id/%s", fe.Field, fe.Code, validator.CodeInvalidUUID7)
			}
		})
	}
}

func TestToResp(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	got := toResp(row{ID: id, Title: "Title", Description: "Description"})
	want := response{ID: id, Title: "Title", Description: "Description"}
	if got != want {
		t.Errorf("resp: got %+v, want %+v", got, want)
	}
}
