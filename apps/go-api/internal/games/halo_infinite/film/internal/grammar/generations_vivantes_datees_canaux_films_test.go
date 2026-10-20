package grammar

// generations_vivantes_datees_canaux_films_test.go — LES CANAUX DELTA NE LISENT PLUS UN CORPS AVANT
// SA CREATION (lot R2-bis, 2026-09-29, decouverte 2 du lot R2), sur les films reels. Gate LOCAL (la
// CI n a pas de films) :
//
//	REPLAY_FILM_CACHE=<parc>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run CanauxDelta -v
//
// Pieces (instrument `r2bis_canaux_dates_research_test.go`, 2026-09-29) : avant le lot, le marcheur
// des huit canaux (`walkDeltaBipedRecords`) rendait sur `084a804d` 32 records de generation 2 dont le
// record de creation du corps est lu PLUS TARD — des en-tetes qui ne sont la replication d aucun
// corps, lus par le camouflage, le grappin, l arme tenue, l inventaire... Le test exige qu aucun
// record rendu par le marcheur, ni aucune emission retenue par la recuperation d equipement, ne
// designe un corps a un instant ou la garde datee ([GenerationsVivantes.A]) le refuse.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/testutil"
)

// filmsCanauxDates : les films a slot reboucle ou le marcheur acceptait des en-tetes anterieurs a la
// creation de leur corps (32, 30, 20 et 102 avant le lot).
var filmsCanauxDates = []struct{ film, carte string }{
	{"084a804d", "Fortitude Heavies"},
	{"a349fea8", "Fragmentation Heavies"},
	{"4f77afc1", "Flood Gulch"},
	{"1c4c63c2", "Refuge"},
}

func TestCanauxDelta_AucunCorpsLuAvantSaCreation(t *testing.T) {
	cache := os.Getenv("REPLAY_FILM_CACHE")
	if cache == "" {
		t.Skip("REPLAY_FILM_CACHE absent : gate local des canaux dates (lot R2-bis) saute")
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
	for _, f := range filmsCanauxDates {
		dir := filepath.Join(cache, f.film)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("film %s absent du cache %s", f.film, cache)
			continue
		}
		e, err := cat.Lookup(f.carte)
		if err != nil {
			t.Fatal(err)
		}
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		wr := e.Range()
		verifierCanauxDates(t, f.film, NewFilmContextForMap(film, &e, nil), &wr)
	}
}

// verifierCanauxDates juge, sur un film, le marcheur des huit canaux puis la recuperation.
func verifierCanauxDates(t *testing.T, film string, fc *FilmContext, wr *profile.Vec3Range) {
	t.Helper()
	gens := fc.GenerationsVivantes()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("%s : decoupage i0 : %v", film, err)
	}
	records, avant := 0, 0
	walkDeltaBipedRecords(fc, fc.ChunkNumbers(), fc.BipedSlots(), lay, func(r deltaBipedRecord) {
		records++
		if !gens.A(r.Packet.TimestampUS).Accepte(r.Vie()) {
			if avant++; avant <= 3 {
				t.Errorf("%s : record du corps (%d, %d) a %d us, anterieur a sa creation, rendu aux canaux",
					film, r.Slot, r.Gen, r.Packet.TimestampUS)
			}
		}
	})
	if records == 0 {
		t.Fatalf("%s : aucun record delta : le test ne prouverait rien", film)
	}
	if avant > 0 {
		t.Errorf("%s : %d record(s) sur %d designent un corps avant son record de creation (lot R2-bis)",
			film, avant, records)
	}
	for _, e := range recuperationsDuFilm(t, fc, wr) {
		if !gens.A(e.TimestampUS).Accepte(types.LifeKey{Slot: e.Slot, Gen: e.Gen}) {
			t.Errorf("%s : emission d equipement recuperee du corps (%d, %d) a %d us, anterieure a sa creation",
				film, e.Slot, e.Gen, e.TimestampUS)
		}
	}
}

// recuperationsDuFilm rejoue la recuperation d equipement de [ScanEquipmentChanges], naissances par
// slot comprises (le premier echantillon de position du slot, comme la cuisson).
func recuperationsDuFilm(t *testing.T, fc *FilmContext, wr *profile.Vec3Range) []equipRecovered {
	t.Helper()
	setup, err := resolveAbilityScan(fc)
	if err != nil {
		t.Fatalf("balayage des capacites : %v", err)
	}
	parVie := map[types.LifeKey][]abilityEmission{}
	walkAbilityEmissionsWith(setup, func(e abilityEmission) { parVie[e.Vie()] = append(parVie[e.Vie()], e) })
	for _, l := range parVie {
		sortEmissionsByFilmOrder(l)
	}
	opt := DefaultScanFilmOptions()
	opt.WorldRange = wr
	pos, err := ScanBipedPositions(fc, opt)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	naissance := map[uint32]uint64{}
	for _, p := range pos {
		if at, ok := naissance[p.Slot]; !ok || p.TimestampUS < at {
			naissance[p.Slot] = p.TimestampUS
		}
	}
	bornAt := func(s uint32) (uint64, bool) { at, ok := naissance[s]; return at, ok }
	return scanEquipmentRecovery(setup, buildEquipRecoveryWindows(parVie, bornAt))
}
