package erresp

type ErrorResponse struct {
	Message string `json:"message"`
}

func New(message string) ErrorResponse {
	return ErrorResponse{
		Message: message,
	}
}
