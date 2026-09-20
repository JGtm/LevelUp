//go:build research

package killsource

// calib_p4_research_test.go — LOT 5.1.7 : CE QUE LA CALIBRATION DE `killsource` RETENAIT.
//
// La chaine `Result.Calibration` porte, a la BASE du lot, la valeur que le balayage
// `calibrateRSP` retenait pour `param_4` (`recordStateParam=N [croissance xN]`). C est elle
// qu il faut lire pour savoir si, sur un film donne, la base lisait `ti=40` a une largeur
// DIFFERENTE de celle du registre (3 / 2 / 4 pour `i10` / `i19` / `i20`).
//
// LECTURE SEULE, UN FILM PAR INVOCATION :
//
//	CALIB_FILM=<abs>/film_chunks/084a804d CALIB_CARTE="Fortitude Heavies" \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	  -run '^TestCalibrationP4$' -v -timeout 60m

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestCalibrationP4(t *testing.T) {
	dir, carte := os.Getenv("CALIB_FILM"), os.Getenv("CALIB_CARTE")
	if dir == "" {
		t.Skip("instrument de mesure : CALIB_FILM requis")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	o := DefaultOptions()
	if carte != "" {
		cat, err := profile.LoadMapQuantCatalog(os.Getenv("CALIB_CATALOGUE"))
		if err != nil {
			t.Fatalf("catalogue : %v", err)
		}
		e, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("carte %q : %v", carte, err)
		}
		o.Carte = &e
	}
	res, err := Decode(context.Background(), filepath.Base(dir), film, &o)
	if err != nil {
		t.Fatalf("decodage : %v", err)
	}
	t.Logf("FILM %s — CALIBRATION : %s", filepath.Base(dir), res.Calibration)
}
