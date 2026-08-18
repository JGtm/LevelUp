package replay

// attachement_phase0_controle_test.go — ITEM 0.1, VOLET DE CONTRÔLE.
//
// POURQUOI CE CONTRÔLE PRÉCÈDE LES DEUX ORACLES, ET POURQUOI IL EST PLUS DUR À TROMPER.
// Confronter directement un champ d'i10 à l'oracle CTF empile trois hypothèses (le champ est
// un handle, l'entité lue est le drapeau, le calage d'horloge est bon) : un échec ne dirait
// pas laquelle a cédé. Ici on n'en teste qu'UNE — un handle d'entité pointe sur une entité
// qui EXISTE. Le World tient la table des entités vivantes ; le test est donc exact, et son
// témoin (un slot décorrélé, même World, même appel) donne le taux du hasard sur place.
//
// LES COUPLES (archétype de l'enfant -> archétype du parent) SONT LE RÉSULTAT UTILE : c'est
// eux, et non un taux, qui répondent à la question du plan. `42 -> 35` serait le drapeau tenu
// par un bipède ; `35 -> 40`, le Spartan dans son véhicule.

import (
	"sort"
	"testing"
)

// TestAttachementPhase0Parents — ITEM 0.1, VOLET DE CONTRÔLE : le champ candidat désigne-t-il
// une ENTITÉ VIVANTE, et laquelle ?
//
// POURQUOI CE CONTRÔLE PRÉCÈDE LES DEUX ORACLES, ET POURQUOI IL EST PLUS DUR À TROMPER.
// Confronter directement le champ à l'oracle CTF empile trois hypothèses (le champ est un
// handle, l'entité lue est le drapeau, le calage d'horloge est bon) : un échec ne dirait pas
// laquelle a cédé. Ici on n'en teste qu'UNE — un handle d'entité pointe sur une entité qui
// EXISTE. Le World tient la table des entités vivantes ; le test est donc exact, et son
// témoin (un slot décorrélé, même World, même appel) donne le taux du hasard sur place.
//
// LES COUPLES (archétype de l'enfant -> archétype du parent) SONT LE RÉSULTAT UTILE : c'est
// eux, et non un taux, qui répondent à la question du plan. `42 -> 35` serait le drapeau
// tenu par un bipède ; `35 -> 40`, le Spartan dans son véhicule.
func TestAttachementPhase0Parents(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	for _, id := range attTousFilms() {
		if _, ok := objOpenFilm(t, root, id); !ok {
			continue
		}
		joues++
		lectures, st := attScanOf(t, root, id)
		t.Logf("%s : %d paquets delta dont %d BIT-EXACTS (%.1f %%) · %d lectures d'i10 dont "+
			"%d venues d'un paquet bit-exact",
			id, st.Paquets, st.PaquetsPropres, 100*attPart(st.PaquetsPropres, st.Paquets),
			st.Lectures, st.LecturesPropres)
		attLogParents(t, id, lectures, false)
		attLogParents(t, id, lectures, true)
	}
	if joues == 0 {
		t.Skipf("aucun film du corpus dans le cache (%s=%q)", attFilmEnv, root)
	}
}

// attLogParents publie le contrôle « le handle désigne-t-il une entité vivante » sur un
// sous-ensemble de lectures : toutes, ou les seules venues d'un paquet bit-exact.
func attLogParents(t *testing.T, id string, lectures []attI10, propresSeules bool) {
	t.Helper()
	etiquette := "TOUTES"
	if propresSeules {
		etiquette = "BIT-EXACTES"
	}
	total, att, lies, temoins, cumulLies := 0, 0, 0, 0, 0
	couples := map[[2]uint32]int{}
	for _, l := range lectures {
		if propresSeules && !l.Propre {
			continue
		}
		total++
		if !l.St.Attached {
			continue
		}
		att++
		cumulLies += l.Lies
		if l.TemoinLie {
			temoins++
		}
		if !l.ParentLie {
			continue
		}
		lies++
		couples[[2]uint32{l.TI, l.ParentTI}]++
	}
	moy := cumulLies / max(att, 1)
	t.Logf("%s : [%s] %d lectures dont %d ATTACHÉES (%.1f %%) · handle -> entité vivante "+
		"%d (%.1f %%) · témoin (slot décalé de %d) %d (%.1f %%) · %d entités liées en moyenne "+
		"sur %d slots (taux du hasard %.1f %%)",
		id, etiquette, total, att, 100*attPart(att, total), lies, 100*attPart(lies, att),
		attDecalageTemoin, temoins, 100*attPart(temoins, att), moy, attSlotMask+1,
		100*attPart(moy, int(attSlotMask)+1))
	attLogCouples(t, id+" ["+etiquette+"]", couples)
}

// attLogCouples publie les couples (archétype de l'enfant, archétype du parent) par
// fréquence décroissante.
func attLogCouples(t *testing.T, id string, couples map[[2]uint32]int) {
	t.Helper()
	type ligne struct {
		enfant, parent uint32
		n              int
	}
	var ls []ligne
	for k, n := range couples {
		ls = append(ls, ligne{k[0], k[1], n})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].n != ls[j].n {
			return ls[i].n > ls[j].n
		}
		if ls[i].enfant != ls[j].enfant {
			return ls[i].enfant < ls[j].enfant
		}
		return ls[i].parent < ls[j].parent
	})
	for i, l := range ls {
		if i >= 12 {
			t.Logf("%s :   ... %d autres couples", id, len(ls)-12)
			break
		}
		t.Logf("%s :   enfant ti=%-3d -> parent ti=%-3d : %d", id, l.enfant, l.parent, l.n)
	}
}
