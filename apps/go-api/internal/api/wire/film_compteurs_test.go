package wire

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmobs"
	"levelup/go-api/internal/observability"
)

// Les compteurs du film doivent atteindre `/debug/vars` sous leur nom d'origine : sans le puits,
// la brique les compterait pour elle seule.
func TestCompteursDuFilmAtteignentObservability(t *testing.T) {
	const nom = "film_compteurs_wire_test"
	t.Cleanup(func() { filmobs.SetSink(nil) })
	brancherCompteursDuFilm()

	avant := observability.LoadCounter(nom)
	filmobs.IncCounter(nom)
	filmobs.AddInt(nom, 2)
	if got := observability.LoadCounter(nom) - avant; got != 3 {
		t.Fatalf("observability a reçu %d, attendu 3", got)
	}
}
