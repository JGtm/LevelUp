package powerpos

// ponderation.go — LE RANG DU TUEUR COMME POIDS D'UNE ELIMINATION (D10a du plan des
// positions de force, 2026-09-20).
//
// # LE PROBLEME QUE CA TRAITE
//
// Notre corpus n'est pas un corpus professionnel : il est fait des matchs d'une poignee de
// joueurs, de niveaux tres inegaux (constat de l'utilisateur, 2026-09-20). Une position de
// force est ce que des joueurs QUI SAVENT JOUER cherchent a tenir ; un kill obtenu par un
// joueur qui se trouvait la par hasard dit moins du terrain qu'un kill obtenu par un joueur
// qui a choisi son poste.
//
// # LA FORME RETENUE : UNE RAMPE, PAS UN COUPERET
//
// Restreindre le corpus aux tueurs au-dessus d'un rang jetterait la majorite des
// eliminations — et sur un corpus qui compte 3 000 a 8 000 kills par carte, diviser par
// trois ramenerait le disque de 2 m sous le seuil d'engagements. La ponderation garde tout
// l'echantillon et le PENCHE : poids `PoidsBas` au rang bas de reference, `PoidsHaut` au
// rang haut, lineairement entre les deux, borne en dehors.
//
// LE RANG INCONNU PESE 1,0, ET CE N'EST PAS UN DEFAUT PAR DEPIT. Le rang par match et par
// joueur (`match_csrs_latest`) n'existe QUE pour les matchs classes : en playlist sociale
// aucun joueur du lobby n'a de CSR en base. Donner un poids bas a un rang inconnu
// reviendrait a effacer les playlists sociales du corpus ; leur donner un poids haut, a les
// privilegier. Le poids neutre est la seule valeur qui ne decide rien.
//
// LA VARIANTE « AU-DESSUS DE LA MEDIANE » EST MESUREE A PART, jamais melangee : le
// comptage `KillsTueurFort` compte les eliminations dont le tueur a un rang CONNU et
// superieur ou egal a la mediane du corpus. Elle sert a verifier si la rampe capture
// vraiment quelque chose, sans avoir a rejouer la passe.

// Ponderation convertit un rang de tueur en poids d'elimination. Les bornes sont mesurees
// SUR LE CORPUS par l'appelant (quantiles des rangs vus), jamais devinees : l'echelle CSR
// s'etale de 0 a plus de 1 800 selon la saison et la playlist.
type Ponderation struct {
	// RangBas / RangHaut : les deux rangs de reference (en pratique p10 et p90 du corpus).
	RangBas  float64 `json:"rang_bas"`
	RangHaut float64 `json:"rang_haut"`
	// RangMedian : la mediane du corpus, qui separe les tueurs « forts » des autres.
	RangMedian float64 `json:"rang_median"`

	// PoidsBas / PoidsHaut : les poids aux deux bornes.
	PoidsBas  float64 `json:"poids_bas"`
	PoidsHaut float64 `json:"poids_haut"`
	// PoidsInconnu : le poids d'une elimination dont le rang du tueur est absent.
	PoidsInconnu float64 `json:"poids_inconnu"`
}

// PonderationNeutre rend une ponderation qui ne penche rien : tout pese 1,0. C'est le
// defaut, et c'est ce qui garantit qu'un accumulateur construit sans rang se comporte
// exactement comme avant l'introduction de ce fichier.
func PonderationNeutre() Ponderation {
	return Ponderation{PoidsBas: 1, PoidsHaut: 1, PoidsInconnu: 1}
}

// Active dit si la ponderation penche quelque chose.
func (p Ponderation) Active() bool {
	return p.RangHaut > p.RangBas && p.PoidsHaut != p.PoidsBas
}

// Poids rend le poids d'une elimination dont le tueur avait ce rang. Un rang absent (nil)
// ou une ponderation inactive rendent le poids neutre.
func (p Ponderation) Poids(rang *float64) float64 {
	if rang == nil {
		if p.PoidsInconnu > 0 {
			return p.PoidsInconnu
		}
		return 1
	}
	if !p.Active() {
		return 1
	}
	t := (*rang - p.RangBas) / (p.RangHaut - p.RangBas)
	switch {
	case t < 0:
		t = 0
	case t > 1:
		t = 1
	}
	return p.PoidsBas + t*(p.PoidsHaut-p.PoidsBas)
}

// EstFort dit si le rang est CONNU et au moins egal a la mediane du corpus. Un rang absent
// n'est pas fort — et il n'est pas faible non plus : il n'entre simplement pas dans ce
// comptage, qui est une VARIANTE de mesure et non un filtre du score.
func (p Ponderation) EstFort(rang *float64) bool {
	return rang != nil && p.RangMedian > 0 && *rang >= p.RangMedian
}
