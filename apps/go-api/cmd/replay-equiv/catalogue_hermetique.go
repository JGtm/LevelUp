package main

// catalogue_hermetique.go — LE CATALOGUE DES SOCLES DU HARNAIS NE DEPEND PAS DU POSTE.
//
// Les references figees ont ete cuites avec les points d'apparition d'une carte (Thunderhead)
// que le catalogue VERSIONNE ne porte pas : elle n'y entre que par l'OVERLAY non versionne du
// runtime (`PathResolver.MapWeaponPadsOverlayPath`), que chaque poste remplit a sa maniere. Sur
// un poste sans overlay, le film de cette carte rendait `map_absent` ; sur un poste qui l'avait
// rattrapee, 54 points : le meme depot donnait deux verdicts, selon un fichier que le depot ne
// voit pas.
//
// Le harnais lit donc SON catalogue : le versionne, plus une fixture versionnee a cote du code
// (`testdata/`). L'overlay du poste n'est jamais lu. La fixture n'est pas promue au catalogue
// versionne, ce qui changerait le catalogue de production pour un seul film.
//
// UNE FIXTURE ABSENTE EST UNE ERREUR, pas un cas nominal : `LoadMapWeaponPadsMerged` traite un
// overlay absent comme « rien rattrape » et rendrait sans un mot le catalogue versionne seul,
// donc un harnais redevenu dependant du poste sans que rien ne le dise.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// cheminFixtureSocles est la fixture d'overlay du harnais, relative a la racine du depot.
const cheminFixtureSocles = "apps/go-api/cmd/replay-equiv/testdata/map_weapon_pads_overlay.json"

// catalogueDesSocles rend le catalogue que cuit le harnais : versionne + fixture.
func catalogueDesSocles(ctx context.Context, repoRoot, titleSlug string) (*replay.MapWeaponPadsCatalog, error) {
	fixture := filepath.Join(repoRoot, filepath.FromSlash(cheminFixtureSocles))
	if _, err := os.Stat(fixture); err != nil {
		return nil, fmt.Errorf("fixture du catalogue des socles illisible (%s) : %w", fixture, err)
	}
	versionne := title.NewPathResolver(repoRoot).MapWeaponPadsPath(titleSlug)
	cat, err := replay.LoadMapWeaponPadsMerged(ctx, versionne, fixture)
	if err != nil {
		return nil, fmt.Errorf("catalogue des socles du harnais (titre %s) : %w", titleSlug, err)
	}
	return cat, nil
}
