package replayverite

// images convertit une duree en nombre d'images entieres du document, au pas qu'il publie. La
// conversion arrondit vers le bas : une tolerance ne s'elargit jamais, et un seuil de N pas exacts
// rend N a l'identique (200 ms valent 2 images a 100 ms, 6 a 33 ms).
//
// AUCUN REPLI SUR UN PAS ABSENT : [LireDocument] refuse un artefact dont `frameIntervalMs` n'est pas positif, donc
// tout document lu en a un. Un Document construit a la main sans pas rend 0 image (aucune tolerance), comme V-1 ne
// juge rien sans pas.
func (d *Document) images(ms int) int {
	if d.FrameIntervalMs <= 0 {
		return 0
	}
	return ms / d.FrameIntervalMs
}
