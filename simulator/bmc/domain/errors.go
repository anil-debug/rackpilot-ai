package domain

import "fmt"

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ErrNotFound     = APIError{Code: "not_found", Message: "resource not found"}
	ErrInvalidID    = APIError{Code: "invalid_id", Message: "invalid or missing component id"}
	ErrInvalidJSON  = APIError{Code: "invalid_json", Message: "request body must be valid JSON"}
	ErrUnknownFault = APIError{Code: "unknown_scenario", Message: "unknown scenario"}
)
