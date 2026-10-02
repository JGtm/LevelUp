package replaybuild

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// kills_test.go — la résolution d'identité hors ligne (gamertag -> xuid), sur données
// 100 % synthétiques (aucun film) : PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1.

func TestResolveKillIdentity_Gamertag(t *testing.T) {
	byGamertag := map[string]uint64{"Chocoboflor": 2533274822053343}
	xuid, ok := resolveKillIdentity("Chocoboflor", byGamertag)
	if !ok || xuid != 2533274822053343 {
		t.Fatalf("xuid=%d ok=%v, attendu 2533274822053343/true", xuid, ok)
	}
}

func TestResolveKillIdentity_GamertagInconnu(t *testing.T) {
	xuid, ok := resolveKillIdentity("Inconnu", map[string]uint64{"Autre": 1})
	if ok || xuid != 0 {
		t.Fatalf("xuid=%d ok=%v, attendu 0/false (gamertag hors roster)", xuid, ok)
	}
}

func TestGamertagXUIDIndex_DoublonNeResoutRien(t *testing.T) {
	deaths := []types.Death{
		{XUID: 111, Gamertag: "Joueur"},
		{XUID: 222, Gamertag: "Joueur"}, // même nom, second xuid : le nom ne désigne plus personne
		{XUID: 333, Gamertag: ""},       // sans gamertag : absent de l'index
		{XUID: 444, Gamertag: "Seul"},
	}
	idx, ambigus := gamertagXUIDIndex(deaths)
	if len(idx) != 1 || idx["Seul"] != 444 || ambigus != 1 {
		t.Fatalf("index = %v, ambigus = %d — attendu {Seul: 444} et 1 ambigu", idx, ambigus)
	}
}

func TestGamertagXUIDIndex_ListeVide(t *testing.T) {
	if idx, _ := gamertagXUIDIndex(nil); len(idx) != 0 {
		t.Fatalf("index = %+v, attendu vide", idx)
	}
}

// --- Lot G.6 (2026-09-05) : la VICTIME entre dans la résolution -------------------------
//
// Ces cas tiennent la SECONDE sortie de `resolveKills` — les couples (tueur, victime, instant)
// que `bomb_carriers_killed` consomme. Ils sont écrits pour qu'un couple mal formé ne puisse pas
// se glisser en silence : chaque perte est comptée, et le total des deux compteurs plus les
// couples publiés vaut le nombre de kills fournis.

// killDe fabrique un kill killsource minimal : un tueur, une victime, un instant.
func killDe(tueur, victime string, ms int) decfilm.Kill {
	return decfilm.Kill{TimeMS: ms, Victim: victime, Feed: decfilm.FeedTruth{Killer: tueur, Present: true}}
}

func TestResolveKills_TueurEtVictimeResolus(t *testing.T) {
	idx := map[string]uint64{"Tueur": 11, "Victime": 22}
	r := resolveKills([]decfilm.Kill{killDe("Tueur", "Victime", 4200)}, idx)
	if len(r.refs) != 1 || r.refs[0].XUID != 11 || r.refs[0].TimeMS != 4200 {
		t.Fatalf("refs = %+v, attendu un frag du xuid 11 à 4200 ms", r.refs)
	}
	if len(r.pairs) != 1 {
		t.Fatalf("pairs = %+v, attendu un couple", r.pairs)
	}
	// L'INSTANT VOYAGE VERBATIM : `decfilm.Kill.TimeMS` est déjà sur l'horloge du match
	// (celle du fil des morts) — aucune conversion ici, ni ailleurs (cf. MatchKillsInput).
	veut := replay.KillRef{KillerXUID: 11, VictimXUID: 22, TimeMS: 4200}
	if r.pairs[0] != veut {
		t.Fatalf("couple = %+v, attendu %+v", r.pairs[0], veut)
	}
	if r.killerUnresolved != 0 || r.victimUnresolved != 0 {
		t.Fatalf("pertes = %d tueurs / %d victimes, attendu 0/0", r.killerUnresolved, r.victimUnresolved)
	}
}

func TestResolveKills_VictimeInconnueEcarteeEtComptee(t *testing.T) {
	// Cas NOMINAL : la victime est un bot, elle n'a aucun xuid au fil des morts. Le frag
	// reste crédité au tueur (jointure équipement), le COUPLE est perdu — et compté.
	idx := map[string]uint64{"Tueur": 11}
	r := resolveKills([]decfilm.Kill{killDe("Tueur", "Bot 004", 900)}, idx)
	if len(r.refs) != 1 {
		t.Fatalf("refs = %+v, attendu le frag du tueur malgré la victime inconnue", r.refs)
	}
	if len(r.pairs) != 0 {
		t.Fatalf("pairs = %+v, attendu aucun couple", r.pairs)
	}
	if r.victimUnresolved != 1 || r.killerUnresolved != 0 {
		t.Fatalf("pertes = %d tueurs / %d victimes, attendu 0/1", r.killerUnresolved, r.victimUnresolved)
	}
}

func TestResolveKills_TueurInconnuPerdLesDeuxSorties(t *testing.T) {
	idx := map[string]uint64{"Victime": 22}
	r := resolveKills([]decfilm.Kill{killDe("Fantome", "Victime", 900)}, idx)
	if len(r.refs) != 0 || len(r.pairs) != 0 {
		t.Fatalf("refs = %+v, pairs = %+v, attendu vides", r.refs, r.pairs)
	}
	if r.killerUnresolved != 1 || r.victimUnresolved != 0 {
		t.Fatalf("pertes = %d tueurs / %d victimes, attendu 1/0 (la victime n'est pas atteinte)",
			r.killerUnresolved, r.victimUnresolved)
	}
}

// TestResolveKills_ComptesConserves : sur une population mêlée, RIEN ne disparaît sans être
// compté — c'est l'invariant qui rend `MatchKillsInput.Dropped` lisible.
func TestResolveKills_ComptesConserves(t *testing.T) {
	idx := map[string]uint64{"A": 1, "B": 2}
	kills := []decfilm.Kill{
		killDe("A", "B", 100),       // couple complet
		killDe("B", "Bot 001", 200), // victime bot
		killDe("Inconnu", "A", 300), // tueur hors roster
	}
	r := resolveKills(kills, idx)
	if len(r.pairs)+r.killerUnresolved+r.victimUnresolved != len(kills) {
		t.Fatalf("%d couples + %d tueurs perdus + %d victimes perdues != %d kills fournis",
			len(r.pairs), r.killerUnresolved, r.victimUnresolved, len(kills))
	}
	if len(r.pairs) != 1 {
		t.Fatalf("pairs = %+v, attendu 1", r.pairs)
	}
	if len(r.refs) != 2 {
		t.Fatalf("refs = %d, attendu 2 (seul le tueur hors roster manque)", len(r.refs))
	}
}

// TestKillRefs_PortesFermees : les trois refus rendent les DEUX entrées non lues. Un couple à
// zéro se lirait comme « personne n'a tué de porteur », ce qui est une affirmation.
func TestKillRefs_PortesFermees(t *testing.T) {
	b := &Builder{}
	deaths := filmDeaths{list: []types.Death{{XUID: 22, Gamertag: "Victime"}}}
	// `BijectionMargin > 0` et aucune alerte de santé : la porte ligne-par-ligne est OUVERTE,
	// ce qui isole chacun des deux autres refus.
	ouvert := &decfilm.Result{BijectionMargin: 1, Kills: []decfilm.Kill{killDe("Tueur", "Victime", 10)}}
	if _, mk := b.killRefs(context.Background(), "m", deaths, nil, nil); mk.Read || len(mk.Kills) != 0 {
		t.Fatalf("killsource nil : MatchKills = %+v, attendu non lu", mk)
	}
	if _, mk := b.killRefs(context.Background(), "m", deaths, &decfilm.Result{}, nil); mk.Read {
		t.Fatalf("porte ligne-par-ligne fermée : MatchKills lu, attendu non lu")
	}
	if _, mk := b.killRefs(context.Background(), "m", filmDeaths{err: errFilTest}, ouvert, nil); mk.Read {
		t.Fatalf("fil des morts illisible : MatchKills lu, attendu non lu")
	}
	// Porte OUVERTE et fil LISIBLE : la sortie est lue, et le compte des écartés est publié.
	_, mk := b.killRefs(context.Background(), "m", deaths, ouvert, nil)
	if !mk.Read || len(mk.Kills) != 0 || mk.Dropped != 1 {
		t.Fatalf("MatchKills = %+v, attendu lu, 0 couple, 1 écarté (le tueur est hors roster)", mk)
	}
}

var errFilTest = errors.New("fil des morts illisible (test)")
