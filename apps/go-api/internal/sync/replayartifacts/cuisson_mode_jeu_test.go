package replayartifacts

import (
	"context"
	"errors"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/gameproc"
)

// installerPartie pose un mode jeu dont le jeu tourne, le temps du test.
func installerPartie(t *testing.T) {
	t.Helper()
	m := gameproc.NewMode()
	w := &gameproc.Watcher{
		Lister: gameproc.ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(context.Background())
	prev := gameproc.SetDefault(m)
	t.Cleanup(func() { gameproc.SetDefault(prev) })
}

// TestBuildAll_ModeJeu_AucuneCuissonNeDemarre : tant que le jeu tourne, la boucle persiste les films
// (ils expirent côté serveur Halo) mais ne lance AUCUNE cuisson ; les matchs sont comptés reportés,
// jamais en échec.
func TestBuildAll_ModeJeu_AucuneCuissonNeDemarre(t *testing.T) {
	installerPartie(t)
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, FilmRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			appels++
			return BuildOneResult{}, errors.New("aucune cuisson ne doit partir pendant la partie")
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "0000c001"}, {matchID: "0000c002"}})

	if appels != 0 {
		t.Fatalf("%d cuisson(s) lancée(s) pendant la partie, attendu 0", appels)
	}
	if b.reportesModeJeu != 2 || b.echecs != 0 {
		t.Errorf("reportés mode jeu = %d, échecs = %d ; attendu 2 et 0", b.reportesModeJeu, b.echecs)
	}
	if b.filmsSauves != 2 {
		t.Errorf("films persistés = %d, attendu 2 : le pont disque continue pendant la partie", b.filmsSauves)
	}
}

// TestBuildAll_HorsPartie_LaCuissonPart : le mode jeu inactif ne change rien.
func TestBuildAll_HorsPartie_LaCuissonPart(t *testing.T) {
	prev := gameproc.SetDefault(gameproc.NewMode())
	t.Cleanup(func() { gameproc.SetDefault(prev) })
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, FilmRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			appels++
			return BuildOneResult{}, errors.New("echec ordinaire")
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "0000c003"}})
	if appels != 1 || b.reportesModeJeu != 0 {
		t.Fatalf("cuissons = %d, reportés mode jeu = %d ; attendu 1 et 0", appels, b.reportesModeJeu)
	}
}
