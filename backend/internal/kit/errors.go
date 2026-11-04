// Package kit defines errors shared by services.
package kit

import "errors"

// Shared storage errors distinguish missing resources from stale versions.
var (
	ErrNotFound        = errors.New("not found")
	ErrVersionConflict = errors.New("version conflict")
)
