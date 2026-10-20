package grammar

// generations_vivantes_datees_films_test.go — LES TROIS POINTS ABERRANTS DU CONSTAT C2 (G-corpus
// J11.1, 2026-09-28), sur les films reels : aucun n est plus lu en position. Gate LOCAL (la CI n a
// pas de films) :
//
//	REPLAY_FILM_CACHE=<parc>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run PointsAberrantsC2 -v
//
// Pieces (instrument `r2_positions_aberrantes_research_test.go`) : chaque point est un en-tete de
// generation 2 ancre AVANT le record de creation de son corps — `a349fea8` slot 532 a
// 9 499 373 382 us (creation de (532, 2) a 9 791 544 840), `084a804d` slot 623 a 12 713 881 538
// (creation de (623, 2) a 13 313 289 984) et slot 516 a 12 991 592 069 (creation de (516, 2) a
// 12 994 900 961, 3,3 s plus tard).

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

type pointAberrantC2 struct {
	film, carte string
	slot        uint32
	tUS         uint64
}

var pointsAberrantsC2 = []pointAberrantC2{
	{"a349fea8", "Fragmentation Heavies", 532, 9_499_373_382},
	{"084a804d", "Fortitude Heavies", 623, 12_713_881_538},
	{"084a804d", "Fortitude Heavies", 516, 12_991_592_069},
}

func TestPointsAberrantsC2(t *testing.T) {
	cache := os.Getenv("REPLAY_FILM_CACHE")
	if cache == "" {
		t.Skip("REPLAY_FILM_CACHE absent : gate local des points aberrants C2 saute")
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
	parFilm := map[string][]pointAberrantC2{}
	var ordre []string
	for _, p := range pointsAberrantsC2 {
		if parFilm[p.film] == nil {
			ordre = append(ordre, p.film)
		}
		parFilm[p.film] = append(parFilm[p.film], p)
	}
	for _, id := range ordre {
		dir := filepath.Join(cache, id)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("film %s absent du cache %s", id, cache)
			continue
		}
		e, err := cat.Lookup(parFilm[id][0].carte)
		if err != nil {
			t.Fatal(err)
		}
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		opt := DefaultScanFilmOptions()
		wr := e.Range()
		opt.WorldRange = &wr
		pos, err := ScanBipedPositions(NewFilmContextForMap(film, &e, nil), opt)
		if err != nil {
			t.Fatal(err)
		}
		for _, pt := range parFilm[id] {
			for _, p := range pos {
				if p.Slot == pt.slot && p.TimestampUS == pt.tUS {
					t.Errorf("%s slot %d t=%d : position (%.2f ; %.2f ; %.2f) LUE — en-tete de generation 2 "+
						"anterieur a la creation de son corps (constat C2)", id, pt.slot, pt.tUS, p.X, p.Y, p.Z)
				}
			}
		}
	}
}
