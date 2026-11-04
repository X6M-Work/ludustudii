package validator

// ValidateMaxLengthText checks that val is trimmed, in NFC, not empty and at
// most max runes. It returns the first failed check as a *FieldError for field.
func ValidateMaxLengthText(field, val string, max int) error {
	if err := Trimmed(field, val); err != nil {
		return err
	}
	if err := Normalized(field, val); err != nil {
		return err
	}
	if err := Required(field, val); err != nil {
		return err
	}
	if err := MaxLength(field, val, max); err != nil {
		return err
	}
	return nil
}
