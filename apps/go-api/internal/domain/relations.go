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

// ---------------------------------------------------------------------------
// Phase 3a — Moments & Rivalités (sous-endpoint dédié /relations/moments)
// ---------------------------------------------------------------------------

// RelationHeatmapRawRow : ligne brute du heatmap relation × tranche horaire
// (top-N relations par matchs communs). Bucketing en day-part fait côté Go.
type RelationHeatmapRawRow struct {
	XUID     string
	Gamertag string
	Hour     int // 0..23 (UTC canonique)
	Dow      int // 0=dimanche … 6=samedi (UTC canonique)
	Count    int // matchs communs sur cette (heure, jour)
}

// RelationHeatmapCell : une cellule du heatmap agrégé « Quand tu les croises »
// (une relation × une tranche horaire). Intensity = count de matchs communs.
type RelationHeatmapCell struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	Daypart  int    `json:"daypart"` // 0=Nuit … 5=Tard (cf. analysis/relations.Daypart)
	Count    int    `json:"count"`
}

// RelationHeatmapDowCell : variante par JOUR DE SEMAINE du heatmap « Quand tu
// les croises » (une relation × un jour). DayOfWeek : 0=dimanche … 6=samedi (UTC).
type RelationHeatmapDowCell struct {
	XUID      string `json:"xuid"`
	Gamertag  string `json:"gamertag"`
	DayOfWeek int    `json:"day_of_week"`
	Count     int    `json:"count"`
}

// RelationDuelRawRow : ligne brute de la timeline d'un rival (un match commun
// joué en ennemi), ordonnée ancien→récent. Result est title-aware (1=win,
// 2=loss, 0=autre), décidé en SQL via outcomeSQLEq (jamais 2/3 en dur).
type RelationDuelRawRow struct {
	MatchID       string
	StartTime     time.Time
	Result        int // 1=win, 2=loss, 0=non décisif (analysis/relations.Result*)
	KillsOnRival  int
	DeathsByRival int
	Mode          string // pair_name du match (mode), '' si absent
	MapName       string // map (FR si dispo, sinon EN), '' si absent
}

// RelationDuelEntry : un duel exposé dans la frise (DTO JSON).
type RelationDuelEntry struct {
	MatchID       string  `json:"match_id"`
	StartedAt     *string `json:"started_at"` // RFC3339 UTC (nil si start_time absent)
	Outcome       string  `json:"outcome"`    // "win" | "loss" | "other"
	KillsOnRival  int     `json:"kills_on_rival"`
	DeathsByRival int     `json:"deaths_by_rival"`
	Mode          string  `json:"mode"`     // mode du match (pair_name), '' si absent
	MapName       string  `json:"map_name"` // map du match (FR si dispo), '' si absent
}

// RelationRivalry : une carte revanche (bête noire + autres rivaux). Frise des
// duels + taux de victoire glissant + KPIs (récent vs global, série en cours,
// écart de frags cumulé).
type RelationRivalry struct {
	XUID         string `json:"xuid"`
	Gamertag     string `json:"gamertag"`
	EnemyMatches int    `json:"enemy_matches"`

	Duels []RelationDuelEntry `json:"duels"` // ancien→récent

	// RollingWinRate : un point par duel (ancien→récent), aligné sur Duels.
	// nil ⇒ aucun duel décisif dans la fenêtre se terminant à ce point.
	RollingWinRate []*float64 `json:"rolling_win_rate"`
	RollingWindow  int        `json:"rolling_window"`

	RecentWinRate *float64 `json:"recent_win_rate"`
	GlobalWinRate *float64 `json:"global_win_rate"`
	CurrentStreak int      `json:"current_streak"` // >0 victoires, <0 défaites
	FragGap       int      `json:"frag_gap"`       // kills cumulés − morts cumulées
}

// RelationsMomentsResponse : réponse du sous-endpoint « Moments & Rivalités ».
type RelationsMomentsResponse struct {
	Heatmap      []RelationHeatmapCell    `json:"heatmap"`
	HeatmapDow   []RelationHeatmapDowCell `json:"heatmap_dow"` // même top-N, agrégé par jour de semaine
	Rivalries    []RelationRivalry        `json:"rivalries"`
	TopRelations int                      `json:"top_relations"` // N relations dans le heatmap
}
