package wire

import (
	"levelup/go-api/internal/games/halo_infinite/film/filmobs"
	"levelup/go-api/internal/observability"
)

// brancherCompteursDuFilm republie sur `/debug/vars` les compteurs que le décodeur de film tient
// lui-même (`filmobs`, la brique ne connaît pas `observability`) : chaque incrément du film
// atteint aussi `observability.AddInt`, sous le même nom. Idempotent ; posé à la construction du
// registre de services, point de démarrage unique du serveur.
func brancherCompteursDuFilm() {
	filmobs.SetSink(observability.AddInt)
}
