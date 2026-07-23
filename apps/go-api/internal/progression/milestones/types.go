// Package milestones — types et constantes pour les milestones (paliers cumulatifs).
//
// Un milestone est un palier cumulatif débloquable une seule fois (ex : 100
// matchs joués, 50 victoires, 30 jours distincts avec accuracy > 0.50). Le
// catalogue vient d'un TOML (config/titles/{title}/milestones/), versionné en
// metadata.duckdb pour les jointures rapides.
//
// Persistance :
//   - `milestone_catalog` dans metadata.duckdb (référentiel chargé du TOML)
//   - `milestone_earned` dans stats.duckdb (paliers débloqués par joueur)
package milestones

import "time"

// NearMissRatio est la distance relative au seuil considérée comme "proche".
// Si valeur courante >= seuil × (1 - NearMissRatio), on émet une notif near-miss.
//
// DP14 (2026-07) : 0.10 → 0.02. « Proche » à 90 % de 10 000 kills = 1 000 kills
// restants (des semaines) — pas une approche. À 98 % (200 kills restants) le
// dénouement est imminent, le nudge a du sens.
const NearMissRatio = 0.02

// CatalogEntry est une définition de milestone dans le référentiel.
//
// Stocké dans `milestone_catalog` (metadata.duckdb). Chargé du TOML au boot
// ou via reload admin. Indépendant des joueurs.
type CatalogEntry struct {
	ID        string  `json:"id"`
	TitleSlug string  `json:"title_slug"`
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
	TitleEN   string  `json:"title_en"`
	TitleFR   string  `json:"title_fr"`
	Icon      string  `json:"icon,omitempty"`
	// Condition est la description TECHNIQUE (formule) du milestone, non localisée
	// (ex : "accuracy >= 0.50 on 30 distinct days"). Référence interne — n'est PLUS
	// servie à l'UI (A9). Non évaluée.
	Condition string `json:"condition,omitempty"`
	// ConditionFR / ConditionEN : description lisible localisée servie à l'UI (A9).
	// Vide si le milestone n'a pas de condition explicite → le front n'affiche rien.
	ConditionFR string `json:"condition_fr,omitempty"`
	ConditionEN string `json:"condition_en,omitempty"`
}

// Earned représente un milestone débloqué par un joueur.
//
// Stocké dans `milestone_earned` (stats.duckdb par joueur). PK composite
// (UserID, TitleSlug, MilestoneID) garantit l'idempotence : un milestone
// ne peut être "earned" qu'une seule fois.
type Earned struct {
	UserID      string    `json:"user_id"`
	TitleSlug   string    `json:"title_slug"`
	MilestoneID string    `json:"milestone_id"`
	EarnedAt    time.Time `json:"earned_at"`
}
