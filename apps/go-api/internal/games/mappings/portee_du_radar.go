package mappings

// portee_du_radar.go — LA RESOLUTION « VARIANTE -> PORTEE DU RADAR », EN UN SEUL ENDROIT.
//
// # POURQUOI UN HELPER (CLAUDE.md regle 6)
//
// La lecture (Tactique, placement des vies de l'Emprise, vies des Series temporelles) et
// l'ecriture du placement des vies au sync doivent rendre la MEME portee pour la MEME variante :
// une ligne ecrite au sync avec une portee que la lecture ne reconnaitrait pas serait ecartee
// comme perimee. [PorteeDuRadar] resout une variante, [PorteesDuRadarParMatch] les matchs d'une
// lecture. Le garde-rail `archlint/no_local_radar_range_lookup_test.go` interdit toute autre
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

// PorteesDuRadarParMatch resout la portee COURANTE du radar de chaque match par sa variante
// (match_id -> game_variant_name), par [PorteeDuRadar]. Un match dont la variante n'a pas de
// portee SORT de la table rendue — il sort donc de l'univers de la lecture, pas seulement de ses
// numerateurs — et se compte dans le second retour. Seul appelant autorise de PorteeDuRadar dans
// une boucle sur des matchs : le garde-rail `archlint/no_local_radar_range_lookup_test.go`
// interdit une resolution par match ecrite ailleurs.
func PorteesDuRadarParMatch(table map[string]int, variantes map[string]string) (map[string]float64, int) {
	out := make(map[string]float64, len(variantes))
	sans := 0
	for matchID, variante := range variantes {
		metres, ok := PorteeDuRadar(table, variante)
		if !ok {
			sans++
			continue
		}
		out[matchID] = metres
	}
	return out, sans
}
