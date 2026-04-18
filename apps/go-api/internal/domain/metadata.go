// Package domain — metadata.go : types pour les métadonnées de saison et snapshots Waypoint.
//
// Sprint 54 A : season_calendars, csr_season_calendars, waypoint_resource_snapshots.
package domain

import "time"

// SeasonCalendar représente une saison Halo Infinite (CSR ou standard).
type SeasonCalendar struct {
	TitleID     string     `json:"title_id"`
	SeasonID    string     `json:"season_id"`
	Version     string     `json:"version"`
	Name        string     `json:"name"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	ContentHash string     `json:"content_hash"`
	ETag        string     `json:"etag,omitempty"`
	SourceURL   string     `json:"source_url,omitempty"`
}

// CSRSeasonCalendar représente une saison CSR (classée).
type CSRSeasonCalendar struct {
	TitleID     string     `json:"title_id"`
	SeasonID    string     `json:"season_id"`
	Version     string     `json:"version"`
	Name        string     `json:"name"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	ContentHash string     `json:"content_hash"`
	ETag        string     `json:"etag,omitempty"`
	SourceURL   string     `json:"source_url,omitempty"`
}

// WaypointResourceSnapshot est un snapshot versionné d'une ressource Waypoint.
// Stocké dans metadata.duckdb pour audit + anti-régression.
type WaypointResourceSnapshot struct {
	TitleID     string    `json:"title_id"`
	ResourceKey string    `json:"resource_key"` // ex: "season_calendar", "csr_season_calendar"
	Version     string    `json:"version"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentHash string    `json:"content_hash"`
	ETag        string    `json:"etag,omitempty"`
	SourceURL   string    `json:"source_url,omitempty"`
	Payload     string    `json:"payload"` // JSON brut archivé
}

// SeasonSynthetic est une saison synthétique retournée en fallback si metadata.duckdb
// ne contient pas de saison courante.
type SeasonSynthetic struct {
	SeasonID   string    `json:"season_id"`
	Name       string    `json:"name"`
	StartDate  time.Time `json:"start_date"`
	IsFallback bool      `json:"is_fallback"`
}

// CurrentSeasonResult est le résultat de GetCurrentSeason avec fallback.
type CurrentSeasonResult struct {
	Season    *SeasonCalendar
	CSRSeason *CSRSeasonCalendar
	Synthetic *SeasonSynthetic // non nil seulement si fallback
}
