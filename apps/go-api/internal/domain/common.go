// Package domain — types communs d'erreur et de pagination.
package domain

import (
	"errors"
	"strings"
	"time"
)

// APIError représente une erreur structurée retournée par les endpoints.
type APIError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Retryable bool        `json:"retryable"`
	Details   interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// ErrorMessageForCode rend le message FIXE d'un code d'erreur (snake_case) : le code lu
// avec des espaces ("player_not_found" -> "player not found"). Le texte d'une erreur
// interne ne part jamais au client ; le détail va au journal.
func ErrorMessageForCode(code string) string {
	if code == "" {
		return "error"
	}
	return strings.ReplaceAll(code, "_", " ")
}

// ErrRecordNotFound : une lecture n'a trouvé aucune ligne (donnée absente, pas une panne). Les dépôts la
// rendent (platform/duckdb.NotFound) et la couche métier la reconnaît par errors.Is, jamais par le sentinelle
// du moteur de base ; port.ErrNotFound en est l'alias que consomment les services.
var ErrRecordNotFound = errors.New("record not found")

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

// ErrForbidden crée une erreur 403 standard (autorisation au niveau de la
// RESSOURCE — couche B de l'ADR 0029, distincte du refus d'accès au profil
// rendu par le middleware RequirePlayerOwnership).
func ErrForbidden(msg string) *APIError {
	return &APIError{Code: "forbidden", Message: msg, Retryable: false}
}

// ErrInternal crée une erreur 500 standard.
func ErrInternal(msg string) *APIError {
	return &APIError{Code: "internal_error", Message: msg, Retryable: true}
}

// MediaToolingStatus reflète la disponibilité de l'outillage média (ffmpeg/ffprobe)
// telle que sondée UNE FOIS au boot. Exposée dans /admin/health pour rendre l'info
// observable en prod même quand LEVELUP_LOG_LEVEL=warn masque la ligne INFO
// « media tooling: ffmpeg/ffprobe disponibles » émise au démarrage.
type MediaToolingStatus struct {
	FFmpeg        bool   `json:"ffmpeg"`
	FFprobe       bool   `json:"ffprobe"`
	FFmpegVersion string `json:"ffmpeg_version,omitempty"`
	// H264EncoderMissing : ffmpeg a répondu et aucun encodeur H.264 n'est utilisable (aucun de libx264,
	// h264_mf, libopenh264) : le transcodage vidéo HLS est refusé (media.ErrNoH264Encoder). Faux tant que la
	// détection n'a pas abouti : un doute n'est pas un verdict.
	H264EncoderMissing bool `json:"h264_encoder_missing"`
}

// PublicHealthResponse est la réponse de GET /health, sonde publique : l'état et rien
// d'autre (aucune version, aucun compteur).
type PublicHealthResponse struct {
	Status string `json:"status"`
}

// HealthResponse est la réponse de GET /admin/health : le détail du service.
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
