package replayartifacts

// raster_refaire.go — le fichier tactique d'un match refait SANS FILM, depuis le rejeu déjà rangé (la lecture en place du
// fichier tactique est dans `replayartifact`, `LireSidecarRasterCourant` et `SidecarRasterCourantRapide`).
//
// Deux appelants se servent de ces gestes : le rattrapage en ligne de commande (`levelup tactical-rasters --backfill`) et
// la file de régénération du bureau, qui refait le fichier tactique d'un rejeu dont le film a quitté l'archive (il n'y a
// plus de quoi refaire le rejeu, mais le fichier tactique ne dépend que du rejeu rangé).

import (
	"context"
)

// RefaireRasterTactique projette le rejeu rangé à artefactPath et dépose le fichier tactique à sidecarPath. Aucun film
// n'est lu, aucune base n'est ouverte, aucune cuisson n'est lancée : le rejeu rangé n'est pas touché.
// previsionArmee : le titre déclare `film.spawn_forecast` (voir [ProjeterRasterTactique]).
func RefaireRasterTactique(ctx context.Context, artefactPath, sidecarPath string, previsionArmee bool) error {
	s, err := ProjeterRasterTactique(ctx, artefactPath, previsionArmee)
	if err != nil {
		return err
	}
	return EcrireSidecarRaster(ctx, sidecarPath, s)
}
