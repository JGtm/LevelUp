package filmdec

// ti47_annonces_largeur_test.go — LES TROIS MESURES DE LARGEUR de l'instrument ti=47
// (plan `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`, item 0.1). Le balayage est dans
// `ti47_annonces_scan_test.go`, les rapports dans `ti47_annonces_test.go`.
//
// POURQUOI TROIS ET NON UNE. La recette du depot lit les largeurs dans le descripteur `+0x28` du
// binaire du jeu ; sans instance Ghidra, ce lot ne peut pas les LIRE. Il les MESURE — et une
// mesure unique se serait fait piéger, parce que la charge utile de ce composant contient un
// motif qui RESSEMBLE a un en-tete de record (mesure du lot : un « en-tete valide » demarre au
// bit +1 dans 71 a 99 % des cas, sur tous les films). Les trois mesures ne tombent pas dans le
// meme piege :
//
//	1. CHAINAGE (ti47Chain) — a quels decalages un en-tete valide demarre. Large, mais sensible
//	   aux motifs internes : c'est elle qui voit le faux pic a +1.
//	2. CIBLE RESTREINTE — le meme test, mais en n'acceptant qu'un slot de la bande ti=47. Le faux
//	   pic tombe a zero (le motif interne ne pointe pas un objet de cet archetype) ; la vraie fin
//	   de record survit quand deux objets de l'archetype emettent d'affilee.
//	3. DISTANCE (ti47Moisson.ecart) — l'ecart entre les DEBUTS de deux records consecutifs de la
//	   bande. C'est la mesure directe : taille = en-tete + index + largeur, sans aucune hypothese
//	   sur ce qui suit. Elle exige seulement deux records dans le meme paquet.
//
// Et une quatrieme, de refutation : LONGUEUR DE CHAINE (mesureRuns) — une fausse largeur peut
// marquer un pic, elle ne peut pas enchainer deux records de suite, parce que l'arrivee est
// fausse d'autant a chaque pas.
//
// LE TEMOIN POSITIF EST DANS LE MEME ARCHETYPE : `i1 splash-message-dynamic` est porte depuis le
// lot 0 et vaut R(24). Les quatre mesures doivent le retrouver, sur chaque film, avant qu'on
// accorde le moindre credit a ce qu'elles disent de `i2`.

import "sort"

// ti47Chain est un histogramme de chainage : pour chaque decalage d en bits, combien de fois un
// en-tete de record valide demarre a `depart + d`, sur combien de tentatives possibles.
type ti47Chain struct{ hits, denom [ti47MaxDecalage + 1]int }

// mesure teste tous les decalages depuis `at` et incremente l'histogramme.
func (c *ti47Chain) mesure(pay []byte, at int, cible map[uint32]bool) {
	total := len(pay) * 8
	for d := 1; d <= ti47MaxDecalage; d++ {
		p := at + d
		if p+worldObjectHeaderBits+worldObjectIndexBits > total {
			return
		}
		c.denom[d]++
		if _, ok := matchWorldObjectRecord(pay, p, cible); ok {
			c.hits[d]++
		}
	}
}

// taux rend le taux de chainage au decalage d, ou -1 quand aucune tentative n'a eu lieu.
func (c *ti47Chain) taux(d int) float64 {
	if d < 1 || d > ti47MaxDecalage || c.denom[d] == 0 {
		return -1
	}
	return float64(c.hits[d]) / float64(c.denom[d])
}

// plancher rend le taux MEDIAN sur tous les decalages : le niveau de hasard de ce film, mesure
// sur les memes octets que le signal. Un pic se juge contre lui, jamais contre un pourcentage
// ecrit d'avance (regle de redaction des gates, registre du 18/08).
func (c *ti47Chain) plancher() float64 {
	var v []float64
	for d := 1; d <= ti47MaxDecalage; d++ {
		if t := c.taux(d); t >= 0 {
			v = append(v, t)
		}
	}
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	return v[len(v)/2]
}

// sommets rend les n decalages au plus fort taux, tries par taux decroissant.
func (c *ti47Chain) sommets(n int) []int {
	d := make([]int, 0, ti47MaxDecalage)
	for i := 1; i <= ti47MaxDecalage; i++ {
		if c.taux(i) >= 0 {
			d = append(d, i)
		}
	}
	sort.Slice(d, func(x, y int) bool { return c.taux(d[x]) > c.taux(d[y]) })
	if len(d) > n {
		d = d[:n]
	}
	return d
}

// ti47Singleton rend l'index annonce quand le masque n'en porte qu'un, -1 sinon.
func ti47Singleton(rec WorldObjectRecord) int {
	if len(rec.Idx) == 1 {
		return rec.Idx[0]
	}
	return -1
}

// ecart range la distance entre le debut du record precedent de la bande et celui-ci.
func (m *ti47Moisson) ecart(precDebut, precIdx, debut int, rec WorldObjectRecord) {
	if precDebut < 0 || precIdx < 0 {
		return
	}
	d := debut - precDebut
	if d <= 0 {
		return
	}
	if d > ti47MaxEcart {
		m.ecartsHors[precIdx]++
		return
	}
	if m.ecarts[precIdx] == nil {
		m.ecarts[precIdx] = make([]int, ti47MaxEcart+1)
		m.suivants[precIdx] = map[int]int{}
	}
	m.ecarts[precIdx][d]++
	m.suivants[precIdx][ti47Singleton(rec)]++
}

// mesureRuns compte, pour chaque largeur candidate, combien de records SINGLETON {i2} de la
// bande s'enchainent d'affilee depuis ce record.
func (m *ti47Moisson) mesureRuns(pay []byte, at int) {
	if len(m.runs) == 0 {
		return
	}
	total := len(pay) * 8
	for w, hist := range m.runs {
		p, k := at+w, 0
		for k < ti47MaxRun {
			if p+worldObjectHeaderBits+worldObjectIndexBits > total {
				break
			}
			rec, ok := matchWorldObjectRecord(pay, p, m.b.band)
			if !ok || len(rec.Idx) != 1 || rec.Idx[0] != m.b.iPersonal {
				break
			}
			k++
			p = rec.After + w
		}
		hist[k]++
	}
}
