// Package domain — citations.go : types pour les pages Citations et Commendations.
//
// Sprint 13 :
//
//	POST /api/v1/players/{slug}/pages/citations      → CitationsPageResponse
//	POST /api/v1/players/{slug}/pages/commendations  → CommendationsPageResponse
package domain

// Valeurs possibles pour CitationMappingRow.MappingType / CitationFullMapping.MappingType.
// Utilisées pour dispatcher le calcul d'une citation selon sa source de données.
const (
	CitationMappingTypeMedal         = "medal"
	CitationMappingTypeStat          = "stat"
	CitationMappingTypePveStat       = "pve_stat"
	CitationMappingTypeWeaponStat    = "weapon_stat"
	CitationMappingTypeObjectiveStat = "objective_stat"
	CitationMappingTypeAward         = "award"
	CitationMappingTypeCustom        = "custom"
	CitationMappingTypeComposite     = "composite"
)

// CitationCategoryMisc est la catégorie fallback pour les citations sans metadata.
const CitationCategoryMisc = "misc"

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Citations
// ---------------------------------------------------------------------------

// CitationMappingRow est une ligne brute chargée depuis Q34 (metadata.duckdb).
type CitationMappingRow struct {
	NameNorm          string
	NameDisplay       string
	MappingType       string
	Category          string
	ImagePath         *string
	Description       *string
	TierTargets       *string
	CompositeChildren *string // JSON list — nil pour les non-composites
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

// CitationItem est une citation enrichie avec son total, ses paliers et sa progression.
type CitationItem struct {
	NameNorm       string  `json:"name_norm"`
	NameDisplay    string  `json:"name_display"`
	Category       string  `json:"category"`
	Total          int     `json:"total"`
	TierCount      int     `json:"tier_count"`
	EarnedTiers    int     `json:"earned_tiers"`
	NextTierTarget int     `json:"next_tier_target"` // 0 si maîtrisé
	MasteryPct     float64 `json:"mastery_pct"`      // 0..100
	ImageURL       *string `json:"image_url,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// CitationCategoryGroup regroupe les citations d'une catégorie.
type CitationCategoryGroup struct {
	Category  string         `json:"category"`
	Items     []CitationItem `json:"items"`
	Total     int            `json:"total"`
	Completed int            `json:"completed"` // items avec MasteryPct == 100
}

// CitationsPageResponse est la réponse de la page Citations.
type CitationsPageResponse struct {
	Citations           []CitationItem          `json:"citations"`
	CitationsByCategory []CitationCategoryGroup `json:"citations_by_category"`
	Categories          []string                `json:"categories"`
	TotalCount          int                     `json:"total_count"`
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

// ---------------------------------------------------------------------------
// Types de réponse — Commendations NATIVES (Halo 5, totaux à vie) — AXE B
// ---------------------------------------------------------------------------

// NativeCommendationTotal — total à vie d'une commendation NATIVE (dernier progress
// absolu), enrichi nom/icône/catégorie. Distinct de CommendationItem (médaille,
// medal_id int64) : la clé est l'UUID natif de la commendation.
type NativeCommendationTotal struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Total    int     `json:"total"`
	IconURL  *string `json:"icon_url,omitempty"`
	// Progression à vie (paliers) — parité avec MatchNativeCommendation pour réutiliser
	// le même rendu d'anneau côté front (CitationProgressRing). Calculés depuis Total +
	// tier_targets via analysis.ComputeTierProgression. tier_count=0 → front masque
	// l'anneau (commendation sans paliers connus / définition non seedée).
	ProgressPct    float64 `json:"progress_pct"`               // 0..100, 100 si maîtrisé
	TierIndex      int     `json:"tier_index,omitempty"`       // nb de paliers atteints
	TierCount      int     `json:"tier_count,omitempty"`       // nb total de paliers
	NextTierTarget int     `json:"next_tier_target,omitempty"` // seuil du prochain palier (0 si maîtrisé)
	// IsMastered : palier final franchi (état À VIE) — distinct de is_newly_mastered
	// (notion par-match) qui n'a pas de sens sur une vitrine de totaux.
	IsMastered bool `json:"is_mastered,omitempty"`
}

// NativeCommendationCategoryGroup regroupe les commendations natives par catégorie
// (ex. MULTIPLAYER, GAME MODE), items triés par total décroissant.
type NativeCommendationCategoryGroup struct {
	Category string                    `json:"category"`
	Items    []NativeCommendationTotal `json:"items"`
}

// NativeCommendationsTotalsResponse — réponse de la page Totaux des commendations
// natives. TotalCount = nombre de commendations distinctes progressées (PAS la somme
// des totaux absolus, qui n'aurait pas de sens). Vide pour les titres sans
// commendations natives (Infinite).
type NativeCommendationsTotalsResponse struct {
	Categories []NativeCommendationCategoryGroup `json:"categories"`
	TotalCount int                               `json:"total_count"`
}
