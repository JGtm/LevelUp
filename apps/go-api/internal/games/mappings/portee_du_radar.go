package mappings

// portee_du_radar.go — LA RESOLUTION « VARIANTE -> PORTEE DU RADAR », EN UN SEUL ENDROIT.
//
// # POURQUOI UN HELPER (CLAUDE.md regle 6, plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2b)
//
// La table `[radar_range_m]` de regulation.toml se lisait par variante en DEUX copies (l'onglet
// Tactique, `TacticalService.rayonsParMatch` ; le placement des vies de l'Emprise (ex-nuage d'isolement),
// `rayonParMatchDuScope`), et l'ecriture du placement des vies au sync en demandait une
// troisieme. Les trois lecteurs doivent rendre la MEME portee pour la MEME variante : une ligne
// ecrite au sync avec une portee que la lecture ne reconnaitrait pas serait ecartee comme
// perimee. Le garde-rail `archlint/no_local_radar_range_lookup_test.go` interdit toute autre
// resolution.

import "strings"

// PorteeDuRadar rend la portee du radar d'une variante, en metres, lue dans la table
// `[radar_range_m]` (`RegulationSet.RadarRangeMap`), et true si elle est connue.
//
// LA CLE SE NETTOIE ICI : le nom vient de `match_registry.game_variant_name`, donc de ce que
// l'API a envoye, et des variantes y arrivent avec un blanc (cf. `RadarRangeMap`). Une portee
// nulle ou negative n'est pas une portee (le chargeur la refuse deja ; la table peut venir
// d'ailleurs) : (0, false), jamais un rayon qui dirait « tout le monde est isole ».
func PorteeDuRadar(table map[string]int, variante string) (float64, bool) {
	metres, ok := table[strings.TrimSpace(variante)]
	if !ok || metres <= 0 {
		return 0, false
	}
	return float64(metres), true
}
