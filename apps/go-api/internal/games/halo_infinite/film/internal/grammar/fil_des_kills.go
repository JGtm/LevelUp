package grammar

// fil_des_kills.go — LE FIL DES KILLS DE KILLSOURCE : les evenements du chunk de temps forts qui
// porte le plus de kills (lot 2.7.c1 de la representation intermediaire : la lecture descend de
// `facts/killsource/feed.go`, a l identique).
//
// Le chunk se reconnait a son CONTENU, sans borne de numero : un BTB porte ses temps forts au chunk
// 62. C est l une des deux regles du depot ; la cuisson designe ce chunk par le type de son
// manifeste ([ScanDeaths]).

import (
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// FilDesKills rend les evenements du chunk de `f` qui porte le plus de kills, lus sous la version
// majeure `version` du film ([ParseHighlightEvents]) ; nil quand aucun chunk n en porte. Un chunk qui
// ne se lit pas comme des temps forts (un chunk de replication) est ecarte : ce n est pas une
// erreur.
func FilDesKills(f *source.Film, version int) []highlightevent.HighlightEvent {
	var meilleurs []highlightevent.HighlightEvent
	plus := 0
	for pos := range f.NumChunks() {
		evs, err := ParseHighlightEvents(f.Chunk(pos), version)
		if err != nil {
			continue
		}
		if n := compterLesKills(evs); n > plus {
			plus, meilleurs = n, evs
		}
	}
	return meilleurs
}

// compterLesKills rend le nombre d evenements `kill` de `evs`.
func compterLesKills(evs []highlightevent.HighlightEvent) int {
	n := 0
	for _, e := range evs {
		if e.EventType == highlightevent.EventTypeKill {
			n++
		}
	}
	return n
}
