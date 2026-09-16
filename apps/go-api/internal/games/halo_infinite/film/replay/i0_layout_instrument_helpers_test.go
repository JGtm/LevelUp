package replay

// i0_layout_instrument_helpers_test.go — LA DETECTION DU DECOUPAGE D i0 DEPUIS UN REPERTOIRE,
// POUR LES INSTRUMENTS DE CE PAQUET.
//
// # POURQUOI CETTE COPIE EXISTE (revue de jalon M1, constat C4, 2026-09-16)
//
// `detecterI0Layout(dir)` etait une enveloppe EXPORTEE et de PRODUCTION dont aucun
// appelant de production ne subsistait depuis le lot 1.9.4 : ses 48 appels etaient tous des
// tests. Elle est devenue `detectI0Layout`, non exportee, dans un fichier de test de `grammar`
// (regle 7 du depot : « 0 code mort »).
//
// Un symbole de test n est pas importable d un autre paquet. Les huit instruments de `replay`
// qui l appelaient passent donc par cette fonction-ci, qui refait ses deux lignes : charger le
// film, puis appeler la forme de production [grammar.DetectI0LayoutOf], qui prend un film deja
// charge. Deux copies d un appel de deux lignes restent sous le plafond de la regle 6 du depot ;
// l alternative — garder une enveloppe en production pour que des tests l atteignent — est
// precisement la dette que ce deplacement solde.
//
// PAS DE TAG `research` : ce fichier ne lit rien de lui-meme, il charge ce que son appelant lui
// nomme. Les instruments appelants portent leur propre garde (helper partage dans un fichier NON
// tague, regle du 2026-09-16, §2.3 du plan).

import (
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// detecterI0Layout lit le decoupage d i0 DANS le film de `dir`.
func detecterI0Layout(dir string) (grammar.I0Layout, grammar.I0LayoutReport, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return grammar.I0Layout{}, grammar.I0LayoutReport{}, err
	}
	return grammar.DetectI0LayoutOf(film)
}
