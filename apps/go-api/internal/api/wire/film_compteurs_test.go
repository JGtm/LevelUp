package wire

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/observability"

	"github.com/JGtm/LevelUp/libs/film/filmobs"
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

// Le serveur branche le puits par le chemin réel : construire le registre de services suffit à
// republier les compteurs du film. Retirer l'appel de `NewServiceRegistry` rend ce test rouge
// (le test ci-dessus ne le voit pas, il appelle `brancherCompteursDuFilm` lui-même).
func TestNewServiceRegistryBrancheLePuitsDesCompteursDuFilm(t *testing.T) {
	const nom = "film_compteurs_registre_wire_test"
	filmobs.SetSink(nil)
	t.Cleanup(func() { filmobs.SetSink(nil) })

	NewServiceRegistry(&config.AppConfig{}, nil)

	avant := observability.LoadCounter(nom)
	filmobs.AddInt(nom, 3)
	if got := observability.LoadCounter(nom) - avant; got != 3 {
		t.Fatalf("observability a reçu %d, attendu 3 : NewServiceRegistry ne branche pas le puits", got)
	}
}
