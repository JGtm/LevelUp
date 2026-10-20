package grammar

// marche_trames_preliminaires.go — LES PRELIMINAIRES DE LA MARCHE DES TRAMES, LUS DANS LA PHASE DES
// IMAGES-CLES (ADR 0037 IR-3 ; lot 2.2 du plan de l etape 2).
//
// La marche des trames lie le monde depuis les images-cles avant d en marcher les trames : la table
// anticipee de tout le film ([TableAnticipee]), posee avant la premiere trame, puis chunk par chunk ce
// que les images-cles du chunk declarent ([liaisonDesImagesCles]). Les deux se lisent dans UNE phase
// des images-cles — celle de la distribution ([Distribuer]), ou celle que la marche des trames joue
// seule ([FilmContext.Trames]) — au lieu de deux parcours de la marche d ancres. Aucun corps n y est
// parcouru : la table et la liaison ne lisent que des ancres, et la table de datums de chaque
// image-cle.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// preliminairesDesTrames porte ce que la marche des trames lit des images-cles avant ses trames.
type preliminairesDesTrames struct {
	table   *TableAnticipee
	liaison liaisonDesImagesCles
}

// nouveauxPreliminaires rend des preliminaires vides pour le film de `fc`.
func nouveauxPreliminaires(fc *FilmContext) *preliminairesDesTrames {
	return &preliminairesDesTrames{table: nouvelleTableAnticipeeDe(fc),
		liaison: liaisonDesImagesCles{parChunk: map[int][]declarationDImageCle{}}}
}

// lireLesPreliminaires joue la phase des images-cles pour les seuls preliminaires de la marche des
// trames : aucun canal, aucun corps parcouru.
func (c *FilmContext) lireLesPreliminaires() *preliminairesDesTrames {
	prel := nouveauxPreliminaires(c)
	distribuerLesImagesCles(c, demandeDImagesCles{}, &MarcheDistribuee{EnTete: c.EnTete()}, nil, prel)
	return prel
}

// recevoir verse le paquet d image-cle `p`, de marche d ancres [MarcheDistribuee.Ancres], dans la
// table et dans la liaison. Sans preliminaires (nil), rien.
func (pr *preliminairesDesTrames) recevoir(p *lecture.Paquet, m *MarcheDistribuee) {
	if pr == nil {
		return
	}
	pr.table.ajouterImageCle(p)
	pr.liaison.recevoir(p, m.Ancres)
}

// clore clot la table une fois la phase des images-cles finie. Sans preliminaires (nil), rien.
func (pr *preliminairesDesTrames) clore() {
	if pr == nil {
		return
	}
	pr.table.Clore()
}
