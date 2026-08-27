package main

// variantes_conditionnelles.go — CE QUE LE PARCOURS ECARTE, ET SOUS QUELLE CONDITION.
//
// LE PARCOURS D'EVENEMENT APPLIQUE DEUX FILTRES, chacun justifie par une mesure anterieure :
// un `Switch` ne rend que l'etat par defaut, un `Blend` a courbes ne rend que les enfants
// audibles AU POINT DE REFERENCE (parametre de jeu a sa valeur minimale). Les deux sont des
// choix defendables pour un rejeu 2D qui ne pilote aucun parametre de jeu.
//
// MAIS ILS RENDENT DES SONS INVISIBLES, ET LA MESURE LE DIT :
//
//	dcfaa487 (translocateur)  2 orphelins, les DEUX PLUS LONGS sons de la banque (6,77 s et
//	                          6,22 s), sous un `Blend` pilote par le parametre 3236399890 :
//	                          ils entrent a x = 0,797 et dominent a x = 0,939.
//	61007dcf (drapeau)        2 orphelins, meme cause, sur deux evenements distincts.
//	6fd78d85 (bobine a choc)  16 orphelins — et les trois autres banques de bobine en ont
//	                          SEIZE aussi — sous un `Switch` de groupe 2275666646, etat
//	                          163696720 au lieu du defaut 1093928064.
//
// Un enfant inaudible a x minimal et audible plus loin n'est pas un dechet : c'est une
// PHASE du geste. L'utilisateur decrit exactement cela pour le translocateur — « c'est comme
// si on le chargeait, ca monte en intensite, et ensuite il est pose ». La montee est le
// parametre qui croit ; la pose est la phase haute.
//
// Ce fichier ne change pas ce que le parcours rend. Il rend VISIBLE ce qu'il ecarte, avec la
// condition nommee, pour que le rendu puisse servir la phase manquante au lieu de la taire.

import (
	"fmt"
	"sort"
)

// varianteConditionnelle : un sous-arbre que le parcours ecarte, avec la condition sous
// laquelle le jeu le joue, et les medias qu'il porte.
type varianteConditionnelle struct {
	Noeud     string             `json:"noeud"`
	TypeNoeud string             `json:"type_noeud"`
	Condition string             `json:"condition"`
	Wems      []uint32           `json:"wems"`
	Gains     map[string]float32 `json:"gains_db,omitempty"`
}

// variantesConditionnelles rend, pour un evenement, les sous-arbres ecartes par les filtres.
//
// La descente suit les listes d'enfants BRUTES : c'est la seule vue qui permette de voir un
// lien coupe. Des qu'un lien est coupe, le sous-arbre entier est rattache a la variante et
// la descente ne le suit pas deux fois.
func (b *bank) variantesConditionnelles(idEvent uint32) []varianteConditionnelle {
	var out []varianteConditionnelle
	vus := map[uint32]bool{}
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }

	var descendre func(n uint32, gain float64)
	descendre = func(n uint32, gain float64) {
		if vus[n] {
			return
		}
		vus[n] = true
		o, ok := b.Objets[n]
		if !ok {
			return
		}
		gain += float64(b.VolNoeud[n])
		retenus := map[uint32]bool{}
		for _, e := range b.Enfants[n] {
			retenus[e] = true
		}
		for _, e := range lireEnfants(o.Data, connu) {
			if retenus[e] {
				descendre(e, gain)
				continue
			}
			s := evaluerSaut(b, n, e)
			wems, gains := b.sousArbreBrut(e, gain)
			if len(wems) == 0 {
				continue
			}
			out = append(out, varianteConditionnelle{
				Noeud: fmt.Sprintf("%08x", n), TypeNoeud: nomType(o.Type),
				Condition: s.Pourquoi, Wems: wems, Gains: gains,
			})
		}
	}
	for _, idAction := range b.Events[idEvent] {
		if cible, ok := b.Actions[idAction]; ok {
			descendre(cible, 0)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Noeud < out[j].Noeud })
	return out
}

// sousArbreBrut collecte les medias d'un sous-arbre en suivant les listes d'enfants BRUTES,
// et le gain de chemin de chacun. Un sous-arbre ecarte n'est pas filtre a son tour : on
// rend ce qu'il contient, la condition de sa racine suffit a le qualifier.
func (b *bank) sousArbreBrut(racine uint32, gain float64) ([]uint32, map[string]float32) {
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }
	wems := map[uint32]float64{}
	vus := map[uint32]bool{}
	var descendre func(uint32, float64)
	descendre = func(n uint32, g float64) {
		if vus[n] {
			return
		}
		vus[n] = true
		o, ok := b.Objets[n]
		if !ok {
			return
		}
		g += float64(b.VolNoeud[n])
		if w, estSon := b.Sons[n]; estSon {
			if _, deja := wems[w]; !deja {
				wems[w] = g
			}
			return
		}
		for _, e := range lireEnfants(o.Data, connu) {
			descendre(e, g)
		}
	}
	descendre(racine, gain)

	ids := make([]uint32, 0, len(wems))
	gains := map[string]float32{}
	for w, g := range wems {
		ids = append(ids, w)
		if g != 0 {
			gains[fmt.Sprintf("%d", w)] = float32(g)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(gains) == 0 {
		gains = nil
	}
	return ids, gains
}
