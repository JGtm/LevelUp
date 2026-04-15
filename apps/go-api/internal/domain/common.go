// Package domain — types communs d'erreur et de pagination.
package domain

// APIError représente une erreur structurée retournée par les endpoints.
type APIError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Retryable bool        `json:"retryable"`
	Details   interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// ErrNotFound crée une erreur 404 standard.
func ErrNotFound(resource, id string) *APIError {
	return &APIError{
		Code:      "not_found",
		Message:   resource + " introuvable : " + id,
		Retryable: false,
	}
}

// ErrBadRequest crée une erreur 400 standard.
func ErrBadRequest(msg string) *APIError {
	return &APIError{Code: "bad_request", Message: msg, Retryable: false}
}

// ErrInternal crée une erreur 500 standard.
func ErrInternal(msg string) *APIError {
	return &APIError{Code: "internal_error", Message: msg, Retryable: true}
}

// HealthResponse est la réponse de GET /health.
type HealthResponse struct {
	Status     string `json:"status"`
	MatchCount int    `json:"match_count"`
	DBVersion  string `json:"db_version,omitempty"`
}
