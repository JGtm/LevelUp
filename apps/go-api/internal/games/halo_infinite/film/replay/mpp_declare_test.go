package replay

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestLaCuissonPoseLeDecoupageDeclareParLeFilm : la cuisson pose sur son contexte le decoupage MPP
// que le film declare (format 25 : 8/3 presume par mesure) ou que son format porte (format 27 :
// 9/5), PAR-DESSUS un profil calibre qui portait l invariant — c est l ordre de
// [poserProfilPuisCarte].
func TestLaCuissonPoseLeDecoupageDeclareParLeFilm(t *testing.T) {
	cas := []struct {
		court   string
		attendu profile.MPPWidths
	}{
		{"e5adf7b2", profile.MPPWidths{Lead: 8, Index: 3}},
		{"fb1a1a72", profile.MPPParDefaut()},
	}
	for _, c := range cas {
		film, err := source.LoadDir(filepath.Join("testdata", "minifilm_"+c.court), nil)
		if err != nil {
			t.Fatalf("%s : chargement %v", c.court, err)
		}
		fc := grammar.NewFilmContext(film)
		calibre := grammar.ProfilDeBalayageParDefaut()
		fc.PoserProfilDeBalayage(calibre)
		poserLeDecoupageMPPDuFilm(context.Background(), fc, c.court)
		if got := fc.ProfilDeBalayage().MPP; got != c.attendu {
			t.Errorf("%s : decoupage pose %v, attendu %v", c.court, got, c.attendu)
		}
	}
}
