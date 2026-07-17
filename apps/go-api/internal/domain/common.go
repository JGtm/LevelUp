// Package domain — types communs d'erreur et de pagination.
package domain

import "time"

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

// MediaToolingStatus reflète la disponibilité de l'outillage média (ffmpeg/ffprobe)
// telle que sondée UNE FOIS au boot. Exposée dans /health pour rendre l'info
// observable en prod même quand LEVELUP_LOG_LEVEL=warn masque la ligne INFO
// « media tooling: ffmpeg/ffprobe disponibles » émise au démarrage.
type MediaToolingStatus struct {
	FFmpeg        bool   `json:"ffmpeg"`
	FFprobe       bool   `json:"ffprobe"`
	FFmpegVersion string `json:"ffmpeg_version,omitempty"`
}

// HealthResponse est la réponse de GET /health.
type HealthResponse struct {
	Status     string `json:"status"`
	MatchCount int    `json:"match_count"`
	DBVersion  string `json:"db_version,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
	// Sprint 41 T3 : enrichissement du healthcheck
	PlayerCount int        `json:"player_count"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	Uptime      string     `json:"uptime,omitempty"`
	GoVersion   string     `json:"go_version,omitempty"`
	// Disponibilité de l'outillage média sondée au boot (preuve positive côté /health).
	MediaTooling MediaToolingStatus `json:"media_tooling"`
}
