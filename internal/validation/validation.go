package validation

// BatchValidationResult represents results of validation, resulting batch size,
// if validation was passed successfully, and errors, if validation was failed.
type BatchValidationResult struct {
	Ok           bool
	BatchSize    int
	ErrorCode    string
	ErrorMessage string
}
