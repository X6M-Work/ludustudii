//go:build unit

package list

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	listFn func(ctx context.Context) ([]row, error)
}

func (m *mockStore) List(ctx context.Context) ([]row, error) {
	return m.listFn(ctx)
}

func testContext(t *testing.T) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/lcards", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func testHandler(t *testing.T, store func(*mockStore)) *Handler {
	t.Helper()

	m := &mockStore{
		listFn: func(ctx context.Context) ([]row, error) {
			t.Fatal("unexpected call to listFn")
			return nil, errors.New("")
		},
	}
	if store != nil {
		store(m)
	}

	return &Handler{store: m}
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		store       func(*mockStore)
		wantCode    int
		wantResp    []response
		wantErrCode string
	}{
		{
			name: "list error",
			store: func(m *mockStore) {
				m.listFn = func(_ context.Context) ([]row, error) {
					return nil, errors.New("")
				}
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: erresp.CodeUnexpected,
		},
		{
			name: "success",
			store: func(m *mockStore) {
				m.listFn = func(_ context.Context) ([]row, error) {
					return []row{
						{
							ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
							Title:       "Title",
							Description: "Description",
						},
					}, nil
				}
			},
			wantCode: http.StatusOK,
			wantResp: []response{
				{
					ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
					Title:       "Title",
					Description: "Description",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup Echo context and request/response recorder
			c, rec := testContext(t)

			// Setup handler with mock store
			h := testHandler(t, tt.store)

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
				var got []response
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if !reflect.DeepEqual(got, tt.wantResp) {
					t.Errorf("resp: got %+v, want %+v", got, tt.wantResp)
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

func TestToResp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []row
		want []response
	}{
		{
			name: "nil",
			data: nil,
			want: []response{},
		},
		{
			name: "empty",
			data: []row{},
			want: []response{},
		},
		{
			name: "preserves order",
			data: []row{
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title 1", Description: "Description 1"},
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066d", Title: "Title 2", Description: "Description 2"},
			},
			want: []response{
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066c", Title: "Title 1", Description: "Description 1"},
				{ID: "01a1730e-199d-7f9f-b59d-3bcd6f84066d", Title: "Title 2", Description: "Description 2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := toResp(tt.data)
			// A nil slice would serialize to null instead of [].
			if got == nil {
				t.Fatal("resp: got nil, want non-nil slice")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resp: got %+v, want %+v", got, tt.want)
			}
		})
	}
}
