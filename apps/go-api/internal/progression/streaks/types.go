// Package streaks — types et constantes pour le suivi des streaks (séries).
//
// Une streak est une série continue de périodes (jour, semaine) satisfaisant
// une condition simple, propre au joueur. Le seuil est calculé depuis
// PlayerProfile (V1), typiquement la médiane des 100 derniers matchs.
//
// Persistance : table `streak` dans stats.duckdb (par joueur).
package streaks

import "time"

// StreakType identifie le type de streak suivie.
type StreakType string

const (
	// StreakTypeDailyPlay : au moins 1 match joué dans la journée.
	StreakTypeDailyPlay StreakType = "daily_play"
	// StreakTypeDailyPerf : au moins 1 match avec stat principale > seuil personnel.
	StreakTypeDailyPerf StreakType = "daily_perf"
	// StreakTypeWeeklyPlay : au moins 5 matchs dans la semaine ISO.
	StreakTypeWeeklyPlay StreakType = "weekly_play"
	// StreakTypeWeeklyKDAThreshold : KDA moyen hebdomadaire > seuil personnel.
	StreakTypeWeeklyKDAThreshold StreakType = "weekly_kda_threshold"
)

// AllStreakTypes liste tous les types supportés.
func AllStreakTypes() []StreakType {
	return []StreakType{
		StreakTypeDailyPlay, StreakTypeDailyPerf,
		StreakTypeWeeklyPlay, StreakTypeWeeklyKDAThreshold,
	}
}

// StreakStatus indique l'état courant d'une streak.
type StreakStatus string

const (
	// StreakStatusActive : la streak progresse normalement.
	StreakStatusActive StreakStatus = "active"
	// StreakStatusPaused : un shield a été utilisé hier, streak préservée.
	StreakStatusPaused StreakStatus = "paused"
	// StreakStatusBroken : la streak est cassée, mais conservée pour l'historique.
	StreakStatusBroken StreakStatus = "broken"
)

// MaxShieldsPerMonth est le nombre de shields disponibles par mois calendaire.
// Régénère le 1er du mois. Anti-frustration "1 jour raté = tout perdu".
const MaxShieldsPerMonth = 1

// PPMultiplier retourne le multiplicateur PP appliqué aux défis complétés
// pendant une streak active de la longueur donnée. Plafonné à 1.75x à 30j+.
//
// Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §4.3.
func PPMultiplier(length int) float64 {
	switch {
	case length >= 30:
		return 1.75
	case length >= 15:
		return 1.50
	case length >= 8:
		return 1.25
	case length >= 4:
		return 1.10
	default:
		return 1.00
	}
}

// Streak représente une série en cours ou passée pour un joueur.
//
// Une seule streak peut être `active` par (UserID, TitleSlug, Type). Les autres
// statuts (paused, broken) sont conservés pour la timeline et le record.
type Streak struct {
	ID                string       `json:"id"`
	UserID            string       `json:"user_id"`
	TitleSlug         string       `json:"title_slug"`
	Type              StreakType   `json:"type"`
	StartedAt         time.Time    `json:"started_at"`
	CurrentLength     int          `json:"current_length"`
	BestLength        int          `json:"best_length"`
	LastIncrementAt   *time.Time   `json:"last_increment_at,omitempty"`
	Threshold         *float64     `json:"threshold,omitempty"`
	ShieldsUsed       int          `json:"shields_used"`
	ShieldsAvailable  int          `json:"shields_available"`
	Status            StreakStatus `json:"status"`
	BrokenAt          *time.Time   `json:"broken_at,omitempty"`
}
