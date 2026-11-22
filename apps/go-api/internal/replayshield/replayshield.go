// Package replayshield rend les vies d'un document de rejeu sous la forme que lit
// `analysis/shieldcycle`, qui n'importe pas le décodeur de film (garde-rail
// `TestFermetureDeAnalysisSansDecodeurDeFilm`, dans `archlint/no_title_package_in_analysis_test.go`).
//
// C'est le seul passage « document -> mesures de bouclier » : la dérivation du résumé d'usage et la
// vue du rejeu l'appellent toutes deux, pour qu'un bris n'ait qu'une définition.
package replayshield

import (
	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/analysis/shieldcycle"
)

// Lives rend une vie par piste du document, avec les mesures `Sh` de ses points dans l'ordre du
// film. Un point sans mesure est ignoré ; une piste sans mesure rend une vie sans mesure, que
// [shieldcycle.Cycles] n'enregistre pas. Un document nil ne rend aucune vie.
func Lives(doc *replay.ReplayDocument) []shieldcycle.Life {
	if doc == nil {
		return nil
	}
	lives := make([]shieldcycle.Life, 0, len(doc.Tracks))
	for i := range doc.Tracks {
		tr := &doc.Tracks[i]
		l := shieldcycle.Life{XUID: tr.XUID}
		for j := range tr.Points {
			if sh := tr.Points[j].Sh; sh != nil {
				l.Measures = append(l.Measures, shieldcycle.Measure{Frame: tr.Points[j].T, Shield: *sh})
			}
		}
		lives = append(lives, l)
	}
	return lives
}

// FrameMS convertit une image du document en millisecondes de l'axe du document (image fois
// intervalle d'image). Pour l'horloge « début du film », l'appelant ajoute l'origine du document.
func FrameMS(frame, frameIntervalMS int) int {
	return frame * frameIntervalMS
}
