// Package domain — medals.go : types pour la page Médailles (catalogue complet du
// titre + compteur obtenu par joueur, 0 = jamais obtenue).
//
//	POST /api/v1/players/{player_slug}/pages/medals → MedalsPageResponse
//
// Contrat title-agnostic : toutes les médailles du titre courant (catalogue
// medal_definitions) sont retournées, chacune regroupée par catégorie/super-section
// résolue via un MedalCategoryResolver (baseline medal_type pour tous titres,
// enrichissement Halo Infinite optionnel). Les clés Category/SuperSection/Difficulty
// sont STABLES (localisation FR/EN faite côté front).
package domain

// MedalCatalogRow est une ligne brute du catalogue medal_definitions résolue
// locale-aware (label/description). DifficultyIndex est l'index de rareté Waypoint
// (0..3 pour Halo Infinite : Normal/Heroic/Legendary/Mythic).
type MedalCatalogRow struct {
	MedalID         int64
	Label           string
	Description     string
	Difficulty      string // "Normal"|"Heroic"|... (HINF) ou valeur brute du titre
	MedalType       string // spree|mode|multikill|proficiency|skill|style|... (source catégorie baseline)
	DifficultyIndex int    // 0..3 (index de rareté ; sert de tri baseline)
	PersonalScore   int
}

// MedalSummaryItem est une médaille du catalogue enrichie de son compteur joueur
// (Count=0 → jamais obtenue), de sa catégorie/super-section résolues et de son image.
// Description est toujours présente (chaîne vide si aucune, jamais null).
type MedalSummaryItem struct {
	MedalID        int64   `json:"medal_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Difficulty     string  `json:"difficulty"`      // libellé brut du titre (ex. "Heroic")
	DifficultyKey  string  `json:"difficulty_key"`  // clé stable (ex. "heroic")
	DifficultyRank int     `json:"difficulty_rank"` // index de rareté (0..3)
	Category       string  `json:"category"`        // clé stable de catégorie
	SuperSection   string  `json:"super_section"`   // clé stable de super-section
	PersonalScore  int     `json:"personal_score"`
	Count          int     `json:"count"` // total obtenu par le joueur (0 = jamais)
	Sort           int     `json:"sort"`  // ordre d'affichage dans la catégorie
	ImageURL       *string `json:"image_url,omitempty"`
	SpriteSheet    string  `json:"sprite_sheet,omitempty"`
	SpriteLeft     int     `json:"sprite_left,omitempty"`
	SpriteTop      int     `json:"sprite_top,omitempty"`
	SpriteWidth    int     `json:"sprite_width,omitempty"`
	SpriteHeight   int     `json:"sprite_height,omitempty"`
}

// MedalCategoryGroup regroupe les médailles d'une catégorie. Earned = nb de
// médailles distinctes obtenues (Count>0), Total = nb de médailles de la catégorie,
// TotalCount = somme des compteurs obtenus.
type MedalCategoryGroup struct {
	SuperSection string             `json:"super_section"`
	Category     string             `json:"category"`
	Items        []MedalSummaryItem `json:"items"`
	Earned       int                `json:"earned"`
	Total        int                `json:"total"`
	TotalCount   int                `json:"total_count"`
}

// MedalsPageResponse est la réponse de la page Médailles. EarnedTotal = nb de
// médailles distinctes obtenues (Count>0), CatalogTotal = nb total de médailles du
// titre, TotalCount = somme de tous les compteurs.
type MedalsPageResponse struct {
	Medals       []MedalSummaryItem   `json:"medals"`
	Categories   []MedalCategoryGroup `json:"categories"`
	EarnedTotal  int                  `json:"earned_total"`
	CatalogTotal int                  `json:"catalog_total"`
	TotalCount   int                  `json:"total_count"`
}

// MedalsPageRequest : corps (optionnel) de POST /pages/medals.
type MedalsPageRequest struct {
	Category string `json:"category,omitempty"` // filtre par catégorie (vide = toutes)
}
