//go:build research

package killsource

// assist_fil_research_test.go — L INSTRUMENT DE LA LECTURE DE L ASSISTANT AU FIL DES EVENEMENTS
// (lot `assist-film`, 2026-10-09), sur UN film a la fois : pour chaque mort publiee sans kill-event,
// la trame de son dead-state, les messages du fil qu elle porte, et ce que la lecture en publie ;
// puis le lexique appris et sa confrontation aux kill-events du film.
//
// LECTURE SEULE, gardee par deux variables — sautee partout ailleurs, CI comprise :
//
//	KS_FIL_FILM=<repo>/data/cache/film_chunks/0a08d2f2 KS_FIL_CARTE="banished narrows" \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	  -run '^TestAssistantAuFil$' -v -count=1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestAssistantAuFil(t *testing.T) {
	dir := os.Getenv("KS_FIL_FILM")
	if dir == "" {
		t.Skip("KS_FIL_FILM absent : mesure sautee")
	}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	o := DefaultOptions()
	e := carteDuCatalogue(t, os.Getenv("KS_FIL_CARTE"))
	o.Carte = &e
	o.normalize()
	c := &decodeCtx{name: filepath.Base(dir), opts: o}
	if err := c.prepare(context.Background(), src); err != nil {
		t.Fatalf("preparation : %v", err)
	}
	kills := c.run().kills()
	c.attachAssists(kills, c.killEvents)
	sansKillEvent := map[int]bool{}
	for _, k := range kills {
		if !k.Assist.Known {
			sansKillEvent[k.TimeMS] = true
		}
	}
	st := c.attachAssistsDuFil(kills)
	t.Logf("LEXIQUE : %+v", st)
	victimesBot := 0
	for _, r := range c.killEvents.recs {
		if c.roster.isBotIndex(r.fields.victim) {
			victimesBot++
			t.Logf("   kill-event %7d ms  victime %d (%s)  tueur %d (%s)", r.ms, r.fields.victim,
				c.nomA(r.fields.victim, r.ms), r.fields.killer, c.nomA(r.fields.killer, r.ms))
		}
	}
	t.Logf("KILL-EVENTS : %d, dont %d a victime bot", len(c.killEvents.recs), victimesBot)
	for _, k := range kills {
		if !sansKillEvent[k.TimeMS] {
			continue
		}
		t.Logf("%7d ms  %-22s <- %-16s paquet %d:%d  connu %v nom %q indice %d refus %q", k.TimeMS, k.Victim,
			k.Feed.Killer, k.paquet.chunk, k.paquet.pidx, k.Assist.Known, k.Assist.Name, k.Assist.Index,
			k.Assist.Rejected)
		for _, r := range c.fil.parPaquet[[2]int{k.paquet.chunk, k.paquet.pidx}] {
			var dest []int
			for i := range grammar.ParticipantsDuFil {
				if r.evenement.Destine(i) {
					dest = append(dest, i)
				}
			}
			t.Logf("            fil type %4d couple t%d(%s) v%d(%s) -> %v complet %v", r.typ, r.tueur,
				c.nomA(r.tueur, k.TimeMS), r.victime, c.nomA(r.victime, k.TimeMS), dest, r.complet)
		}
	}
}
