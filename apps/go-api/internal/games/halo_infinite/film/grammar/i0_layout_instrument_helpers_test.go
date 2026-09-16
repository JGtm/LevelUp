package grammar

// i0_layout_instrument_helpers_test.go — L ENVELOPPE `dir` DE LA DETECTION DU DECOUPAGE D i0 :
// UN INSTRUMENT, PAS UNE LECTURE DE PRODUCTION.
//
// # POURQUOI CE CODE EST DANS UN FICHIER `_test.go` (revue de jalon M1, constat C4, 2026-09-16)
//
// `detectI0Layout(dir)` chargeait un film ENTIER pour appeler [DetectI0LayoutOf]. Elle etait
// declaree en PRODUCTION alors que son dernier appelant de production a disparu au lot 1.9.4
// (2026-09-15, `DetectFilmMapEntry` supprimee : la carte d un match vient de son NOM, plus d une
// signature de largeurs d axe). Inventaire sur pieces du 2026-09-16 :
//
//	grep -rn "detectI0Layout(" --include=*.go . | grep -v "DetectI0LayoutOf("
//	  -> 48 appels, TOUS dans des `_test.go` ; 0 appel de production.
//
// Regle 7 du depot (« 0 code mort ») : ce qu on debranche du routing sort de la production. Les
// deux lignes vivent donc ici, sous le nom non exporte `detectI0Layout`, et la production garde
// la seule forme qu elle emploie — [DetectI0LayoutOf], sur un film deja charge.
//
// # LES HUIT APPELANTS DU PAQUET `replay` ONT LEUR PROPRE COPIE, ET C EST LE PRIX A PAYER
//
// Un symbole de test n est pas importable d un autre paquet. `replay` porte donc les memes deux
// lignes dans `replay/i0_layout_instrument_helpers_test.go`. Deux copies d un appel de deux
// lignes restent sous le plafond de la regle 6 du depot (« <= 2 copies »), et l alternative —
// garder une enveloppe exportee en production pour que des tests d un autre paquet l appellent —
// est exactement la dette que ce deplacement solde.
//
// # PAS DE TAG `research`
//
// Ce fichier ne lit rien par lui-meme : il ne fait que charger ce que son appelant lui nomme. Ce
// sont les instruments APPELANTS qui ouvrent des films et qui portent leur propre garde (helper
// partage dans un fichier NON tague, regle du 2026-09-16, §2.3 du plan).

import (
	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// detectI0Layout lit le decoupage d i0 DANS le film de `dir`. Rend le decoupage, le rapport de
// mesure, et une erreur si le profil ne fait pas apparaitre trois frontieres nettes (film trop
// court, ou grammaire de record differente).
func detectI0Layout(dir string) (profile.I0Layout, I0LayoutReport, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return profile.I0Layout{}, I0LayoutReport{}, err
	}
	return DetectI0LayoutOf(film)
}
