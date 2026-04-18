// Package domain — compare.go : types pour l'endpoint Compare joueur vs joueur.
//
// Sprint 54 C : NormalizedPlayerStats, CompareRequest/Response, CompareMetricRow.
package domain

import "fmt"

// NormalizedPlayerStats est une projection normalisée des stats d'un joueur,
// utilisée pour la comparaison multi-titre. Les champs extended permettent
// des stats titre-spécifiques sans modifier la structure de base.
type NormalizedPlayerStats struct {
	TitleSlug      string         `json:"title_slug"`
	XUID           string         `json:"xuid"`
	Gamertag       string         `json:"gamertag"`
	Matches        int            `json:"matches"`
	WinRate        float64        `json:"win_rate"`
	KDA            float64        `json:"kda"`
	KDR            float64        `json:"kdr"`
	KillsPerGame   float64        `json:"kills_per_game"`
	DeathsPerGame  float64        `json:"deaths_per_game"`
	AssistsPerGame float64        `json:"assists_per_game"`
	Accuracy       float64        `json:"accuracy"`
	DamagePerGame  float64        `json:"damage_per_game"`
	CareerRank     int            `json:"career_rank"`
	CSRCurrent     int            `json:"csr_current"`
	CSRBest        int            `json:"csr_best"`
	Extended       map[string]any `json:"extended,omitempty"`
}

// CompareRequest est le body de POST .../pages/compare.
type CompareRequest struct {
	TargetGamertag string             `json:"target_gamertag"`
	Filters        FilterContextInput `json:"filters,omitempty"`
}

// Validate valide les champs de CompareRequest.
func (r CompareRequest) Validate() error {
	if r.TargetGamertag == "" {
		return fmt.Errorf("CompareRequest: target_gamertag requis")
	}
	return r.Filters.Validate()
}

// CompareMetricRow est une ligne de la table de comparaison.
type CompareMetricRow struct {
	Metric  string  `json:"metric"`
	LabelFR string  `json:"label_fr"`
	ValueA  float64 `json:"value_a"`
	ValueB  float64 `json:"value_b"`
	Delta   float64 `json:"delta"`  // value_b - value_a
	Winner  string  `json:"winner"` // "a" | "b" | "tie"
}

// CompareResponse est la réponse de POST .../pages/compare.
type CompareResponse struct {
	PlayerA   NormalizedPlayerStats `json:"player_a"`
	PlayerB   NormalizedPlayerStats `json:"player_b"`
	Metrics   []CompareMetricRow    `json:"metrics"`
	TitleSlug string                `json:"title_slug"`
}
