// Package domain — leaderboard.go : types pour l'endpoint CSR Leaderboards.
//
// Sprint 54 E : LeaderboardEntry, LeaderboardRequest/Response.
package domain

import "time"

// WorldPlayerRef identifie un joueur du classement mondial : son gamertag et, quand
// il est connu (scrapé du snapshot Waypoint), son xuid. XUID vide = à résoudre via
// PeopleHub (lignes de snapshot antérieures à la persistance du xuid, cf. B1).
type WorldPlayerRef struct {
	Gamertag string
	XUID     string
}

// WorldPlaylistRef identifie une playlist classée ACTIVE découverte sur la page
// classement Waypoint (menu déroulant `__NEXT_DATA__.playlists`) : asset id + nom
// affiché. Source directe autoritative des playlists actives (le manifest de build
// renvoie un PlaylistLinks vide) — cf. A1/A2.
type WorldPlaylistRef struct {
	AssetID     string
	DisplayName string
}

// WorldServiceRecord = agrégat CoreStats du service record matchmade d'un joueur,
// filtré par (saison[, playlist]). Une seule requête donne la saison×playlist
// complète (B2 — remplace l'agrégation par-match du classement mondial). Champs bruts
// (sommes natives du jeu) ; la dérivation en WorldPlayerSeasonStats est faite par
// analysis.WorldStatsFromServiceRecord.
type WorldServiceRecord struct {
	MatchesCompleted         int
	Wins, Losses             int
	TimePlayedSec            int64
	Kills, Deaths, Assists   int64
	ShotsFired, ShotsHit     float64
	DamageDealt, DamageTaken float64
	MedalCount               int64
}

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

	// Enrichissement stats mondiales (Phase B — leaderboard enrichi). Pointeurs :
	// nil = joueur non enrichi (pas de ligne world_player_season_stats). Compteurs
	// bruts + ratios dérivés à la lecture + comparaison inter-saison + delta rang.
	MatchCount *int `json:"match_count,omitempty"`
	// CumulativeMatchCount : total des matchs du joueur sur cette playlist cumulé sur
	// toutes les saisons JUSQU'À celle affichée incluse (playlist vide = toutes). Sert
	// à la colonne "Matchs" (≠ MatchCount qui est la seule saison affichée).
	CumulativeMatchCount *int   `json:"cumulative_match_count,omitempty"`
	WinCount             *int   `json:"win_count,omitempty"`
	LossCount            *int   `json:"loss_count,omitempty"`
	TieCount             *int   `json:"tie_count,omitempty"`
	DnfCount             *int   `json:"dnf_count,omitempty"`
	Kills                *int64 `json:"kills,omitempty"`
	Deaths               *int64 `json:"deaths,omitempty"`
	Assists              *int64 `json:"assists,omitempty"`
	PlaytimeSec          *int64 `json:"playtime_seconds,omitempty"`
	MedalCount           *int64 `json:"medal_count,omitempty"`
	// Valeurs natives du jeu, ACCUMULÉES (sommées) — données brutes, aucune dérivation.
	KDA          *float64 `json:"kda,omitempty"`          // somme du KDA natif Halo
	Accuracy     *float64 `json:"accuracy,omitempty"`     // somme de l'Accuracy native (%)
	DamageDealt  *int64   `json:"damage_dealt,omitempty"` // somme des dégâts infligés
	DamageTaken  *int64   `json:"damage_taken,omitempty"` // somme des dégâts subis
	WinRate      *float64 `json:"win_rate,omitempty"`
	KillsPerMin  *float64 `json:"kills_per_min,omitempty"`
	PrevSeasonID *string  `json:"prev_season_id,omitempty"`
	PrevWinRate  *float64 `json:"prev_win_rate,omitempty"`
	PrevKDA      *float64 `json:"prev_kda,omitempty"`
	KDATrend     *string  `json:"kda_trend,omitempty"`      // "up"|"down"|"stable"
	WinRateTrend *string  `json:"win_rate_trend,omitempty"` // idem
	RankDelta    *int     `json:"rank_delta,omitempty"`     // rang saison N vs N-1 (snapshots)

	// FetchedAt : horodatage du scraping (interne, non sérialisé). Persisté en
	// colonne fetched_at de world_csr_leaderboard_snapshots.
	FetchedAt time.Time `json:"-"`
}

// WorldPlayerSeasonStats agrège les stats d'un joueur du classement mondial sur
// une saison CSR x playlist. Compteurs BRUTS (table world_player_season_stats) ;
// les ratios sont DÉRIVÉS à la lecture (nil si dénominateur nul), jamais stockés.
// Attribution (Phase A) : SeasonID via MatchInfo.SeasonId, PlaylistID via
// MatchInfo.Playlist.AssetId. Cf. PLAN_WORLD_LEADERBOARD_ENRICHED.md.
type WorldPlayerSeasonStats struct {
	TitleSlug  string `json:"title_slug"`
	Gamertag   string `json:"gamertag"`
	SeasonID   string `json:"season_id"`
	PlaylistID string `json:"playlist_id"` // "" = agrégat toutes playlists

	// Compteurs bruts.
	MatchCount  int   `json:"match_count"`
	WinCount    int   `json:"win_count"`
	LossCount   int   `json:"loss_count"`
	TieCount    int   `json:"tie_count"`
	DnfCount    int   `json:"dnf_count"`
	Kills       int64 `json:"kills"`
	Deaths      int64 `json:"deaths"`
	Assists     int64 `json:"assists"`
	PlaytimeSec int64 `json:"playtime_seconds"`
	MedalCount  int64 `json:"medal_count"`

	// Valeurs natives du jeu (CoreStats) ACCUMULÉES (sommées), AUCUNE dérivation à
	// l'agrégation — données brutes demandées. KDA = somme du KDA natif Halo
	// (K + A/3 − D par match) ; Accuracy = somme des % natifs ; dégâts = sommes.
	KDA         float64 `json:"kda"`
	Accuracy    float64 `json:"accuracy"`
	DamageDealt int64   `json:"damage_dealt"`
	DamageTaken int64   `json:"damage_taken"`

	// Ratios dérivés à la LECTURE seulement (nil si dénominateur nul) — pas à l'agrégation.
	WinRate     *float64 `json:"win_rate,omitempty"`
	KillsPerMin *float64 `json:"kills_per_min,omitempty"`

	// Comparaison inter-saison (nil = pas de saison précédente avec cette playlist).
	PrevSeasonID *string  `json:"prev_season_id,omitempty"`
	PrevWinRate  *float64 `json:"prev_win_rate,omitempty"`
	PrevKDA      *float64 `json:"prev_kda,omitempty"`
	KDATrend     *string  `json:"kda_trend,omitempty"`
	WinRateTrend *string  `json:"win_rate_trend,omitempty"`
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

// LeaderboardCatalogRef est une option de sélecteur (saison ou playlist) du
// classement CSR mondial. ID est la valeur technique passée en query param ;
// DisplayName est un libellé best-effort (le front peut le localiser).
type LeaderboardCatalogRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	// Enriched : la saison a des stats détaillées (world_player_season_stats), pas
	// seulement un classement CSR scrappé. false = saison archivée affichée en
	// classement seul (le front badge « stats détaillées indisponibles »). Pertinent
	// pour les saisons uniquement ; toujours false pour les playlists.
	Enriched bool `json:"enriched"`
}

// LeaderboardCatalog liste les saisons et playlists pour lesquelles des snapshots
// CSR mondiaux existent réellement en base. Alimente les sélecteurs dynamiques de
// la page Classement (remplace les listes codées en dur v1).
type LeaderboardCatalog struct {
	Seasons   []LeaderboardCatalogRef `json:"seasons"`
	Playlists []LeaderboardCatalogRef `json:"playlists"`
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
