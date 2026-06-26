// Package domain — relations.go : types de la page Communauté > Relations
// (hub des joueurs récurrents). Aucune logique, uniquement des structs +
// la ligne brute repo (RelationRawRow). La réponse JSON {overview, relations[]}
// est consommée par apps/web/src/features/palmares/PalmaresRelationsPage.tsx.
package domain

import "time"

// RelationRawRow : ligne brute de la requête Relations (Q28). Une ligne par
// joueur récurrent (>= 2 matchs communs). first_seen / last_seen sont des
// time.Time zéro quand match_registry n'a pas de start_time pour ces matchs.
type RelationRawRow struct {
	XUID           string
	Gamertag       string
	TotalMatches   int
	TeammateCount  int
	EnemyCount     int
	TeammateWins   int
	TeammateLosses int
	EnemyWins      int
	EnemyLosses    int
	KillsDealt     int
	DeathsSuffered int
	AvgKDAWith     *float64 // KDA moyen sur les matchs en allié (nil si aucun)
	AvgKDAAgainst  *float64 // KDA moyen sur les matchs en ennemi (nil si aucun)
	FirstSeen      time.Time
	LastSeen       time.Time
}

// RelationBadge : badge résolu pour une relation. Style "tinted" (badges
// narratifs existants, fond teinté) ou "solid" (nouveaux badges, fond plein +
// texte blanc côté front).
type RelationBadge struct {
	LabelKey   string         `json:"label_key"`
	ColorToken string         `json:"color_token"`
	Style      string         `json:"style"` // "tinted" | "solid"
	Detail     map[string]any `json:"detail,omitempty"`
}

// RelationInsight : une relation enrichie (un joueur récurrent), prête à être
// affichée dans le tableau du hub.
type RelationInsight struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`

	TotalMatches int `json:"total_matches"`

	TeammateMatches int      `json:"teammate_matches"`
	TeammateWins    int      `json:"teammate_wins"`
	TeammateWinRate *float64 `json:"teammate_win_rate"`

	EnemyMatches int      `json:"enemy_matches"`
	EnemyWins    int      `json:"enemy_wins"`
	EnemyWinRate *float64 `json:"enemy_win_rate"`

	AvgKDAWith    *float64 `json:"avg_kda_with"`
	AvgKDAAgainst *float64 `json:"avg_kda_against"`

	KillsDealt     int      `json:"kills_dealt"`
	DeathsSuffered int      `json:"deaths_suffered"`
	DuelRatio      *float64 `json:"duel_ratio"`

	FirstSeenAt *string `json:"first_seen_at"`
	LastSeenAt  *string `json:"last_seen_at"`

	Category string          `json:"category"` // "ally" | "enemy" | "mixed"
	IsCore   bool            `json:"is_core"`  // noyau dur (source unique : analysis/relations.IsCore)
	Badges   []RelationBadge `json:"badges"`
}

// RelationRef : référence légère vers une relation (KPI hero binôme/bête noire).
type RelationRef struct {
	Gamertag string   `json:"gamertag"`
	WinRate  *float64 `json:"win_rate"`
	Matches  int      `json:"matches"`
}

// RelationsOverview : KPI agrégés du hub Relations.
type RelationsOverview struct {
	DistinctPlayers int          `json:"distinct_players"`
	AlliesCount     int          `json:"allies_count"`
	RivalsCount     int          `json:"rivals_count"`
	CoreCount       int          `json:"core_count"`
	TopAlly         *RelationRef `json:"top_ally"`
	TopNemesis      *RelationRef `json:"top_nemesis"`
}

// RelationsPageResponse : réponse complète de
// GET /players/{slug}/pages/palmares/relations.
type RelationsPageResponse struct {
	Overview  RelationsOverview `json:"overview"`
	Relations []RelationInsight `json:"relations"`
}
