package replay

// film_shims_test.go — LES ENVELOPPES D2 DES DECODEURS DE CALQUE, RESERVEES AUX TESTS.
//
// Le lot 1 de PLAN_CUISSON_PERF (2026-09-02) a fait passer `BuildFromFilm` et ses decodeurs de
// calque (`decodeFilmPlacements`, `decodeFilmPadScans`, `decodeFilmPadScan`) d'un REPERTOIRE a un
// `*filmsource.Film` deja charge : la cuisson ne decompresse plus le film une fois par balayage.
// Une vingtaine d'instruments de mesure de ce paquet les appellent avec un chemin de film sous
// garde d'environnement.
//
// CES ENVELOPPES VIVENT DANS UN `_test.go`, ET C'EST LE POINT : elles ne sont compilees que par
// la suite de tests, donc le compilateur lui-meme interdit qu'un chemin de production les
// appelle — la regle D2 (« enveloppes tolerees hors production ») n'est pas une convention ici.
//
// Elles chargent le film SANS metadonnees de manifeste : les balayages n'en ont pas besoin (le
// numero de chunk se lit dans le nom du fichier), seul l'armement de la bombe demande le
// `start_ms` par chunk, et son instrument de mesure charge le film lui-meme.

import (
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// filmDeDir charge le film d'un repertoire, ou rend nil. Les decodeurs rendent alors leur
// degradation habituelle (journalisee), exactement comme un repertoire illisible avant le lot 1.
func filmDeDir(dir string) *filmsource.Film {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil
	}
	return film
}

// buildFromFilmDir : [BuildFromFilm] depuis un REPERTOIRE de film. Ajoutee a la reconciliation
// du 2026-09-05 : deux instruments d'acceptation venus de l'amont (`ability_impulses_film_test.go`,
// `ability_charges_film_test.go`) rejouent la chaine de production sur un film designe par
// variable d'environnement, et l'amont leur donnait un `dir` parce que `BuildFromFilm` en prenait
// un. Le chargement est ici, dans un `_test.go`, ou le compilateur interdit qu'un chemin de
// production l'emprunte.
//
// Un repertoire illisible fait ECHOUER l'appel plutot que de construire sur un film nil : ces
// instruments comparent des sorties a des releves Theater, un document vide leur mentirait.
func buildFromFilmDir(matchID, titleSlug, dir string, opt Options) (ReplayDocument, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return ReplayDocument{}, err
	}
	return BuildFromFilm(matchID, titleSlug, film, opt)
}

// decodeFilmPlacementsDir : [decodeFilmPlacements] depuis un repertoire.
func decodeFilmPlacementsDir(
	dir string, wr *filmdec.Vec3Range,
) ([]filmdec.EquipmentPlacement, filmdec.EquipmentPlacementStats) {
	return decodeFilmPlacements(filmdec.NewFilmContext(filmDeDir(dir)), dir, wr)
}

// decodeFilmPadScansDir : [decodeFilmPadScans] depuis un repertoire.
func decodeFilmPadScansDir(dir string, wr *filmdec.Vec3Range, mpp filmdec.MPPWidths) PadScans {
	return decodeFilmPadScans(filmdec.NewFilmContext(filmDeDir(dir)), dir, wr, mpp)
}

// decodeFilmPadScanDir : [decodeFilmPadScan] depuis un repertoire.
func decodeFilmPadScanDir(
	dir string, wr *filmdec.Vec3Range, mpp filmdec.MPPWidths, arch padArchetype,
) WorldObjectScan {
	return decodeFilmPadScan(filmdec.NewFilmContext(filmDeDir(dir)), dir, wr, mpp, arch)
}
