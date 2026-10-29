//go:build unit

package get

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type mockStore struct {
	getFn func(ctx context.Context, id string) (row, error)
}

func (m *mockStore) Get(ctx context.Context, id string) (row, error) {
	return m.getFn(ctx, id)
}

func testContext(t *testing.T, id string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/lcards/"+id, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})
	return c, rec
}

func testHandler(t *testing.T, store func(*mockStore), setup func(*Handler)) *Handler {
	t.Helper()

	m := &mockStore{
		getFn: func(ctx context.Context, id string) (row, error) {
			t.Fatal("unexpected call to getFn")
			return row{}, errors.New("")
		},
	}
	if store != nil {
		store(m)
	}

	h := &Handler{
		store:           m,
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
		setup       func(*Handler)
		store       func(*mockStore)
		wantCode    int
		wantResp    *response
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
			c, rec := testContext(t, id)

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

func TestToResp(t *testing.T) {
	t.Parallel()

	const id = "01a1730e-199d-7f9f-b59d-3bcd6f84066c"
	got := toResp(row{ID: id, Title: "Title", Description: "Description"})
	want := response{ID: id, Title: "Title", Description: "Description"}
	if got != want {
		t.Errorf("resp: got %+v, want %+v", got, want)
	}
}

func TestHandler_Wiring(t *testing.T) {
	t.Parallel()

	c, rec := testContext(t, "01A1730E-199D-7F9F-B59D-3BCD6F84066C")

	var gotID string
	m := &mockStore{
		getFn: func(_ context.Context, id string) (row, error) {
			gotID = id
			return row{
				ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
				Title:       "Title",
				Description: "Description",
			}, nil
		},
	}
	h := NewHandler(m)

	if err := h.Handle(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	wantID := "01a1730e-199d-7f9f-b59d-3bcd6f84066c" // lowercased
	if gotID != wantID {
		t.Errorf("id: got %q, want %q", gotID, wantID)
	}

	var gotResp response
	if err := json.Unmarshal(rec.Body.Bytes(), &gotResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	wantResp := response{
		ID:          "01a1730e-199d-7f9f-b59d-3bcd6f84066c",
		Title:       "Title",
		Description: "Description",
	}
	if gotResp != wantResp {
		t.Errorf("resp: got %+v, want %+v", gotResp, wantResp)
	}
}
