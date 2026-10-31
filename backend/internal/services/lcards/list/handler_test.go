//go:build unit

package list

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	listFn func(ctx context.Context, p listParams) (page, error)
}

func (m *mockStore) List(ctx context.Context, p listParams) (page, error) {
	return m.listFn(ctx, p)
}

func testContext(t *testing.T, target string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func testHandler(t *testing.T, store func(*mockStore), setup func(*Handler)) *Handler {
	t.Helper()

	m := &mockStore{
		listFn: func(ctx context.Context, p listParams) (page, error) {
			t.Fatal("unexpected call to listFn")
			return page{}, errors.New("")
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
			name: "list error",
			store: func(m *mockStore) {
				m.listFn = func(_ context.Context, _ listParams) (page, error) {
					return page{}, errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success without next cursor",
			store: func(m *mockStore) {
				m.listFn = func(_ context.Context, _ listParams) (page, error) {
					return page{Rows: []row{
						{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"},
					}}, nil
				}
			},
			wantCode: http.StatusOK,
			wantResp: &response{
				Items: []item{
					{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"},
				},
			},
		},
		{
			name: "success with next cursor",
			store: func(m *mockStore) {
				m.listFn = func(_ context.Context, _ listParams) (page, error) {
					return page{
						Rows: []row{
							{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"},
						},
						NextCursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
					}, nil
				}
			},
			wantCode: http.StatusOK,
			wantResp: &response{
				Items: []item{
					{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"},
				},
				NextCursor: new("01a1730e-199d-7f9f-b59d-3bcd6f84066c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup Echo context and request/response recorder
			c, rec := testContext(t, "/api/lcards")

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
				if !reflect.DeepEqual(got, *tt.wantResp) {
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

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"

	tests := []struct {
		name       string
		target     string
		wantParams listParams
	}{
		{
			name:       "query params",
			target:     "/api/lcards?limit=5&cursor=" + id,
			wantParams: listParams{Limit: 5, Cursor: id},
		},
		{
			name:       "uppercase cursor",
			target:     "/api/lcards?cursor=" + strings.ToUpper(id),
			wantParams: listParams{Limit: domain.ListDefaultLimit, Cursor: id},
		},
		{
			name:       "no query",
			target:     "/api/lcards",
			wantParams: listParams{Limit: domain.ListDefaultLimit, Cursor: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, rec := testContext(t, tt.target)

			var gotParams listParams
			m := &mockStore{
				listFn: func(_ context.Context, p listParams) (page, error) {
					gotParams = p
					return page{}, nil
				},
			}
			h := NewHandler(m)

			if err := h.Handle(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
			}
			if gotParams != tt.wantParams {
				t.Errorf("params: got %+v, want %+v", gotParams, tt.wantParams)
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
		{name: "empty", req: request{}},
		{name: "valid", req: request{Limit: "5", Cursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"}},
		{name: "limit at min", req: request{Limit: "1"}},
		{name: "limit at max", req: request{Limit: "100"}},
		{
			name:     "limit not an integer",
			req:      request{Limit: "abc"},
			wantErrs: []fieldErr{{"limit", validator.CodeInvalidInteger}},
		},
		{
			name:     "limit not canonical",
			req:      request{Limit: "05"},
			wantErrs: []fieldErr{{"limit", validator.CodeInvalidInteger}},
		},
		{
			name:     "limit below min",
			req:      request{Limit: "0"},
			wantErrs: []fieldErr{{"limit", validator.CodeOutOfRange}},
		},
		{
			name:     "limit above max",
			req:      request{Limit: "101"},
			wantErrs: []fieldErr{{"limit", validator.CodeOutOfRange}},
		},
		{
			name:     "cursor invalid",
			req:      request{Cursor: "not-a-uuid"},
			wantErrs: []fieldErr{{"cursor", validator.CodeInvalidUUID7}},
		},
		{
			name:     "cursor uppercase",
			req:      request{Cursor: "01A1730E-199D-7F9F-B59D-3BCD6F84066C"},
			wantErrs: []fieldErr{{"cursor", validator.CodeInvalidUUID7}},
		},
		{
			name: "both invalid",
			req:  request{Limit: "0", Cursor: "bad"},
			wantErrs: []fieldErr{
				{"limit", validator.CodeOutOfRange},
				{"cursor", validator.CodeInvalidUUID7},
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

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  request
		want request
	}{
		{name: "empty", req: request{}, want: request{}},
		{
			name: "lowercase cursor unchanged",
			req:  request{Limit: "5", Cursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
			want: request{Limit: "5", Cursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		},
		{
			name: "uppercase cursor lowercased",
			req:  request{Cursor: "01A1730E-199D-7F9F-B59D-3BCD6F84066C"},
			want: request{Cursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.req
			sanitize(&got)
			if got != tt.want {
				t.Errorf("request: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToParams(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"

	tests := []struct {
		name string
		req  request
		want listParams
	}{
		{
			name: "default limit",
			req:  request{},
			want: listParams{Limit: domain.ListDefaultLimit},
		},
		{
			name: "explicit limit and cursor",
			req:  request{Limit: "5", Cursor: id},
			want: listParams{Limit: 5, Cursor: id},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toParams(tt.req)
			if got != tt.want {
				t.Errorf("params: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToResp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data page
		want response
	}{
		{
			name: "nil rows",
			data: page{},
			want: response{Items: []item{}},
		},
		{
			name: "empty rows",
			data: page{Rows: []row{}},
			want: response{Items: []item{}},
		},
		{
			name: "preserves order",
			data: page{Rows: []row{
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title 1", Description: "Description 1"},
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066d", Title: "Title 2", Description: "Description 2"},
			}},
			want: response{Items: []item{
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title 1", Description: "Description 1"},
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066d", Title: "Title 2", Description: "Description 2"},
			}},
		},
		{
			name: "next cursor",
			data: page{
				Rows:       []row{{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"}},
				NextCursor: "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
			},
			want: response{
				Items:      []item{{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title", Description: "Description"}},
				NextCursor: new("01a1730e-199d-7f9f-b59d-3bcd6f84066c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toResp(tt.data)
			// A nil slice would serialize to null instead of [].
			if got.Items == nil {
				t.Fatal("items: got nil, want non-nil slice")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resp: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToResp_JSON(t *testing.T) {
	t.Parallel()

	// next_cursor must serialize as null, not be omitted.
	got, err := json.Marshal(toResp(page{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"items":[],"next_cursor":null}`
	if string(got) != want {
		t.Errorf("json: got %v, want %v", string(got), want)
	}
}
