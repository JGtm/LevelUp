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
	LUSR              bool `json:"lusr"`
	// CSR : re-fetch les CSR par-match via GetMatchSkill (RankRecap) pour
	// les matchs ranked déjà en DB sans row CSR. Idempotent : skip ceux qui
	// ont déjà un CSR (sauf si ForceRescan=true → tous re-fetchés).
	CSR                    bool `json:"csr"`
	EngagementScores       bool `json:"engagement_scores"`       // Phase 6 plan engagement
	EngagementCoefficients bool `json:"engagement_coefficients"` // Phase recompute coefs (coef-only, rapide)
	// ComebackBadges : calcule dominance_flag (Domination/Humiliation/Remontada/etc.)
	// pour les matchs sans flag. Inclus dans AllData implicitement.
	ComebackBadges bool `json:"comeback_badges"`
	// Citations : calcule match_citations pour les matchs sans entrée dans la table.
	// Inclus dans AllData implicitement. Le sentinel "_processed" empêche le
	// re-traitement des matchs à 0 delta.
	Citations bool `json:"citations"`
	AllData   bool `json:"all_data"`

	// Options
	MaxMatches  int  `json:"max_matches"`  // 0 = illimité
	DryRun      bool `json:"dry_run"`      // Liste seulement, n'exécute pas
	ForceRescan bool `json:"force_rescan"` // Ignore les guards « déjà traité »
}

// BackfillStartResponse est le sous-ensemble d'AsyncJobStatus retourné à 202.
// Alias de AsyncJobStatus pour la lisibilité.
type BackfillStartResponse = AsyncJobStatus
