package wire

import (
	"levelup/go-api/internal/observability"

	"github.com/JGtm/LevelUp/libs/film/filmobs"
)

// brancherCompteursDuFilm republie sur `/debug/vars` les compteurs que le décodeur de film tient
// lui-même (`filmobs`, la brique ne connaît pas `observability`) : chaque incrément du film
// atteint aussi `observability.AddInt`, sous le même nom. Idempotent ; posé à la construction du
// registre de services, point de démarrage unique du serveur.
func brancherCompteursDuFilm() {
	filmobs.SetSink(observability.AddInt)
}
