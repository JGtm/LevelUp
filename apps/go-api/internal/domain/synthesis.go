// Package domain — synthesis.go : types dédiés à la page Synthèse.
//
// Sprint 55 D1 : extraction depuis squad.go — Synthèse devient une page autonome,
// distincte d'Escouade au niveau handler, service, types domaine et contrat OpenAPI.
//
// Endpoint cible : POST /api/v1/players/{slug}/pages/synthesis
package domain

import "time"

// ---------------------------------------------------------------------------
// Types de requête (POST body)
// ---------------------------------------------------------------------------

// SynthesisRequest : corps de POST /pages/synthesis.
// Sprint 55 D2 : period et filters réellement appliqués par le service.
type SynthesisRequest struct {
	Period  string             `json:"period,omitempty"`  // "all" | "1w" | "1m" | "1y" | "2y"
	Filters FilterContextInput `json:"filters,omitempty"` // filtres réellement appliqués en D2
}

// ---------------------------------------------------------------------------
// Bloc scope (D3) — écho explicite du scope réellement appliqué
// ---------------------------------------------------------------------------

// SynthesisScope décrit le scope réellement appliqué lors du calcul Synthèse.
// Retourné en tête de SynthesisPageV2Response pour ancrer la crédibilité de la page.
type SynthesisScope struct {
	Period         string    `json:"period"`                    // période effective appliquée
	MatchCount     int       `json:"match_count"`               // matchs dans le scope
	FiltersApplied []string  `json:"filters_applied,omitempty"` // filtres réellement utilisés
	FiltersIgnored []string  `json:"filters_ignored,omitempty"` // filtres déclarés mais ignorés
	Description    string    `json:"description"`               // résumé lisible du scope
	ComputedAt     time.Time `json:"computed_at"`               // instant de calcul
}

// ---------------------------------------------------------------------------
// Bloc overview (D4) — cumuls, moyennes et pics fiables
// ---------------------------------------------------------------------------

// SynthesisOverview est le premier vrai bloc analytique de la page.
// Contient uniquement les métriques fiables depuis la DB locale (pas de simulation).
type SynthesisOverview struct {
	// Volumes cumulés
	TotalMatches int `json:"total_matches"`
	TotalWins    int `json:"total_wins"`
	TotalLosses  int `json:"total_losses"`
	TotalKills   int `json:"total_kills"`
	TotalDeaths  int `json:"total_deaths"`
	TotalAssists int `json:"total_assists"`

	// Moyennes d'efficacité
	AvgKDA       *float64 `json:"avg_kda,omitempty"`
	AvgKills     *float64 `json:"avg_kills,omitempty"`
	AvgDeaths    *float64 `json:"avg_deaths,omitempty"`
	WinRate      float64  `json:"win_rate"`
	AvgPerfScore *float64 `json:"avg_perf_score,omitempty"`

	// Records / pics
	BestKillsMatch   *int     `json:"best_kills_match,omitempty"`
	BestKDAMatch     *float64 `json:"best_kda_match,omitempty"`
	LongestWinStreak int      `json:"longest_win_streak,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse principale — SynthesisPageResponse v2 (Sprint 55)
// ---------------------------------------------------------------------------

// SynthesisPageV2Response est la nouvelle réponse de POST /pages/synthesis.
// Sprint 55 D3/D4 : ajoute scope explicite et overview en tête.
// Conserve solo_kpis/squad_kpis/comparison_metrics/heatmap/top_weeks pour compatibilité.
type SynthesisPageV2Response struct {
	// Bloc 0 — scope (D3)
	Scope SynthesisScope `json:"scope"`

	// Bloc 1 — overview (D4)
	Overview SynthesisOverview `json:"overview"`

	// Blocs existants conservés (compatibilité Sprint 43)
	SoloKPIs          SynthesisKPIs          `json:"solo_kpis"`
	SquadKPIs         SynthesisKPIs          `json:"squad_kpis"`
	ComparisonMetrics []ComparisonMetricItem `json:"comparison_metrics"`
	HeatmapData       []TemporalHeatmapCell  `json:"heatmap_data"`
	TopWeeks          []TopWeekEntry         `json:"top_weeks"`

	// Blocs previews (D5/D6/D7)
	HighlightsPreview SynthesisHighlightsPreview `json:"highlights_preview"`
	RivalriesPreview  SynthesisRivalriesPreview  `json:"rivalries_preview"`
	Breakdowns        SynthesisBreakdowns        `json:"breakdowns"`
}

// ---------------------------------------------------------------------------
// Bloc previews — Highlights (D5)
// ---------------------------------------------------------------------------

// SynthesisMatchHighlight est un match notable (top/pire) dans la Synthèse.
type SynthesisMatchHighlight struct {
	MatchID   string   `json:"match_id"`
	Kills     int      `json:"kills"`
	Deaths    int      `json:"deaths"`
	KDA       *float64 `json:"kda,omitempty"`
	Outcome   int      `json:"outcome"` // 2=WIN, 3=LOSS
	PerfScore *float64 `json:"perf_score,omitempty"`
}

// SynthesisHighlightsPreview contient les matchs remarquables extraits du scope.
type SynthesisHighlightsPreview struct {
	TopByKills    []SynthesisMatchHighlight `json:"top_by_kills"`
	TopByKDA      []SynthesisMatchHighlight `json:"top_by_kda"`
	WorstByDeaths []SynthesisMatchHighlight `json:"worst_by_deaths"`
}

// ---------------------------------------------------------------------------
// Bloc previews — Rivalries (D6)
// ---------------------------------------------------------------------------

// SynthesisEncounterPreview est un joueur fréquemment rencontré dans le scope.
type SynthesisEncounterPreview struct {
	XUID       string   `json:"xuid"`
	Gamertag   string   `json:"gamertag"`
	MatchCount int      `json:"match_count"`
	AsTeammate int      `json:"as_teammate"`
	AsEnemy    int      `json:"as_enemy"`
	AvgKDA     *float64 `json:"avg_kda,omitempty"`
}

// SynthesisRivalriesPreview résume les encounters fréquents sur la période.
type SynthesisRivalriesPreview struct {
	TopTeammates []SynthesisEncounterPreview `json:"top_teammates"`
	TopEnemies   []SynthesisEncounterPreview `json:"top_enemies"`
	Total        int                         `json:"total"`
}

// ---------------------------------------------------------------------------
// Bloc previews — Breakdowns (D7)
// ---------------------------------------------------------------------------

// SynthesisMapEntry est une ligne agrégée par carte dans la Synthèse.
type SynthesisMapEntry struct {
	MapName    string  `json:"map_name"`
	MatchCount int     `json:"match_count"`
	Wins       int     `json:"wins"`
	WinRate    float64 `json:"win_rate"`
}

// SynthesisModeEntry est une ligne agrégée par mode de jeu dans la Synthèse.
type SynthesisModeEntry struct {
	ModeName   string  `json:"mode_name"`
	MatchCount int     `json:"match_count"`
	Wins       int     `json:"wins"`
	WinRate    float64 `json:"win_rate"`
}

// SynthesisBreakdowns contient les breakdowns carte et mode du scope.
type SynthesisBreakdowns struct {
	TopMaps  []SynthesisMapEntry  `json:"top_maps"`
	TopModes []SynthesisModeEntry `json:"top_modes"`
}

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Synthèse (reprises depuis squad.go pour autonomie)
// ---------------------------------------------------------------------------

// SynthesisMatchRowV2 est la ligne brute enrichie chargée lors du calcul Synthèse.
// Identique à SynthesisMatchRow pour la compatibilité, mais dans son propre package.
// Note : SynthesisMatchRow reste dans squad.go pour ne pas casser les queries existantes.
type SynthesisMatchRowV2 = SynthesisMatchRow
