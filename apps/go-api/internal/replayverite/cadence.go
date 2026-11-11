package replayverite

// cadenceHistoriqueMS : la duree d'une image d'un artefact qui ne publie pas `frameIntervalMs`
// (les schemas anterieurs au champ). 100 ms est la grille du rejeu depuis sa creation.
const cadenceHistoriqueMS = 100

// images convertit une duree en nombre d'images entieres du document, au pas qu'il publie. La
// conversion arrondit vers le bas : une tolerance ne s'elargit jamais, et un seuil de N pas exacts
// rend N a l'identique (200 ms valent 2 images a 100 ms, 6 a 33 ms). Un document sans pas se lit a
// [cadenceHistoriqueMS].
func (d *Document) images(ms int) int {
	pas := d.FrameIntervalMs
	if pas <= 0 {
		pas = cadenceHistoriqueMS
	}
	return ms / pas
}
