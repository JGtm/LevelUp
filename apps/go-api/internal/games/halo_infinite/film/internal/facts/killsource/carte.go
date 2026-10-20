package killsource

// carte.go — LA CARTE DU MATCH EST OBLIGATOIRE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. » Les
// largeurs d axe du chemin absolu de position sont installees AU CHARGEMENT DE LA CARTE par le
// moteur et ne se lisent nulle part dans le film ; decoder un film aux largeurs d une AUTRE carte
// desynchronise la marche des morts, et le scan publie a sa place (enquete
// `ENQUETE_MARCHE_KILLSOURCE_2026-09-27`, bascule `f3a2f00eb`). C est un repli, et il est interdit.
//
// UN FILM SANS CARTE EST DONC MIS DE COTE, comme un film dont la cle est inconnue (D-4 d ADR 0034) :
// [Decode] rend [ErrCarteAbsente] avant toute lecture, l appelant de production le compte et le
// journalise, et le film reste candidat au rattrapage — il sera decode le jour ou sa carte le sera.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// carteApplicable : l entree de catalogue est-elle fournie ET porte-t-elle des largeurs ? C est LA
// definition de « carte lue » — elle se lit sur l ENTREE, jamais sur une difference de largeurs
// avec l invariant (sur Cliffhanger, l entree EST l invariant). Une largeur nulle est le signe d une
// entree anterieure au champ ou fabriquee a la main : `PoserLargeursObjetDuMondeDepuisDecoupage`
// l ignore et garde le defaut, donc elle ne decrit pas la carte.
func carteApplicable(carte *profile.MapQuantEntry) bool {
	if carte == nil {
		return false
	}
	for _, w := range carte.AxisWidths {
		if w == 0 {
			return false
		}
	}
	return true
}

// exigerLaCarte : la garde de [Decode]. [ErrCarteAbsente], enveloppee avec la cause, quand la carte
// manque — sauf si l appelant est un instrument de recherche qui le DECLARE.
func exigerLaCarte(o Options) error {
	if carteApplicable(o.Carte) || o.RechercheSansCarte {
		return nil
	}
	if o.Carte == nil {
		return fmt.Errorf("%w (aucune entree de catalogue)", ErrCarteAbsente)
	}
	return fmt.Errorf("%w (entree %q sans largeurs d axe)", ErrCarteAbsente, o.Carte.Module)
}
