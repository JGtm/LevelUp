package grammar

// poussees_films_test.go — LES POUSSEES DES TROIS TEMOINS DU LOT F, sur les films reels. Gate LOCAL
// (la CI n a pas de films) :
//
//	REPLAY_FILM_CACHE=<parc>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run PousseesDesTemoins -v
//
// Chaque poussee attendue est (instant en ms de film, slot pousse, slot de l auteur). Les instants
// sont controles a la trajectoire (le slot pousse accelere dans les 300 ms) ; 215e7022 porte le kill
// au repulseur releve au Theater a 325 526 ms. CONTROLE DU PORTEUR : la derniere lecture i48 de la vie de
// l auteur avant la poussee est le rang du repulseur de la palette du film (6 sur les trois). Le contexte
// est celui de la cuisson : entree de catalogue de la carte, puis carte et decoupage poses
// ([FilmContext.PoserLaCarteEtLeDecoupage], `replay/build_from_film.go`) — les rangs lus en dependent.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

type pousseeAttendue struct {
	ms          uint64
	pousse, par uint32
}

type temoinDePoussees struct {
	film, carte   string
	rangRepulseur int
	attendues     []pousseeAttendue
}

var temoinsDePoussees = []temoinDePoussees{
	{"0d9a9af9", "Forest", 6, []pousseeAttendue{{85956, 519, 513}, {95583, 519, 513}, {270179, 535, 532}}},
	{"215e7022", "Argyle", 6, []pousseeAttendue{{325526, 563, 560}, {325671, 564, 560}}},
	{"a6ae19fb", "Kiken'na", 6, []pousseeAttendue{
		{59858, 518, 519}, {104953, 524, 523}, {241207, 544, 526}, {339389, 562, 559}, {403205, 576, 526},
		{404472, 578, 573}, {423392, 583, 526}, {437454, 585, 526}, {448416, 587, 569}, {454724, 589, 526},
	}},
}

func TestPousseesDesTemoins(t *testing.T) {
	cache := os.Getenv("REPLAY_FILM_CACHE")
	if cache == "" {
		t.Skip("REPLAY_FILM_CACHE absent : gate local des poussees saute")
	}
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tm := range temoinsDePoussees {
		entree, err := cat.Lookup(tm.carte)
		if err != nil {
			t.Fatalf("%s : %v", tm.film, err)
		}
		film, err := source.LoadDir(filepath.Join(cache, tm.film), nil)
		if err != nil {
			t.Fatalf("%s : %v", tm.film, err)
		}
		fc := NewFilmContextForMap(film, &entree, nil)
		fc.PoserLaCarteEtLeDecoupage()
		verifierLesPoussees(t, tm, fc)
	}
}

func verifierLesPoussees(t *testing.T, tm temoinDePoussees, fc *FilmContext) {
	t.Helper()
	origine, err := ScanClockOrigin(fc.film)
	if err != nil {
		t.Fatalf("%s : origine : %v", tm.film, err)
	}
	ks, st := ScanKnockbacks(fc)
	if len(ks) != len(tm.attendues) {
		t.Fatalf("%s : %d poussee(s) (%+v), attendu %d", tm.film, len(ks), st, len(tm.attendues))
	}
	rangs, _, err := ScanAbilityRanks(fc)
	if err != nil {
		t.Fatalf("%s : rangs : %v", tm.film, err)
	}
	for i, k := range ks {
		a := tm.attendues[i]
		ms := (k.TimestampUS - origine) / 1000
		if ms != a.ms || k.Pushed.Slot != a.pousse || !k.HasAuthor || k.Author.Slot != a.par || !k.HasImpulse {
			t.Errorf("%s : poussee %d = %d ms, pousse %d, auteur %d (%v), attendu %+v", tm.film, i, ms,
				k.Pushed.Slot, k.Author.Slot, k.HasAuthor, a)
			continue
		}
		dernier := -1
		for _, r := range rangs {
			if r.Slot == k.Author.Slot && r.TimestampUS <= k.TimestampUS {
				dernier = r.Rank
			}
		}
		if dernier != tm.rangRepulseur {
			t.Errorf("%s : poussee a %d ms, auteur %d : dernier rang %d, attendu le repulseur (%d)", tm.film,
				ms, k.Author.Slot, dernier, tm.rangRepulseur)
		}
	}
}
