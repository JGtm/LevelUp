// Package domain — achievements.go : types bruts d'achievements Xbox lus depuis DuckDB.
//
// Les types Row sont les résultats de scan repo (un par table). Le service
// agrège les deux en types réponse (voir achievements_response.go).
package domain

import "time"

// PlayerAchievementRow représente une ligne de player_achievements (stats.duckdb).
// Progression et statut de déverrouillage par joueur (données dynamiques).
type PlayerAchievementRow struct {
	AchievementID   string
	Unlocked        bool
	UnlockedAt      *time.Time
	CurrentProgress *int
	TargetProgress  *int
}

// AchievementDefinitionRow représente une ligne de xbox_achievement_definitions
// (metadata.duckdb). Référentiel bilingue EN/FR statique, partagé entre joueurs.
type AchievementDefinitionRow struct {
	AchievementID  string
	NameEN         string
	NameFR         string
	DescriptionEN  string
	DescriptionFR  string
	LockedDescEN   string
	LockedDescFR   string
	Gamerscore     int
	ImageURL       string
	IsSecret       bool
	RarityCategory string
	RarityPercent  float64
}
