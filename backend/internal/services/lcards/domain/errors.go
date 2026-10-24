package domain

import (
	"errors"
	"fmt"
)

var (
	ErrCardNotFound        = errors.New("card not found")
	ErrTitleRequired       = fmt.Errorf("%q is required", "title")
	ErrTitleTooLong        = fmt.Errorf("%q must be at most %d characters", "title", MaxTitleLength)
	ErrDescriptionRequired = fmt.Errorf("%q is required", "description")
	ErrDescriptionTooLong  = fmt.Errorf("%q must be at most %d characters", "description", MaxDescriptionLength)
)

