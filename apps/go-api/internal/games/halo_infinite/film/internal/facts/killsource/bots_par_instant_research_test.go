//go:build research

package killsource

// bots_par_instant_research_test.go — L INSTRUMENT DU POINT 18 (2026-10-08) : POURQUOI DES MORTS
// DE BOT RESTENT SANS LIGNE, sur UN film a la fois.
//
// Il met cote a cote, pour chaque indice de replication tenu par un bot :
//
//	BOT_METADATA   les bots que le film declare sur cet indice, avec leurs intervalles de
//	               declaration (lot M2.1) ;
//	roster         le nom que le roster donne a l indice, UN pour tout le film ;
//	feed           les kills du kill-feed dont la victime est un bot (lus au kill-event 85, ou
//	               orphelins, ou recolles), et ce que l appariement en a fait ;
//	dead-states    les candidats du balayage qui portent cet indice en victime ou en tueur, et
//	               s ils ont servi.
//
// LECTURE SEULE, garde par deux variables — saute partout ailleurs, CI comprise :
//
//	KS_BOTS_FILM=<repo>/data/cache/film_chunks/0a08d2f2 KS_BOTS_CARTE="banished narrows" \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	  -run '^TestBotsParInstant$' -v -count=1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

const (
	ksBotsFilmEnv  = "KS_BOTS_FILM"
	ksBotsCarteEnv = "KS_BOTS_CARTE"
)

func TestBotsParInstant(t *testing.T) {
	dir := os.Getenv(ksBotsFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", ksBotsFilmEnv)
	}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	o := DefaultOptions()
	e := carteDuCatalogue(t, os.Getenv(ksBotsCarteEnv))
	o.Carte = &e
	o.normalize()
	c := &decodeCtx{name: filepath.Base(dir), opts: o}
	if err := c.prepare(context.Background(), src); err != nil {
		t.Fatalf("preparation : %v", err)
	}
	p := c.run()
	t.Logf("FILM %s : %d lignes publiees, %d candidats a indice de bot non servis", c.name,
		len(p.byTime), p.unexpBot)
	journaliserBots(t, c)
	journaliserIndicesDeBot(t, c)
	journaliserKillsDeBot(t, c, p)
	journaliserMortsParUnBot(t, c)
	journaliserCandidatsDeBot(t, c, p)
	journaliserKillEventsDeBot(t, c)
	journaliserFeed(t, c)
}

// journaliserFeed : les instants du kill-feed, tels que le film les ecrit (un instant porte un
// kill, une mort, ou les deux).
func journaliserFeed(t *testing.T, c *decodeCtx) {
	t.Helper()
	t.Logf("KILL-FEED : %d instants", len(c.feed.events))
	for _, e := range c.feed.events {
		t.Logf("   instant %7d ms  tueur %-16q victime %-16q", e.timeMS, e.killer, e.victim)
	}
}

// journaliserMortsParUnBot : les morts du feed sans tueur humain, et leur appariement.
func journaliserMortsParUnBot(t *testing.T, c *decodeCtx) {
	t.Helper()
	t.Logf("MORTS DU FEED SANS TUEUR HUMAIN : %d", len(c.feed.orphD))
	for _, m := range c.resolveBotKillerDeaths(nil, nil) {
		t.Logf("   %7d ms  victime %-16s", m.event.timeMS, m.event.victim)
	}
}

// journaliserKillEventsDeBot : les kill-events (genre 85) dont un cote est un indice de bot.
func journaliserKillEventsDeBot(t *testing.T, c *decodeCtx) {
	t.Helper()
	n := 0
	for _, r := range c.killEvents.recs {
		if c.roster.isBotIndex(r.fields.victim) || c.roster.isBotIndex(r.fields.killer) {
			n++
			t.Logf("   kill-event %7d ms  tueur %2d  victime %2d", r.ms, r.fields.killer, r.fields.victim)
		}
	}
	t.Logf("KILL-EVENTS : %d au total, %d a indice de bot", len(c.killEvents.recs), n)
}

// journaliserBots : les bots que BOT_METADATA declare, avec leurs intervalles (en ms du film).
func journaliserBots(t *testing.T, c *decodeCtx) {
	t.Helper()
	t.Logf("BOT_METADATA : %d bots, %d paquets, %d incomplets", len(c.roster.bots.Bots),
		c.roster.bots.NPkt, c.roster.bots.Incomplets)
	for _, b := range c.roster.bots.Bots {
		for _, d := range b.declarations {
			fin := -1
			if d.ToUS != 0 {
				fin = c.film.msDe(d.ToUS)
			}
			t.Logf("   indice %2d  bid(%d)  %-20s  declare [%d ms, %d ms)", b.Slot, b.BotID, b.Name,
				c.film.msDe(d.FromUS), fin)
		}
	}
}

// journaliserIndicesDeBot : le nom que le roster donne a chaque indice tenu par un bot.
func journaliserIndicesDeBot(t *testing.T, c *decodeCtx) {
	t.Helper()
	t.Logf("ROSTER (un nom par indice pour tout le film) : %d successions", c.roster.botsSuccedes)
	for i := 0; i < c.roster.nPlay; i++ {
		if c.roster.isBotIndex(i) {
			t.Logf("   indice %2d -> %s", i, c.roster.nameOf(i))
		}
	}
}

// journaliserKillsDeBot : les kills du feed sans victime humaine, et leur appariement.
func journaliserKillsDeBot(t *testing.T, c *decodeCtx, p *pass) {
	t.Helper()
	t.Logf("KILLS DU FEED SANS VICTIME HUMAINE : %d lus, %d orphelins, %d recolles",
		len(c.feed.botLus), len(c.feed.orphK), len(c.feed.fab))
	for _, m := range c.resolveBotDeaths() {
		etat := "NON APPARIE"
		if m.found {
			etat = "apparie"
			if p.botUsed[[3]int{m.cand.chunk, m.cand.pidx, m.cand.bit}] {
				etat = "apparie, publie"
			}
		}
		t.Logf("   %7d ms  tueur %-16s victime %-22s indice lu %2d  %s", m.event.timeMS,
			m.event.killer, m.event.victim, m.victimeLue, etat)
	}
}

// journaliserCandidatsDeBot : les dead-states qui portent un indice de bot, et s ils ont servi.
func journaliserCandidatsDeBot(t *testing.T, c *decodeCtx, p *pass) {
	t.Helper()
	t.Logf("DEAD-STATES A INDICE DE BOT (population de l hybride)")
	for _, cd := range p.all {
		if !c.isBotSide(cd.candidate) {
			continue
		}
		servi := p.botUsed[[3]int{cd.chunk, cd.pidx, cd.bit}]
		t.Logf("   %7d ms  victime %2d (%-22s)  tueur %2d (%-22s)  voie %-12s tag %08x servi %v", cd.ms,
			cd.victim, c.roster.nameOf(cd.victim), cd.killer, c.roster.nameOf(cd.killer), cd.path,
			cd.tag, servi)
	}
}
