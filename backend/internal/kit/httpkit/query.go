package httpkit

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
)

// BindQueryInt parses the named query parameter as a canonical base-10 int.
// Missing or empty values use defaultValue. Malformed or overflowing values
// return a 400 bind_error, retaining the cause. Range validation is left to callers.
func BindQueryInt(r *http.Request, name string, defaultValue int) (int, error) {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(val)
	if err == nil && strconv.Itoa(n) != val {
		err = fmt.Errorf("not canonical: %q", val)
	}
	if err != nil {
		return 0, &StatusError{
			Code:     http.StatusBadRequest,
			Err:      fmt.Errorf("query parameter %q: %w", name, err),
			Response: erresp.Binding(),
		}
	}
	return n, nil
}
