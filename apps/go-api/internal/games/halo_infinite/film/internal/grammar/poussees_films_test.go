package grammar

// poussees_films_test.go — LES POUSSEES DES TROIS TEMOINS DU LOT F, sur les films reels. Gate LOCAL
// (la CI n a pas de films) :
//
//	REPLAY_FILM_CACHE=<parc>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run PousseesDesTemoins -v
//
// Chaque poussee attendue est (instant en ms de film, slot pousse, slot de l auteur). Les instants
// sont controles a la trajectoire (le slot pousse accelere dans les 300 ms) ; 215e7022 porte le kill
// au repulseur releve au Theater a 325 526 ms. CONTROLE DU PORTEUR : la vie de l auteur a lu le rang du
// repulseur de la palette du film (6 sur les trois) avant la poussee, et aucune consommation (`spent`)
// ne l a videe depuis. Ce n est pas « la derniere lecture » : la prise d un bonus (surbouclier) change
// le rang lu sans vider l equipement porte.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
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
		verifierLesPoussees(t, tm, NewFilmContextForMap(film, &entree, nil))
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
	changements, _, err := ScanEquipmentChanges(fc, func(uint32) (uint64, bool) { return 0, false })
	if err != nil {
		t.Fatalf("%s : changements : %v", tm.film, err)
	}
	for i, k := range ks {
		a := tm.attendues[i]
		ms := (k.TimestampUS - origine) / 1000
		if ms != a.ms || k.Pushed.Slot != a.pousse || !k.HasAuthor || k.Author.Slot != a.par || !k.HasImpulse {
			t.Errorf("%s : poussee %d = %d ms, pousse %d, auteur %d (%v), attendu %+v", tm.film, i, ms,
				k.Pushed.Slot, k.Author.Slot, k.HasAuthor, a)
			continue
		}
		var pris uint64
		for _, r := range rangs {
			if r.Slot == k.Author.Slot && r.TimestampUS <= k.TimestampUS && r.Rank == tm.rangRepulseur {
				pris = r.TimestampUS
			}
		}
		vide := false
		for _, c := range changements {
			if c.Slot == k.Author.Slot && c.Kind == types.EquipmentSpent && c.TimestampUS > pris && c.TimestampUS <= k.TimestampUS {
				vide = true
			}
		}
		if pris == 0 || vide {
			t.Errorf("%s : poussee a %d ms, auteur %d : porteur du repulseur %v, vide depuis %v", tm.film, ms,
				k.Author.Slot, pris != 0, vide)
		}
	}
}
