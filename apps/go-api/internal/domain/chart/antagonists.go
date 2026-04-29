// Package chart — types pour les graphiques antagonistes (Explorer).
//
// Port Go des fonctions de src/visualization/ (antagonists.py) :
//   - plot_antagonist_bars    → AntagonistBarChartData
//   - plot_antagonist_duels   → DuelChartData
//   - plot_match_impact_timeline → ImpactTimelineData
//   - plot_team_dominance     → DominanceChartData
//
// Deprecated: P8.9 (revue 2026-04-29) — utiliser `domain.ChartSeries[T]`
// pour les nouvelles features. Migration des consommateurs Explorer prévue
// en backlog post-P8.
package chart

// ---------------------------------------------------------------------------
// Antagonistes (adversaires / coéquipiers fréquents)
// ---------------------------------------------------------------------------

// AntagonistEntry représente un joueur dans le classement antagoniste.
type AntagonistEntry struct {
	XUID     string  `json:"xuid"`
	Gamertag string  `json:"gamertag"`
	Kills    int     `json:"kills"`   // fois où j'ai tué ce joueur
	Deaths   int     `json:"deaths"`  // fois où ce joueur m'a tué
	Balance  int     `json:"balance"` // kills - deaths
	KDRatio  float64 `json:"kd_ratio"`
}

// AntagonistBarChartData contient les données pour le graphique en barres antagoniste.
type AntagonistBarChartData struct {
	TopKilledByMe []AntagonistEntry `json:"top_killed_by_me"`
	TopKilledMe   []AntagonistEntry `json:"top_killed_me"`
	TopN          int               `json:"top_n"`
}

// ---------------------------------------------------------------------------
// Duels (historique 1-vs-1)
// ---------------------------------------------------------------------------

// DuelMatchEntry : résultat d'un duel sur un match précis.
type DuelMatchEntry struct {
	MatchID    string `json:"match_id"`
	KilledMe   int    `json:"killed_me"`
	IKilled    int    `json:"i_killed"`
	WereAllies bool   `json:"were_allies"`
}

// DuelEntry : résumé d'un duel répété contre un même joueur.
type DuelEntry struct {
	XUID        string           `json:"xuid"`
	Gamertag    string           `json:"gamertag"`
	TotalKills  int              `json:"total_kills"`
	TotalDeaths int              `json:"total_deaths"`
	Balance     int              `json:"balance"`
	Matches     []DuelMatchEntry `json:"matches"`
}

// DuelChartData encapsule tous les duels calculés.
type DuelChartData struct {
	Duels []DuelEntry `json:"duels"`
}

// ---------------------------------------------------------------------------
// Impact timeline (kills/deaths par minute d'un match)
// ---------------------------------------------------------------------------

// TimelinePoint représente un point de la timeline d'impact.
type TimelinePoint struct {
	MinuteMS int `json:"minute_ms"` // début de la fenêtre en ms
	Kills    int `json:"kills"`
	Deaths   int `json:"deaths"`
	Balance  int `json:"balance"`
}

// ImpactTimelineData contient la timeline d'un match.
type ImpactTimelineData struct {
	MatchID  string          `json:"match_id"`
	Timeline []TimelinePoint `json:"timeline"`
}

// ---------------------------------------------------------------------------
// Dominance d'équipe
// ---------------------------------------------------------------------------

// TeamSnapshot capture l'état d'une équipe à un instant.
type TeamSnapshot struct {
	TimeMS int `json:"time_ms"`
	TeamID int `json:"team_id"`
	Score  int `json:"score"`
	LeadBy int `json:"lead_by"` // score_team - score_opponent
}

// DominanceChartData contient la série temporelle de dominance.
type DominanceChartData struct {
	MatchID   string         `json:"match_id"`
	Snapshots []TeamSnapshot `json:"snapshots"`
}
