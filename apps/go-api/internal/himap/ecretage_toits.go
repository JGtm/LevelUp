package himap

// ecretage_toits.go — ENLEVER LES TOITS, PAS LES CHOISIR.
//
// CE QUE LA VOIE DE REFERENCE FAIT DEJA, ET SA LIMITE. `AppliqueReference` SUBSTITUE : sous un
// toit, elle montre le sol qui est dessous. C'est juste quand il y a un sol — et ca ne fait
// rien quand il n'y en a pas. Un hangar plein, une passerelle au-dessus du vide, une casquette
// de beton : la substitution n'a rien a proposer, le toit reste, et la carte se lit comme une
// vue de dessus de la ville plutot que du terrain. Elle ne se declenche d'ailleurs que
// au-dela de `SeuilCarteCouverte` : sur Streets, mesure a 7,1 %, elle ne fait rien du tout.
//
// L'ECRETAGE VA PLUS LOIN, ET ASSUME DE SUPPRIMER DE LA MATIERE. Un pixel dont AUCUNE surface
// n'est a hauteur de jeu ne montre pas du terrain : il montre un couvercle. On le vide.
// La silhouette qui reste est le PLAN DE L'ARENE.
//
// CE QU'EST UN TOIT, ET C'EST LA SEULE DEFINITION UTILISEE ICI : une surface a plus de
// `PlafondArene` AU-DESSUS de la reference locale de son pixel. La reference vient des ancres
// d'objectifs (`SurfaceReference`), donc de la carte elle-meme — pas d'un seuil d'altitude
// absolu, qui n'a aucun sens sur des cartes dont le niveau joue va de -136 m a +77 m.
//
// TROIS CAS PAR PIXEL, dans cet ordre :
//
//	surface haute a hauteur de jeu        -> on la garde (c'est le terrain)
//	surface haute trop haute, sol dessous -> on montre le sol (substitution)
//	surface haute trop haute, rien dessous-> on VIDE (c'est un couvercle nu)
//
// PIEGE DEJA PAYE, ecrit pour ne pas etre rejoue : sur une carte dont les rochers hauts font
// partie de l'identite (Cliffhanger, validee par l'utilisateur avec ses rochers), l'ecretage
// les EFFACE. Il n'est donc pas universel et ne doit pas etre arme par defaut : c'est un
// reglage PAR CARTE, declare en donnee avec sa raison, comme l'habillage et l'echelle.

import "math"

// PlafondArene : hauteur, en metres, au-dela de la reference locale, a partir de laquelle une
// surface cesse d'etre tenue pour un etage joue.
//
// 6 m = `EcartPlafondMin` (2 m, un Spartan tient debout sous un plafond) + `TolSolReference`
// (3 m, les sols en pente d'une arene) + 1 m de marge. Mesure d'appui (sonde du 2026-08-13) :
// le surplomb median AU DROIT DES ANCRES vaut 7,2 a 18,0 m sur les cartes jugees couvertes —
// les couvercles vivent au-dela, les etages joues en deca.
const PlafondArene = 6.0

// EcretteToits applique l'ecretage decrit en tete de fichier et rend ce qu'il a fait :
// la part de matiere couverte (meme mesure que `AppliqueReference`, pour que les deux voies
// se comparent), le nombre de pixels dont la surface a ete substituee, et le nombre de pixels
// VIDES.
//
// Comme `AppliqueReference`, elle libere la voie de reference en sortie : la decision est
// prise, les buffers n'ont plus d'objet. Les deux ne s'appellent donc PAS l'une apres l'autre.
func (r *Rendu) EcretteToits(s *SurfaceReference) (taux float64, substituees, videes int) {
	if r.zRef == nil || s.Vide() {
		return 0, 0, 0
	}
	defer func() { r.ref, r.dRef, r.zRef, r.nRef = nil, nil, nil, nil }()
	taux = r.tauxCouverture()
	for k := range r.z {
		if math.IsInf(r.z[k], -1) || math.IsNaN(r.ref[k]) {
			continue
		}
		if r.z[k]-r.ref[k] <= PlafondArene {
			continue // la surface haute est a hauteur de jeu : c'est le terrain.
		}
		// Un sol existe-t-il sous ce toit, assez pres de la reference pour etre praticable ?
		if !math.IsNaN(r.zRef[k]) && r.zRef[k]-r.ref[k] <= PlafondArene {
			r.z[k], r.n[k] = r.zRef[k], r.nRef[k]
			substituees++
			continue
		}
		// Rien a hauteur de jeu sous ce pixel : couvercle nu, on le vide.
		r.z[k] = math.Inf(-1)
		videes++
	}
	return taux, substituees, videes
}
