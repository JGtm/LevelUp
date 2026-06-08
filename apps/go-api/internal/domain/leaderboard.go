// Package domain — leaderboard.go : types pour l'endpoint CSR Leaderboards.
//
// Sprint 54 E : LeaderboardEntry, LeaderboardRequest/Response.
package domain

import "time"

// LeaderboardEntry est une entrée du classement.
//
// Le classement est multi-catégories (cf. LeaderboardCategory). Pour la
// catégorie CSR mondial, les champs CSR/Tier/SubTier sont renseignés. Pour les
// catégories de stats (kills, kda…), Value/ValueFormatted/Unit/MatchesPlayed le
// sont à la place. Tier/SubTier restent disponibles pour le badge de rang.
type LeaderboardEntry struct {
	TitleSlug string `json:"title_slug"`
	XUID      string `json:"xuid"`
	Gamertag  string `json:"gamertag"`
	CSR       int    `json:"-"`
	CSRValue  int    `json:"csr_value"`
	Tier      string `json:"tier"`
	SubTier   int    `json:"sub_tier"`
	Playlist  string `json:"playlist,omitempty"`
	Season    string `json:"season,omitempty"`
	IsLocal   bool   `json:"is_local"` // true = joueur local (DuckDB)
	// Rank dans la liste — rang mondial scrapé (CSR) ou calculé (stats).
	Rank int `json:"rank"`

	// Catégorie générique (stats non-CSR). Vides pour la catégorie CSR.
	Category       string  `json:"category,omitempty"`
	Value          float64 `json:"value,omitempty"`           // valeur brute triable
	ValueFormatted string  `json:"value_formatted,omitempty"` // valeur formatée pour l'UI
	Unit           string  `json:"unit,omitempty"`            // "%", "" …
	MatchesPlayed  int     `json:"matches_played,omitempty"`

	// FetchedAt : horodatage du scraping (interne, non sérialisé). Persisté en
	// colonne fetched_at de world_csr_leaderboard_snapshots.
	FetchedAt time.Time `json:"-"`
}

// LeaderboardCategory énumère les classements disponibles.
type LeaderboardCategory string

const (
	// LeaderboardCSRWorld : classement CSR mondial (snapshot Halo Waypoint).
	LeaderboardCSRWorld LeaderboardCategory = "csr-world"
	// Catégories de stats agrégées depuis shared.match_participants.
	LeaderboardKills         LeaderboardCategory = "kills"
	LeaderboardDeaths        LeaderboardCategory = "deaths"
	LeaderboardAssists       LeaderboardCategory = "assists"
	LeaderboardKillsPerGame  LeaderboardCategory = "kills_per_game"
	LeaderboardKDA           LeaderboardCategory = "kda"
	LeaderboardKDR           LeaderboardCategory = "kdr"
	LeaderboardAccuracy      LeaderboardCategory = "accuracy"
	LeaderboardDamage        LeaderboardCategory = "damage"
	LeaderboardDamagePerGame LeaderboardCategory = "damage_per_game"
)

// LeaderboardRequest est les paramètres de GET .../pages/leaderboard.
type LeaderboardRequest struct {
	Category  string `json:"category"`
	Season    string `json:"season"`
	Playlist  string `json:"playlist"`
	TitleSlug string `json:"title_slug"`
	Limit     int    `json:"limit"`
}

// LeaderboardResponse est la réponse de GET .../pages/leaderboard.
type LeaderboardResponse struct {
	Entries    []LeaderboardEntry `json:"entries"`
	Category   string             `json:"category"`
	Season     string             `json:"season_id"`
	Playlist   string             `json:"playlist_id"`
	TitleSlug  string             `json:"title_slug"`
	TotalLocal int                `json:"total"`
}
