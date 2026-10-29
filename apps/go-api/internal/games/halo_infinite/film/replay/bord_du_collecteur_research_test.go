//go:build research

package replay

// bord_du_collecteur_research_test.go — E8.1 (b) (c) : LA LECTURE DU BORD DU COLLECTEUR EST-ELLE CELLE
// QUE LES MESURES ONT LUE DANS LES FAITS CUITS ?
//
// Les mesures avant / apres d'E8.1 lisent l'occupation et le nuage des vehicules dans les faits de la
// CUISSON (`Vehicles.Occupancy`, `Vehicles.Positions`). La passe des positions du collecteur les lit
// autrement : l'occupation du decodage killsource (`Result.Occupation`) et le nuage
// ([PositionsDesVehicules]) sur un contexte ouvert comme le sien (carte, sans pose du decoupage).
// UN film, decode sous `filmproc.AcquireSolo` et le plafond memoire, rien d'ecrit : les deux lectures
// sont comparees a celles de ses faits.
//
//	BDC_FILMS=<repo>/data/cache/film_chunks BDC_FAITS=<repo>/data/cache/film_facts/halo_infinite \
//	  BDC_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json BDC_ID=4f77afc1 \
//	  BDC_VERROU=<repo>/data/cache go test -tags research -count=1 -run '^TestBordDuCollecteurEgaleLesFaits$' \
//	  -v ./internal/games/halo_infinite/film/replay/

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestBordDuCollecteurEgaleLesFaits(t *testing.T) {
	films, faits, catalogue, id, verrou := os.Getenv("BDC_FILMS"), os.Getenv("BDC_FAITS"),
		os.Getenv("BDC_CATALOGUE"), os.Getenv("BDC_ID"), os.Getenv("BDC_VERROU")
	if films == "" || faits == "" || catalogue == "" || id == "" || verrou == "" {
		t.Skip("BDC_* absent : instrument saute")
	}
	lock, err := filmproc.AcquireSolo(verrou, "bord-du-collecteur", id)
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority("bord-du-collecteur")
	garde := filmproc.Arm("bord-du-collecteur", 3, func(pic uint64) { panic(fmt.Sprintf("plafond memoire franchi : %d", pic)) })
	defer garde.Disarm()

	blob, err := os.ReadFile(filepath.Join(faits, id+".filmfacts.bin")) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	head, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	var entry profile.MapQuantEntry
	for _, m := range cat.Maps {
		if m.Module == head.MapModule {
			entry = m
		}
	}
	ff, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	film, err := source.LoadDir(filepath.Join(films, id), nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	opts := killsource.DefaultOptions()
	opts.Carte = &entry
	res, err := killsource.Decode(context.Background(), id, film, &opts)
	if err != nil {
		t.Fatalf("killsource : %v", err)
	}
	rng := entry.Range()
	veh, err := PositionsDesVehicules(grammar.NewFilmContextForMap(film, &entry, nil), &rng)
	if err != nil {
		t.Fatalf("nuage des vehicules : %v", err)
	}
	occFaits := ff.Facts.Vehicles.Occupancy
	t.Logf("occupation : killsource %d lectures, faits %d, egales=%v", len(res.Occupation), len(occFaits),
		slices.Equal(res.Occupation, occFaits))
	vehFaits := ff.Facts.Vehicles.Positions
	t.Logf("nuage des vehicules : collecteur %d echantillons, faits %d, egaux=%v", len(veh), len(vehFaits),
		memesEchantillons(veh, vehFaits))
}

// memesEchantillons compare deux nuages par (slot, instant, quanta, monde).
func memesEchantillons(a, b []grammar.BipedPosition) bool {
	if len(a) != len(b) {
		return false
	}
	cle := func(p grammar.BipedPosition) string {
		return fmt.Sprintf("%d/%d/%v/%.3f/%.3f/%.3f", p.Slot, p.TimestampUS, p.Q, p.X, p.Y, p.Z)
	}
	ka, kb := make([]string, len(a)), make([]string, len(b))
	for i := range a {
		ka[i], kb[i] = cle(a[i]), cle(b[i])
	}
	slices.Sort(ka)
	slices.Sort(kb)
	return slices.Equal(ka, kb)
}
