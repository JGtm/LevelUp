// Package duckdb — objective_family_sql.go : les deux fragments SQL communs des lectures de
// match_objective_stats_latest — le CASE qui classe une ligne dans sa famille de mode et la liste
// des colonnes à lire. Ils sont GÉNÉRÉS depuis les tables exportées
// narrative.ObjectiveFamilyActionWeights / ObjectiveFamilyHoldColumns (source unique, aucune liste
// de colonnes parallèle) ; la cohérence clés ⊆ schéma est verrouillée par
// TestObjectiveFamilySQL_WeightKeysSubsetOfSchema.
package duckdb

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/narrative"
)

// objectiveFamilyCaseSQL retourne l'expression CASE de classification d'une ligne
// vers sa famille (blocs mutuellement exclusifs par mode ; split KOTH/Strongholds
// par zone_scoring_ticks > 0 AU NIVEAU LIGNE — décision 3 du plan). Discriminants
// alignés sur domain.ObjectiveRaw.HasCTF/HasZones/…/HasVip.
func objectiveFamilyCaseSQL() string {
	return fmt.Sprintf(`CASE
		WHEN COALESCE(o.zone_scoring_ticks, 0) > 0 THEN '%s'
		WHEN o.zone_captures IS NOT NULL OR o.zone_scoring_ticks IS NOT NULL THEN '%s'
		WHEN o.flag_grabs IS NOT NULL OR o.flag_captures IS NOT NULL THEN '%s'
		WHEN o.skull_grabs IS NOT NULL OR o.skull_scoring_ticks IS NOT NULL THEN '%s'
		WHEN o.power_seeds_deposited IS NOT NULL OR o.power_seed_carriers_killed IS NOT NULL THEN '%s'
		WHEN o.successful_extractions IS NOT NULL OR o.extraction_initiations_completed IS NOT NULL THEN '%s'
		WHEN o.times_selected_as_vip IS NOT NULL OR o.kills_as_vip IS NOT NULL THEN '%s'
		ELSE NULL
	END`,
		narrative.FamilyZonesKOTH, narrative.FamilyZonesStrongholds, narrative.FamilyCTF,
		narrative.FamilyOddball, narrative.FamilyStockpile, narrative.FamilyExtraction,
		narrative.FamilyVIP)
}

// objectiveIndexSelectColumns retourne les colonnes d'action et de durée à sommer,
// dans un ordre DÉTERMINISTE (l'ordre du SELECT et du scan en dépend), dérivées des
// tables narrative (source unique, pas de 6e liste de colonnes).
func objectiveIndexSelectColumns() (actionCols, holdCols []string) {
	seen := map[string]bool{}
	for _, fam := range narrative.AllObjectiveFamilies() {
		cols := make([]string, 0, len(narrative.ObjectiveFamilyActionWeights[fam]))
		for col := range narrative.ObjectiveFamilyActionWeights[fam] {
			cols = append(cols, col)
		}
		sort.Strings(cols)
		for _, c := range cols {
			if !seen[c] {
				seen[c] = true
				actionCols = append(actionCols, c)
			}
		}
	}
	seen = map[string]bool{}
	for _, fam := range narrative.AllObjectiveFamilies() {
		for _, c := range narrative.ObjectiveFamilyHoldColumns[fam] {
			if !seen[c] {
				seen[c] = true
				holdCols = append(holdCols, c)
			}
		}
	}
	return actionCols, holdCols
}
