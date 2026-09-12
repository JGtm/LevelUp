package replay

// visee_composant_score_test.go — LOT F : LA TRANSPOSITION EN COLONNES ET LE SCORE.
//
// SEPARE DU MOTEUR DE COLLECTE POUR UNE RAISON DE LECTURE : ce fichier ne connait plus ni film,
// ni paquet, ni composant — il ne voit que des vecteurs de bits dates et une onde. Tout ce qui
// touche au format du film est dans `visee_composant_research_test.go` ; tout ce qui touche a la
// statistique est ici. Les seuils, eux, sont declares une seule fois, dans l en-tete du moteur.
//
// LE SCORE ET SON CONTROLE SONT CEUX DU LOT C (`visee_onde_research_test.go`), reutilises et non
// reecrits : `ondeCol` (transposition), `ondeMasque` (classes d un decalage), `evalue`
// (exactitude equilibree). Les deux seules fonctions redefinies ici — `vfMeilleur` et
// `vfClassement` — le sont parce que leurs equivalents du lot C sautent les sept premiers bits
// du payload : un offset RELATIF au composant n a aucune raison de commencer a 7.

import (
	"fmt"
	"sort"
)

// vfColonne porte le balayage d'UN composant : ses records, ses offsets relatifs transposes en
// colonnes de bits, et la dispersion de sa largeur.
type vfColonne struct {
	idx     int
	nom     string
	col     *ondeCol
	n       int
	offsets int
	largMin int
	largMax int
}

// vfBatColonnes construit une table de colonnes par composant.
//
// LES OFFSETS S'ARRETENT A LA LARGEUR MINIMALE OBSERVEE, et c'est une decision de methode : un
// composant a portes n'a pas la meme longueur d'un record a l'autre, donc au-dela du prefixe
// commun l'offset o ne designerait plus le meme champ dans deux records. Le prefixe commun est
// exactement la partie ou l'hypothese « bit a offset relatif fixe » a un sens.
func vfBatColonnes(recs []vfRecord, echMin int) []vfColonne {
	parComp := map[int][]vfComp{}
	temps := map[int][]int64{}
	for _, r := range recs {
		for _, c := range r.comps {
			parComp[c.idx] = append(parComp[c.idx], c)
			temps[c.idx] = append(temps[c.idx], r.tMS)
		}
	}
	idxs := make([]int, 0, len(parComp))
	for id := range parComp {
		idxs = append(idxs, id)
	}
	sort.Ints(idxs)
	var out []vfColonne
	for _, id := range idxs {
		cs := parComp[id]
		if len(cs) < echMin {
			continue
		}
		vc := vfColonne{idx: id, nom: cs[0].nom, n: len(cs), largMin: cs[0].larg, largMax: cs[0].larg}
		for _, c := range cs {
			if c.larg < vc.largMin {
				vc.largMin = c.larg
			}
			if c.larg > vc.largMax {
				vc.largMax = c.larg
			}
		}
		vc.offsets = vc.largMin
		if vc.offsets > vfOffsetMax {
			vc.offsets = vfOffsetMax
		}
		if vc.offsets <= 0 {
			continue
		}
		vc.col = vfTranspose(temps[id], cs, vc.offsets)
		out = append(out, vc)
	}
	return out
}

// vfTranspose range les bits d'un composant en colonnes indexees par offset relatif.
func vfTranspose(temps []int64, cs []vfComp, offsets int) *ondeCol {
	c := &ondeCol{temps: temps, mots: (len(cs) + 63) / 64, nbits: offsets}
	c.col = make([][]uint64, offsets)
	for b := range c.col {
		c.col[b] = make([]uint64, c.mots)
	}
	for i, comp := range cs {
		mot, bit := i/64, uint(i%64)
		for o := 0; o < offsets; o++ {
			if comp.bits>>(63-uint(o))&1 == 1 {
				c.col[o][mot] |= 1 << bit
			}
		}
	}
	return c
}

// vfMeilleur rend le meilleur score du composant pour un masque de classes donne, sur TOUS ses
// offsets et les deux polarites. Il repart de l'offset 0 — c'est la difference avec le
// `meilleur` du lot C, qui saute les 7 premiers bits parce qu'ils portaient la tete du paquet.
func vfMeilleur(c *ondeCol, m ondeMasque) ondeScore {
	best := ondeScore{pos: -1}
	for b := 0; b < c.nbits; b++ {
		ba, tp, fp := c.evalue(b, m)
		s, pol := ba, 1
		if 1-ba > s {
			s, pol = 1-ba, -1
		}
		if s > best.score {
			best = ondeScore{pos: b, polarite: pol, score: s, tp: tp, fp: fp}
		}
	}
	return best
}

// vfClassement rend tous les offsets d'un composant, tries par score decroissant. Comme
// vfMeilleur, il repart de l'offset 0 — le `classement` du lot C saute les 7 premiers bits.
func vfClassement(c *ondeCol, m ondeMasque) []ondeScore {
	out := make([]ondeScore, 0, c.nbits)
	for b := 0; b < c.nbits; b++ {
		ba, tp, fp := c.evalue(b, m)
		s, pol := ba, 1
		if 1-ba > s {
			s, pol = 1-ba, -1
		}
		out = append(out, ondeScore{pos: b, polarite: pol, score: s, tp: tp, fp: fp})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// vfNomCommandTick est l'etiquette de registre du composant que la phase 7 designe : la
// commande du tick, dont l'octet 6 porte l'etat de zoom cote moteur.
const vfNomCommandTick = "unit-command-tick-component"

// vfDomaine est un sous-ensemble DECLARE de composants (cf. l'en-tete du moteur).
type vfDomaine struct {
	nom     string
	retient func(vfColonne) bool
}

// vfDomainesDe rend les domaines a mesurer : les deux domaines larges declares, puis UN DOMAINE
// PAR COMPOSANT.
//
// POURQUOI UN DOMAINE PAR COMPOSANT. La puissance se paie au nombre d'hypotheses : sur les 138
// couples du domaine complet, un decalage temoin atteint 1,0000 dans 4,75 % des cas, donc le
// negatif global n'y vaut rien (execution 6). Restreint a UN composant, le meme instrument
// retrouve une puissance de 0,00 % — et rend alors un verdict qui tient. Chaque composant a
// donc son p(max) ET sa puissance, et le lecteur voit lesquels concluent.
func vfDomainesDe(cols []vfColonne) []vfDomaine {
	out := []vfDomaine{
		{"D1 COMPLET (le mandat : tous les composants atteints)", func(vfColonne) bool { return true }},
		{"D3 ETAT (hors i0 position et i1 velocite : grandeurs spatiales continues)",
			func(c vfColonne) bool { return c.idx != 0 && c.idx != 1 }},
	}
	for _, c := range cols {
		id, nom := c.idx, c.nom
		etiquette := fmt.Sprintf("i%d %s", id, nom)
		if nom == vfNomCommandTick {
			etiquette += " [D2 : DESIGNE par la phase 7]"
		}
		out = append(out, vfDomaine{etiquette, func(x vfColonne) bool { return x.idx == id }})
	}
	return out
}

// vfMonoComposant dit qu'un domaine ne porte qu'un composant : sa publication est abregee.
func vfMonoComposant(sub []vfColonne) bool { return len(sub) == 1 }

// vfFiltre rend le sous-ensemble de colonnes d'un domaine.
func vfFiltre(cols []vfColonne, d vfDomaine) []vfColonne {
	var out []vfColonne
	for _, c := range cols {
		if d.retient(c) {
			out = append(out, c)
		}
	}
	return out
}

// vfPointe est le meilleur couple (composant, offset) d'un decalage donne.
type vfPointe struct {
	comp    int
	nom     string
	score   ondeScore
	n1, n0  int
	retenus int // composants recevables a ce decalage (S6)
}

// vfBalaye evalue tous les composants a un decalage donne et rend la pointe globale.
func vfBalaye(cols []vfColonne, o ondeCarree, delta int64, echMin int) vfPointe {
	out := vfPointe{comp: -1}
	for _, vc := range cols {
		m := vc.col.marque(o, delta)
		if m.n1 < echMin || m.n0 < echMin {
			continue
		}
		out.retenus++
		s := vfMeilleur(vc.col, m)
		if s.pos >= 0 && s.score > out.score.score {
			out.comp, out.nom, out.score, out.n1, out.n0 = vc.idx, vc.nom, s, m.n1, m.n0
		}
	}
	return out
}
