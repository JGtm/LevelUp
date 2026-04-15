// Package domain — citations.go : types pour les pages Citations et Commendations.
//
// Sprint 13 :
//   GET /api/v1/players/{slug}/pages/citations      → CitationsPageResponse
//   GET /api/v1/players/{slug}/pages/commendations  → CommendationsPageResponse
package domain

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Citations
// ---------------------------------------------------------------------------

// CitationMappingRow est une ligne brute chargée depuis Q34 (metadata.duckdb).
type CitationMappingRow struct {
	NameNorm    string
	NameDisplay string
	MappingType string
	Category    string
	ImagePath   *string
	Description *string
	TierTargets *string
}

// CitationTotalRow est une ligne brute depuis match_citations (Q35).
type CitationTotalRow struct {
	NameNorm string
	Total    int
}

// MedalEarnedRow est une ligne brute depuis shared.medals_earned (Q36a).
type MedalEarnedRow struct {
	MedalID    int64
	TotalCount int
}

// MedalCitationRow est une ligne brute depuis metadata citation_mappings (Q36b).
type MedalCitationRow struct {
	MedalID     int64
	NameDisplay string
	Category    string
	ImagePath   *string
}

// ---------------------------------------------------------------------------
// Types de réponse — Citations
// ---------------------------------------------------------------------------

// CitationItem est une citation enrichie avec son total et sa métadonnée.
type CitationItem struct {
	NameNorm    string  `json:"name_norm"`
	NameDisplay string  `json:"name_display"`
	Category    string  `json:"category"`
	Total       int     `json:"total"`
	ImagePath   *string `json:"image_path,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CitationsPageResponse est la réponse de la page Citations.
type CitationsPageResponse struct {
	Citations  []CitationItem `json:"citations"`
	Categories []string       `json:"categories"`
	TotalCount int            `json:"total_count"`
}

// ---------------------------------------------------------------------------
// Types de réponse — Commendations / Médailles
// ---------------------------------------------------------------------------

// CommendationItem est une médaille avec son total et sa catégorie.
type CommendationItem struct {
	MedalID     int64   `json:"medal_id"`
	MedalName   string  `json:"medal_name"`
	Count       int     `json:"count"`
	Category    string  `json:"category"`
	ImagePath   *string `json:"image_path,omitempty"`
}

// CommendationCategory regroupe les médailles par catégorie.
type CommendationCategory struct {
	Category string             `json:"category"`
	Items    []CommendationItem `json:"items"`
	Total    int                `json:"total"`
}

// CommendationsPageResponse est la réponse de la page Commendations.
type CommendationsPageResponse struct {
	Categories []CommendationCategory `json:"categories"`
	TotalCount int                    `json:"total_count"`
}
