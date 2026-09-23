package sync

// pooled_client_metrics_test.go — UN FILM PAS ENCORE FINALISE N EST PAS UNE ERREUR D APPEL (lot L3
// du 2026-09-23, constat L3-R3 de sa revue adverse).

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
)

// TestObserveHaloCall_FilmNonFinalise_PasUneErreurReseau : le refus d'un film non finalise ne
// compte ni en erreur de l'appel ni en erreur reseau ; une vraie erreur non HTTP, elle, compte
// toujours (le temoin qui prouve que le test lit les bons compteurs).
func TestObserveHaloCall_FilmNonFinalise_PasUneErreurReseau(t *testing.T) {
	const titre, appel = "halo_infinite", "film_chunks"
	lire := func() (int64, int64) {
		return observability.LoadCounterT(titre, "halo_api_err_"+appel+"_total"),
			observability.LoadCounterT(titre, "halo_api_network_total")
	}
	errAppel0, reseau0 := lire()
	observeHaloCall(titre, appel, "", time.Now(),
		fmt.Errorf("GetFilmChunks(m) : %w", filmcache.ErrFilmNonFinalise))
	errAppel1, reseau1 := lire()
	if errAppel1 != errAppel0 || reseau1 != reseau0 {
		t.Errorf("film non finalise compte en erreur : appel +%d, reseau +%d, attendu 0 et 0",
			errAppel1-errAppel0, reseau1-reseau0)
	}

	observeHaloCall(titre, appel, "", time.Now(), errors.New("connexion reinitialisee"))
	errAppel2, reseau2 := lire()
	if errAppel2-errAppel1 != 1 || reseau2-reseau1 != 1 {
		t.Errorf("temoin : erreur reseau reelle comptee appel +%d, reseau +%d, attendu +1 et +1",
			errAppel2-errAppel1, reseau2-reseau1)
	}
}
