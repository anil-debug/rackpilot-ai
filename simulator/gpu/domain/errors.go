package domain

import "fmt"

// APIError is returned to HTTP clients in a consistent envelope.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ErrGPUNotFound  = APIError{Code: "gpu_not_found", Message: "GPU not found"}
	ErrInvalidGPUID = APIError{Code: "invalid_gpu_id", Message: "invalid or missing gpu_id"}
	ErrInvalidJSON  = APIError{Code: "invalid_json", Message: "request body must be valid JSON"}
)
