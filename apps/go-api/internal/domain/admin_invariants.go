// Package domain — admin_invariants.go : payloads du dashboard admin
// « Intégrité des données » (Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md).
//
// Miroir JSON des Violations du package internal/sync/invariants — le domain
// ne dépend pas du package invariants (sens de dépendance : api → invariants,
// api → domain).
package domain

// InvariantViolation est la projection JSON d'une violation d'invariant.
type InvariantViolation struct {
	Key         string   `json:"key"`
	Severity    string   `json:"severity"` // "fail" | "warn"
	Count       int      `json:"count"`
	Sample      []string `json:"sample"`
	Description string   `json:"description"`
}

// PlayerInvariantsReport agrège les violations d'un joueur suivi.
type PlayerInvariantsReport struct {
	PlayerSlug string `json:"player_slug"`
	Gamertag   string `json:"gamertag"`
	XUID       string `json:"xuid"`
	// CheckError non vide = le harnais n'a pas pu vérifier ce joueur (DB
	// inaccessible, table absente…) — affiché comme tel, pas comme « sain ».
	CheckError string               `json:"check_error,omitempty"`
	Violations []InvariantViolation `json:"violations"`
	FailCount  int                  `json:"fail_count"`
	WarnCount  int                  `json:"warn_count"`
}

// AdminInvariantsResponse est la réponse de GET /admin/invariants.
type AdminInvariantsResponse struct {
	TitleSlug   string                   `json:"title_slug"`
	GeneratedAt string                   `json:"generated_at"` // RFC3339
	Reports     []PlayerInvariantsReport `json:"reports"`
}
