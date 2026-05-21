// Package domain — diag_csr.go : types pour l'endpoint /_diag/csr-coverage/{slug}.
//
// Phase 9 du plan pipeline CSR. Permet de constater en 1 ligne si le pipeline
// CSR a correctement capturé les données pour un joueur, ou s'il faut lancer
// un backfill (--csr --force).
package domain

// CSRSnapshotsCoverage : résumé player_csr_snapshots pour un joueur.
type CSRSnapshotsCoverage struct {
	Total               int `json:"total"`
	WithAlltimeValue    int `json:"with_alltime_value"`
	WithPlacementRem    int `json:"with_placement_remaining"`
}

// MSRCSRCoverage : résumé match_skill_rank rating_type='CSR' pour un joueur.
type MSRCSRCoverage struct {
	Total                  int `json:"total"`
	Matured                int `json:"matured"`
	Placement              int `json:"placement"`
	RankedMatchesInRegistry int `json:"ranked_matches_in_registry"`
	CoverageGap            int `json:"coverage_gap"` // matchs ranked sans row CSR
}

// CSRCoverage : payload retourné par GET /_diag/csr-coverage/{slug}.
type CSRCoverage struct {
	PlayerSlug         string                `json:"player_slug"`
	XUID               string                `json:"xuid"`
	Snapshots          CSRSnapshotsCoverage  `json:"snapshots"`
	MatchSkillRankCSR  MSRCSRCoverage        `json:"match_skill_rank_csr"`
	NeedsBackfill      bool                  `json:"needs_backfill"`
}
