// Package domain — diag_csr.go : types pour l'endpoint /_diag/csr-coverage/{slug}.
//
// Phase 9 du plan pipeline CSR. Permet de constater en 1 ligne si le pipeline
// CSR a correctement capturé les données pour un joueur, ou s'il faut lancer
// un backfill (--csr --force).
package domain

// CSRSnapshotsCoverage : résumé player_csr_snapshots pour un joueur.
type CSRSnapshotsCoverage struct {
	Total            int `json:"total"`
	WithAlltimeValue int `json:"with_alltime_value"`
	WithPlacementRem int `json:"with_placement_remaining"`
}

// MSRCSRCoverage : résumé match_skill_rank rating_type='CSR' pour un joueur.
type MSRCSRCoverage struct {
	Total                   int `json:"total"`
	Matured                 int `json:"matured"`
	Placement               int `json:"placement"`
	RankedMatchesInRegistry int `json:"ranked_matches_in_registry"`
	CoverageGap             int `json:"coverage_gap"` // matchs ranked sans row CSR
}

// CSRReferenceControl : contrôle croisé entre la référence des playlists classées
// (rankedplaylists) et la présence du CSR par match. La référence ne DÉCIDE pas du classement d'un
// match (le RankRecap du payload skill décide, décision D-10) : elle sert d'alerte de qualité.
type CSRReferenceControl struct {
	// ReferenceMatchesWithoutCSR : matchs du joueur sur une playlist de la référence sans aucune
	// ligne CSR (CSR non récupéré, ou RankRecap à zéro côté API).
	ReferenceMatchesWithoutCSR int `json:"reference_matches_without_csr"`
	// CSRMatchesOffReference : matchs du joueur portant un CSR sur une playlist absente de la
	// référence (référence à compléter, ou CSR erroné).
	CSRMatchesOffReference int `json:"csr_matches_off_reference"`
	// Alert : l'un des deux compteurs est non nul.
	Alert bool `json:"alert"`
}

// CSRCoverage : payload retourné par GET /_diag/csr-coverage/{slug}.
type CSRCoverage struct {
	PlayerSlug        string               `json:"player_slug"`
	XUID              string               `json:"xuid"`
	Snapshots         CSRSnapshotsCoverage `json:"snapshots"`
	MatchSkillRankCSR MSRCSRCoverage       `json:"match_skill_rank_csr"`
	NeedsBackfill     bool                 `json:"needs_backfill"`
	ReferenceControl  CSRReferenceControl  `json:"reference_control"`
}
