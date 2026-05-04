// Package domain — achievements_response.go : payload HTTP de la page Achievements.
//
// Tags JSON snake_case (convention API). Les noms et descriptions sont servis
// dans les deux langues ; le frontend choisit selon sa locale (même pattern que
// les médailles bilingues).
package domain

import "time"

// AchievementsPageResponse est la réponse complète de GET /pages/achievements.
type AchievementsPageResponse struct {
	Summary      AchievementsSummary `json:"summary"`
	Achievements []AchievementEntry  `json:"achievements"`
}

// AchievementsSummary agrège les métriques globales de la collection.
type AchievementsSummary struct {
	TotalCount       int     `json:"total_count"`
	UnlockedCount    int     `json:"unlocked_count"`
	TotalGamerscore  int     `json:"total_gamerscore"`
	EarnedGamerscore int     `json:"earned_gamerscore"`
	CompletionPct    float64 `json:"completion_pct"`
}

// AchievementEntry représente un achievement, fusion d'une définition et de la
// progression du joueur. Si le joueur n'a pas de ligne player_achievements pour
// cet ID, Unlocked=false et UnlockedAt/Current/Target sont nil.
type AchievementEntry struct {
	AchievementID   string     `json:"achievement_id"`
	NameEN          string     `json:"name_en"`
	NameFR          string     `json:"name_fr"`
	DescriptionEN   string     `json:"description_en"`
	DescriptionFR   string     `json:"description_fr"`
	LockedDescEN    string     `json:"locked_desc_en,omitempty"`
	LockedDescFR    string     `json:"locked_desc_fr,omitempty"`
	Gamerscore      int        `json:"gamerscore"`
	ImageURL        string     `json:"image_url,omitempty"`
	IsSecret        bool       `json:"is_secret"`
	RarityCategory  string     `json:"rarity_category,omitempty"`
	RarityPercent   float64    `json:"rarity_percent,omitempty"`
	Unlocked        bool       `json:"unlocked"`
	UnlockedAt      *time.Time `json:"unlocked_at,omitempty"`
	CurrentProgress *int       `json:"current_progress,omitempty"`
	TargetProgress  *int       `json:"target_progress,omitempty"`
}
