//go:build research

package replay

// kill_positions_cout_vehicules_research_test.go — E8.1 (c) : COMBIEN COUTERAIT, DANS LA PASSE DES
// POSITIONS DU COLLECTEUR, LA LECTURE DES VEHICULES ET DE L'OCCUPATION ?
//
// UN film, decode sous `filmproc.AcquireSolo` et le plafond memoire, rien d'ecrit. Chaque lecture est
// chronometree sur un contexte NEUF (aucun cache de contexte partage entre deux mesures), sur la
// carte du film (`NewFilmContextForMap` + `PoserLaCarteEtLeDecoupage`, comme la cuisson).
//
//	KPC_FILMS=<repo>/data/cache/film_chunks KPC_FAITS=<repo>/data/cache/film_facts/halo_infinite \
//	  KPC_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json KPC_ID=4f77afc1 \
//	  KPC_VERROU=<repo>/data/cache go test -tags research -count=1 -run '^TestCoutDesLecturesVehicules$' \
//	  -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestCoutDesLecturesVehicules(t *testing.T) {
	films, faits, catalogue, id, verrou := os.Getenv("KPC_FILMS"), os.Getenv("KPC_FAITS"),
		os.Getenv("KPC_CATALOGUE"), os.Getenv("KPC_ID"), os.Getenv("KPC_VERROU")
	if films == "" || faits == "" || catalogue == "" || id == "" || verrou == "" {
		t.Skip("KPC_* absent : instrument saute")
	}
	lock, err := filmproc.AcquireSolo(verrou, "kp-cout-vehicules", id)
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority("kp-cout-vehicules")
	garde := filmproc.Arm("kp-cout-vehicules", 3, func(pic uint64) { panic(fmt.Sprintf("plafond memoire franchi : %d", pic)) })
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
	film, err := source.LoadDir(filepath.Join(films, id), nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	neuf := func() *grammar.FilmContext {
		fc := grammar.NewFilmContextForMap(film, &entry, nil)
		fc.PoserLaCarteEtLeDecoupage()
		return fc
	}
	chrono := func(nom string, f func(fc *grammar.FilmContext) string) {
		fc := neuf()
		debug.FreeOSMemory()
		t0 := time.Now()
		res := f(fc)
		t.Logf("%-46s %8.2f s  %s", nom, time.Since(t0).Seconds(), res)
	}
	rng := entry.Range()
	chrono("pont du collecteur (ScanPontDIdentite)", func(fc *grammar.FilmContext) string {
		opt := grammar.DefaultScanFilmOptions()
		opt.WorldRange, opt.Layout = &rng, fc.ImposedLayout()
		p := grammar.ScanPontDIdentite(fc, grammar.OptionsDuPont{Balayage: opt, Carte: &entry})
		return fmt.Sprintf("positions=%d morts=%d", len(p.Positions), len(p.Morts))
	})
	chrono("marche de killsource (deja payee par le collecteur)", func(fc *grammar.FilmContext) string {
		l, err := grammar.LireLaMarcheDeKillsource(fc)
		return fmt.Sprintf("kills=%d err=%v", len(l.Kills), err)
	})
	chrono("marche des trames {Morts} (occupation i10)", func(fc *grammar.FilmContext) string {
		m, err := grammar.ScanMarcheDesTramesAvec(fc, grammar.LecturesDeLaMarche{Morts: true})
		return fmt.Sprintf("occupation=%d err=%v", len(m.Occupancy), err)
	})
	chrono("images-cles ti=40 + nuage de positions ti=40", func(fc *grammar.FilmContext) string {
		kf := grammar.ScanWorldObjectKeyframes(fc, grammar.VehicleTypeIndex)
		opt := vehicleScanOptions(fc, &rng)
		pos, err := grammar.ScanBipedPositionsForBand(fc, grammar.NewSlotBand(kf.Band), opt)
		return fmt.Sprintf("bande=%d positions=%d err=%v", len(kf.Band), len(pos), err)
	})
	chrono("evenements d embarquement (passe des tetes)", func(fc *grammar.FilmContext) string {
		ev, err := grammar.ScanVehicleEvents(fc)
		return fmt.Sprintf("evenements=%d err=%v", len(ev), err)
	})
}
