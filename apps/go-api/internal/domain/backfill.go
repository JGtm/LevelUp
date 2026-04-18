// Package domain — backfill.go : types de requête/réponse pour le pipeline backfill.
package domain

// BackfillStartRequest définit le payload de POST /backfill/start.
type BackfillStartRequest struct {
	// PlayerSlug identifie le joueur dans db_profiles.json.
	PlayerSlug string `json:"player_slug"`

	// Scope : données à inclure dans le backfill.
	// Si tous les champs sont false (défaut), AllData est implicitement true.
	Medals            bool `json:"medals"`
	Events            bool `json:"events"`
	Skill             bool `json:"skill"`
	PersonalScores    bool `json:"personal_scores"`
	PerformanceScores bool `json:"performance_scores"`
	Aliases           bool `json:"aliases"`
	Weapons           bool `json:"weapons"`
	LUSR              bool `json:"lusr"`
	AllData           bool `json:"all_data"`

	// Options
	MaxMatches  int  `json:"max_matches"`  // 0 = illimité
	DryRun      bool `json:"dry_run"`      // Liste seulement, n'exécute pas
	ForceRescan bool `json:"force_rescan"` // Ignore les guards « déjà traité »
}

// BackfillStartResponse est le sous-ensemble d'AsyncJobStatus retourné à 202.
// Alias de AsyncJobStatus pour la lisibilité.
type BackfillStartResponse = AsyncJobStatus
