// Package domain — citations.go : types pour les pages Citations et Commendations.
//
// Sprint 13 :
//
//	POST /api/v1/players/{slug}/pages/citations      → CitationsPageResponse
//	POST /api/v1/players/{slug}/pages/commendations  → CommendationsPageResponse
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
	MedalID   int64   `json:"medal_id"`
	MedalName string  `json:"medal_name"`
	Count     int     `json:"count"`
	Category  string  `json:"category"`
	ImagePath *string `json:"image_path,omitempty"`
}

// ---------------------------------------------------------------------------
// Types de requête — Citations / Commendations (POST body)
// ---------------------------------------------------------------------------

// CitationsPageRequest : corps de POST /pages/citations.
// Tous les champs sont optionnels (filtre facultatif).
type CitationsPageRequest struct {
	Category string `json:"category,omitempty"` // filtre par catégorie (vide = toutes)
}

// CommendationsPageRequest : corps de POST /pages/commendations.
type CommendationsPageRequest struct {
	Category string `json:"category,omitempty"` // filtre par catégorie (vide = toutes)
}

// ---------------------------------------------------------------------------
// Types moteur citations (computation + stockage par match)
// ---------------------------------------------------------------------------

// CitationMedalMapping : une règle citation → medal_id pour le moteur de calcul.
// Chargée depuis metadata.citation_mappings (Q39).
type CitationMedalMapping struct {
	NameNorm    string
	NameDisplay string
	MedalID     int64
	MappingType string // medal | composite | pve
}

// CitationFullMapping contient tous les champs de citation_mappings pour le moteur complet.
// Chargée depuis metadata.citation_mappings (Q40).
type CitationFullMapping struct {
	NameNorm          string
	NameDisplay       string
	MappingType       string // medal|stat|pve_stat|weapon_stat|award|custom|composite
	Category          string
	MedalID           *int64  // medal_id simple (type medal)
	MedalIDs          *string // CSV de medal_ids (type medal multi)
	StatName          *string // colonne match_participants ou "weapon_kills:<name>"
	AwardName         *string // nom d'award (type award)
	CustomFunction    *string // nom de fonction custom
	CompositeChildren *string // JSON list des enfants (type composite)
	TierTargets       *string // CSV de seuils de tiers
}

// CitationEventRow est une ligne depuis shared.highlight_events filtrée par match.
type CitationEventRow struct {
	EventType string
	TimeMS    int64
	XUID      string
}

// CitationContext regroupe toutes les données d'un match nécessaires au moteur complet.
// Stats contient les colonnes match_participants ET "weapon_kills:<name>" pour weapon_stat.
type CitationContext struct {
	Stats       map[string]float64 // colonnes match_participants + "weapon_kills:<name>"
	Medals      map[int64]int      // medal_id → count
	Awards      map[string]int     // award_name → total
	Events      []CitationEventRow // highlight_events du match (triés par time_ms)
	PlayerXUID  string
	Playlist    string // playlist_name normalisée minuscules
	GameVariant string // game_variant_name normalisée minuscules
	Outcome     int    // 1=TIE, 2=WIN, 3=LOSS, 4=DNF
	IsFirefight bool
}

// CitationMatchDelta : valeur calculée d'une citation pour un match.
// Stockée dans match_citations (citation_name_norm, value).
type CitationMatchDelta struct {
	NameNorm string
	Value    int
}

// CitationMatchViewRow : ligne enrichie pour l'onglet Citations de la vue match.
// Résultat de Q38 (match_citations + citation_mappings).
type CitationMatchViewRow struct {
	NameNorm    string
	NameDisplay string
	Value       int
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
