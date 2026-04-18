// Package domain — leaderboard.go : types pour l'endpoint CSR Leaderboards.
//
// Sprint 54 E : LeaderboardEntry, LeaderboardRequest/Response.
package domain

// LeaderboardEntry est une entrée du classement CSR.
type LeaderboardEntry struct {
	TitleSlug string `json:"title_slug"`
	XUID      string `json:"xuid"`
	Gamertag  string `json:"gamertag"`
	CSR       int    `json:"csr"`
	Playlist  string `json:"playlist,omitempty"`
	Season    string `json:"season,omitempty"`
	IsLocal   bool   `json:"is_local"` // true = données DuckDB locale
	// Rank dans la liste — calculé côté service.
	Rank int `json:"rank"`
}

// LeaderboardRequest est les paramètres de GET .../pages/leaderboard.
type LeaderboardRequest struct {
	Season    string `json:"season"`
	Playlist  string `json:"playlist"`
	TitleSlug string `json:"title_slug"`
}

// LeaderboardResponse est la réponse de GET .../pages/leaderboard.
type LeaderboardResponse struct {
	Entries    []LeaderboardEntry `json:"entries"`
	Season     string             `json:"season"`
	Playlist   string             `json:"playlist"`
	TitleSlug  string             `json:"title_slug"`
	TotalLocal int                `json:"total_local"`
}
