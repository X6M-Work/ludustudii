package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	updateFn func(ctx context.Context, p updateParams) error
}

func (m *mockStore) Update(ctx context.Context, p updateParams) error {
	return m.updateFn(ctx, p)
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
				h.validateParam = func(id string) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusBadRequest,
			wantErrCode: erresp.CodeParam,
		},
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
			name: "not found error",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) error {
					return domain.ErrNotFound
				}
			},
			wantCode:    http.StatusNotFound,
			wantErrCode: erresp.CodeNotFound,
		},
		{
			name: "update error",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) error {
					return errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.updateFn = func(_ context.Context, _ updateParams) error {
					return nil
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

			// Setup Echo context and request/response recorder
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/api/lcards/"+id, bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
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

func TestBind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		want    request
		wantErr bool
	}{
		{
			name: "valid body",
			body: `{"title":"Title","description":"Description"}`,
			want: request{Title: "Title", Description: "Description"},
		},
		{
			name: "missing fields",
			body: `{}`,
			want: request{},
		},
		{
			name:    "malformed json",
			body:    `{"title":`,
			wantErr: true,
		},
		{
			name:    "wrong field type",
			body:    `{"title":1}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := e.NewContext(req, httptest.NewRecorder())

			var got request
			err := bind(c, &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("req: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  request
		want request
	}{
		{
			name: "trims",
			req:  request{Title: " \t \n A title \t \n ", Description: " \n \t A description \n \t "},
			want: request{Title: "A title", Description: "A description"},
		},
		{
			name: "whitespace",
			req:  request{Title: "   ", Description: "\t\n"},
			want: request{},
		},
		{
			name: "trimmed",
			req:  request{Title: "Title", Description: "Description"},
			want: request{Title: "Title", Description: "Description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.req
			sanitize(&got)
			if got != tt.want {
				t.Errorf("req: got %+v, want %+v", got, tt.want)
			}
		})
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
			name: "all valid",
			req:  request{Title: "Title", Description: "Description"},
		},
		{
			name: "all at max length",
			req: request{
				Title:       strings.Repeat("é", domain.TitleMaxLength),
				Description: strings.Repeat("é", domain.DescriptionMaxLength),
			},
		},
		{
			name:     "title missing",
			req:      request{Description: "Description"},
			wantErrs: []fieldErr{{"title", validator.CodeRequired}},
		},
		{
			name:     "description missing",
			req:      request{Title: "Title"},
			wantErrs: []fieldErr{{"description", validator.CodeRequired}},
		},
		{
			name: "all missing",
			req:  request{},
			wantErrs: []fieldErr{
				{"title", validator.CodeRequired},
				{"description", validator.CodeRequired},
			},
		},
		{
			name: "title too long",
			req: request{
				Title:       strings.Repeat("a", domain.TitleMaxLength+1),
				Description: "Description",
			},
			wantErrs: []fieldErr{{"title", validator.CodeMaxLength}},
		},
		{
			name: "description too long",
			req: request{
				Title:       "Title",
				Description: strings.Repeat("a", domain.DescriptionMaxLength+1),
			},
			wantErrs: []fieldErr{{"description", validator.CodeMaxLength}},
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

func TestToParams(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	got := toParams(request{Title: "Title", Description: "Description"}, id)
	want := updateParams{ID: id, Title: "Title", Description: "Description"}
	if got != want {
		t.Errorf("params: got %+v, want %+v", got, want)
	}
}
