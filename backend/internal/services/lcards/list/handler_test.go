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
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/lcards", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Setup handler with mock store
			h := NewHandler(&mockStore{})
			if tt.store != nil {
				tt.store(h.store.(*mockStore))
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
