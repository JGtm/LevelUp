package replaybuild

// artifact_purge.go — RETIRER LES FICHIERS DE REJEU D'UN MATCH, et eux seuls.
//
// Un match a, sur disque, trois fichiers de rejeu : l'artefact (`<short8>.json`), sa marque de
// dérivation (`<short8>.derived.json`) et le sidecar de raster tactique (`rasters/<short8>.json`).
// Le film (`film_chunks/`, `film_manifests/`) et les faits persistés (`film_facts/`) ne sont PAS
// des fichiers de rejeu : ils ne sont jamais touchés ici (le film expire côté serveur Halo,
// l'archive est irremplaçable).
//
// Premier appelant : `levelup backfill-replay --purge-non-geres`, qui retire les artefacts déjà
// cuits d'un mode que le rejeu ne gère pas ([ErrModeNonGere]).

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	titlePkg "levelup/go-api/internal/domain/title"
)

// PurgeMatchFiles retire les fichiers de rejeu d'UN match et rend les chemins qui existaient
// (supprimés, ou qui l'auraient été si `dryRun`). Un fichier absent n'est pas une erreur ; la
// première suppression en échec arrête le match et remonte l'erreur nommée.
func PurgeMatchFiles(pr *titlePkg.PathResolver, titleSlug, matchID string, dryRun bool) ([]string, error) {
	artefact := pr.ReplayArtifactPath(titleSlug, matchID)
	candidats := []string{artefact, DerivationsMarkPath(artefact), pr.TacticalRasterPath(titleSlug, matchID)}
	var touches []string
	for _, path := range candidats {
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return touches, fmt.Errorf("purge du rejeu %s : %w", path, err)
		}
		if info.IsDir() {
			continue
		}
		if !dryRun {
			if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return touches, fmt.Errorf("purge du rejeu %s : %w", path, err)
			}
		}
		touches = append(touches, path)
	}
	return touches, nil
}
