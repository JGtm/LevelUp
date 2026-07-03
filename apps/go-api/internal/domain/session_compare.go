// Package domain — session_compare.go : types de RÉSUMÉ de session PARTAGÉS.
//
// NB (G3, 2026-07-03, DEC-1) : la feature « comparaison de sessions » (endpoint
// POST /pages/session-compare + front features/session-compare) a été SUPPRIMÉE.
// Les types ci-dessous SURVIVENT car ils sont désormais consommés par la page
// SESSION-DETAIL (session_page) : SessionCompareEntry (résumé d'une session),
// SessionCompareMetricRow (ligne de métrique), SessionParticipationAxis et
// SessionMatchPoint (agrégats embarqués dans SessionCompareEntry). Les types
// compare-only (Request/Response/MapRow/ModeRow) ont été retirés avec la feature.
package domain

// SessionParticipationAxis est un axe du profil de participation normalisé 0..100.
// Le nom correspond aux constantes narrative.Axis* ("combat", "survival", etc.).
type SessionParticipationAxis struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"` // 0..100
}

// SessionCompareEntry résume une session sélectionnée.
type SessionCompareEntry struct {
	SessionLabel     string   `json:"session_label"`
	StartTime        *string  `json:"start_time"`
	EndTime          *string  `json:"end_time"`
	TotalMatches     int      `json:"total_matches"`
	Wins             int      `json:"wins"`
	Losses           int      `json:"losses"`
	KDA              *float64 `json:"kda"`
	PerformanceScore *float64 `json:"performance_score"`
	// Métriques dérivées (mêmes helpers que compare_metrics) — pour le résumé single ET comparé.
	WinRate       float64 `json:"win_rate"`        // 0..100 (convention winRate())
	KDR           float64 `json:"kdr"`             // total_kills / total_deaths
	KillsPerMatch float64 `json:"kills_per_match"` // total_kills / total_matches
	// Stats du radar de frags — AGRÉGATS DE SESSION (le "compte" de la session) :
	// MaxKillingSpree = max atteint ; TotalHeadshotKills / TotalPerfectKills = totaux.
	// nil si aucun match n'a la stat. Le tooltip du radar affiche ces valeurs brutes.
	MaxKillingSpree    *int    `json:"max_killing_spree,omitempty"`
	TotalHeadshotKills *int    `json:"total_headshot_kills,omitempty"`
	TotalPerfectKills  *int    `json:"total_perfect_kills,omitempty"`
	WithFriends        bool    `json:"with_friends"`
	DominantCategory   *string `json:"dominant_category"`
	// OC/DR moyens sur la session — nil si aucun match avec données dégâts.
	// Ref : PLAN_COMBAT_PROFILE_WIRING.md Phase 3.
	AvgOC *float64 `json:"avg_oc,omitempty"`
	AvgDR *float64 `json:"avg_dr,omitempty"`
	// DmgPerKill / DmgPerDeath : dégâts moyens par frag / par mort sur la session
	// (Σ damage_dealt / Σ kills, Σ damage_taken / Σ deaths). Nil si dénominateur
	// nul. Affichés à côté du rendement/résistance (parité bande Synthesis).
	DmgPerKill  *float64 `json:"dmg_per_kill,omitempty"`
	DmgPerDeath *float64 `json:"dmg_per_death,omitempty"`
	// AvgPaceRatio : engagement absolu moyen (pace_joueur/pace_lobby ; 1.0 = rythme lobby).
	AvgPaceRatio *float64 `json:"avg_pace_ratio,omitempty"`
	// MatchSeries : données par match pour les charts de progression (K/D, cumul, précision).
	MatchSeries []SessionMatchPoint `json:"match_series"`
	// Skill rating (LUSR ou CSR) — dernier match de la session.
	LastSkillRating  *float64 `json:"last_skill_rating,omitempty"`
	SkillRatingType  string   `json:"skill_rating_type,omitempty"`  // "csr" | "lusr" | ""
	SkillRatingDelta *float64 `json:"skill_rating_delta,omitempty"` // last − first
	// MMR moyen sur la session.
	AvgTeamMMR  *float64 `json:"avg_team_mmr,omitempty"`
	AvgEnemyMMR *float64 `json:"avg_enemy_mmr,omitempty"`
	// Durée de vie moyenne sur la session (secondes) — pour la KPI "Durée de vie".
	AvgLifeSeconds *float64 `json:"avg_life_seconds,omitempty"`
	// Précision moyenne sur la session (0..1, ADR 0006) — nil si aucun match avec
	// précision. Le frontend multiplie par 100 pour l'affichage (KPI "Précision").
	AvgAccuracy *float64 `json:"avg_accuracy,omitempty"`
	// Profil de participation 6 axes (Combat/Survival/Support/Score/Objective/Impact), normalisé 0..100.
	Participation []SessionParticipationAxis `json:"participation"`
	// Historique des matchs de la session (ordre chronologique).
	Matches []SessionDetailMatchRow `json:"matches"`
	// Meilleur et pire match par performance score computé.
	BestMatch  *SessionDetailMatchRow `json:"best_match,omitempty"`
	WorstMatch *SessionDetailMatchRow `json:"worst_match,omitempty"`
}

// SessionCompareMetricRow est une ligne de comparaison métrique A vs B.
type SessionCompareMetricRow struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	ValueA string  `json:"value_a"`
	ValueB string  `json:"value_b"`
	Delta  *string `json:"delta"`
	Winner *string `json:"winner"` // "a" | "b" | "tie"
}

// SessionMatchPoint est un point de données par match pour les charts de progression.
// Accuracy est en convention 0..1 (ADR 0006) — le frontend multiplie par 100 pour l'affichage.
type SessionMatchPoint struct {
	Index           int      `json:"index"`                      // 1-based
	KD              float64  `json:"kd"`                         // kills / deaths (deaths=0 → kills)
	Cumulative      int      `json:"cumulative"`                 // solde cumulé W=+1 / L=-1 / autre=0
	Accuracy        *float64 `json:"accuracy"`                   // 0..1, nil si indisponible
	PerfScore       *float64 `json:"perf_score,omitempty"`       // performance_score computé du match
	SkillRating     *float64 `json:"skill_rating,omitempty"`     // LUSR ou CSR après ce match
	EngagementScore *float64 `json:"engagement_score,omitempty"` // résidu brut d'engagement du match
}
