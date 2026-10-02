//go:build research

package replay

// instruments_partages_film_shims_research_test.go — adaptateurs de film_shims_test.go dont les seuls utilisateurs sont des instruments research (J12.7 lint : inutilises dans le build par defaut). Deplacement pur.

import (
	"context"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// decodeFilmPadScansDir : [decodeFilmPadScans] depuis un repertoire.
func decodeFilmPadScansDir(dir string, wr *profile.Vec3Range, mpp profile.MPPWidths) PadScans {
	return decodeFilmPadScans(context.Background(), grammar.NewFilmContext(filmDeDir(dir)), dir, wr, mpp)
}

// decodeFilmPadScanDir : [decodeFilmPadScan] depuis un repertoire.
func decodeFilmPadScanDir(
	dir string, wr *profile.Vec3Range, mpp profile.MPPWidths, arch padArchetype,
) WorldObjectScan {
	return decodeFilmPadScan(context.Background(), grammar.NewFilmContext(filmDeDir(dir)), dir, wr, mpp, arch)
}
