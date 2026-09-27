package duckdb

// perimetre_liste.go — LA LISTE DES MATCHS D'UNE LECTURE, LIÉE EN UNE SEULE CONSTANTE (lot B du
// plan perf « compaction et périmètre joueur », 2026-09-27, ADR 0036 I2).
//
// Les lectures d'historique complet (Carrière Q26 / Q27, Relations Q28, Tactique MortsParCarte)
// bornent chaque vue `_latest` qu'elles lisent aux matchs de la lecture : DuckDB ne pousse sous
// la fenêtre `QUALIFY … OVER (PARTITION BY match_id …)` qu'un filtre CONSTANT sur `match_id`.
//
// POURQUOI UN PARAMÈTRE LISTE, ET PAS `IN (?, ?, …)`. Mesuré sur la copie compactée (2 threads,
// 512 Mo, kill-feed d'un joueur) : un `IN` de 7 190 paramètres (Nuzzles) coûte plus que la
// fenêtre entière (304-341 ms contre 195-214 sans liste) ; la MÊME liste en un seul paramètre
// `VARCHAR[]` descend aussi sous la fenêtre (272 132 lignes vues au lieu de 412 216) et coûte
// 192-249 ms ; à 1 160 matchs (JGtm), 64-87 ms contre 98-107 en `IN`. Le coût d'un `IN` long
// est la liaison de ses milliers de paramètres, pas le filtre.
//
// UN APPEL PAR VUE : un filtre posé sur une vue ne traverse pas la jointure vers une autre vue
// `_latest` (ADR 0036 I2). Chaque vue lue reçoit son propre prédicat, avec la même liste.

// clauseListeMatchs rend le prédicat qui borne la colonne `col` à la liste `ids`, et son unique
// argument : la liste, liée comme `VARCHAR[]`. Une liste vide ne retient aucun match (le
// prédicat vaut faux, jamais « tous ») ; les lecteurs rendent la main avant de toute façon.
func clauseListeMatchs(col string, ids []string) (string, any) {
	if ids == nil {
		ids = []string{}
	}
	return "list_contains(?::VARCHAR[], " + col + ")", ids
}
