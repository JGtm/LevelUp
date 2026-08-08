package main

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// LA REGRESSION, ET POURQUOI CE N'EST PAS UN BALAYAGE DE FRONTIERES.
//
// Un balayage bit a bit cherche une distribution qui « a l'air bonne » et en tire une
// frontiere : il fabrique toujours une reponse, le chantier se l'interdit. Ici la cible n'est
// pas une forme, c'est une EGALITE NUMERIQUE avec une position vraie, et la statistique est
// bornee : un champ candidat juste rend une relation affine (R2 -> 1) entre l'entier lu et la
// coordonnee interpolee ; un champ faux rend du bruit. Trois gardes accompagnent le chiffre :
// le controle positif (retrouver les champs CONNUS de la branche basse), la nulle par
// permutation, et la distribution complete des R2 sur tous les candidats.

// candidate est un champ suppose : offset en bits depuis le debut du composant, largeur.
type candidate struct{ off, w int }

// fitResult porte le resultat d'une regression p = a*q + b.
type fitResult struct {
	cand   candidate
	r2     float64
	a, b   float64
	n      int
	minQ   uint64
	maxQ   uint64
	extent float64 // a * 2^w : l'etendue implicite de la plage, si le champ est juste
}

// runRegression confronte chaque champ candidat a la position vraie. hi=false rejoue la
// mecanique sur la branche BASSE : c'est le controle positif, il DOIT retrouver 3/13, 16/13,
// 29/14.
func runRegression(all []sample, wr *filmdec.Vec3Range, hi bool) {
	mode := "CONTROLE POSITIF (branche basse : les champs sont connus)"
	total := lowPosBits
	if hi {
		mode = "BRANCHE OPAQUE (porte = 1)"
		total = hiPosBits
	}
	fmt.Printf("=== %s ===\n", mode)

	pairs := collectPairs(all, hi)
	if len(pairs) < 30 {
		fmt.Printf("ECHANTILLON INSUFFISANT : %d records encadres. Rien a conclure — et surtout\n"+
			"pas un negatif sur la question : c'est un negatif sur la COUVERTURE.\n\n", len(pairs))
		return
	}
	fmt.Printf("echantillon : %d records encadres par deux positions vraies\n\n", len(pairs))

	// TROIS REGIMES, ET LE TROISIEME EST LE PLUS PLAUSIBLE DES TROIS.
	//
	// Les deux premiers supposent que la branche opaque encode une position ABSOLUE — dans
	// une plage fixe, ou aux largeurs de la carte. Le troisieme suppose qu'elle encode un
	// DELTA depuis la position precedente, et c'est ce qu'un protocole de replication fait
	// quand il annonce « haute precision » : on n'envoie que le petit changement. La plage
	// +-100 de `DAT_143b8c6d0`, absurde pour une position absolue sur une carte de 113 unites,
	// devient tres naturelle pour un DEPLACEMENT par image.
	//
	// Le depot connait deja ce motif : `components_movement.go` modelise un chemin delta a
	// dequantification CENTREE-ZERO pour l'autre i0 (`q` dans [0, 2^W) recentre sur
	// [-2^(W-1), 2^(W-1))). Une regression affine le retrouve sans avoir a le supposer : le
	// recentrage est une constante additive, qu'elle absorbe dans l'ordonnee a l'origine.
	for _, r := range []int{regimeWorld, regimeNorm, regimeDelta} {
		fmt.Printf("-- regime : %s\n", regimeLabel(r))
		runOneRegime(pairs, wr, total, r)
	}
}

const (
	regimeWorld = iota // position absolue, coordonnee monde -> teste une plage FIXE
	regimeNorm         // position absolue, coordonnee normalisee -> teste les largeurs de la CARTE
	regimeDelta        // DEPLACEMENT depuis le record precedent -> teste un encodage delta
)

func regimeLabel(r int) string {
	switch r {
	case regimeNorm:
		return "position ABSOLUE, coordonnee NORMALISEE [0,1] (teste les largeurs de la CARTE)"
	case regimeDelta:
		return "DEPLACEMENT depuis la position precedente (teste un encodage DELTA)"
	default:
		return "position ABSOLUE, coordonnee MONDE (teste une plage FIXE : +-100, +-20000)"
	}
}

func runOneRegime(pairs []pair, wr *filmdec.Vec3Range, total int, regime int) {
	var cands []candidate
	for w := 10; w <= 24; w++ {
		for off := 1; off+w <= total; off++ {
			cands = append(cands, candidate{off, w})
		}
	}

	for a := 0; a < 3; a++ {
		axis := a + 3*regime
		axisName := []string{"X", "Y", "Z"}[a]
		results := make([]fitResult, 0, len(cands))
		for _, c := range cands {
			if r, ok := fitOne(pairs, axis, c); ok {
				results = append(results, r)
			}
		}
		if len(results) == 0 {
			fmt.Printf("axe %s : aucun candidat exploitable\n", axisName)
			continue
		}
		sort.Slice(results, func(i, j int) bool { return results[i].r2 > results[j].r2 })

		// La NULLE : la distribution des R2 de TOUS les candidats. Un gagnant qui n'est pas
		// aberrant devant elle n'est pas un gagnant.
		r2s := make([]float64, len(results))
		for i, r := range results {
			r2s[i] = r.r2
		}
		sort.Float64s(r2s)
		med := r2s[len(r2s)/2]
		q99 := r2s[int(float64(len(r2s))*0.99)]

		fmt.Printf("axe %s — %d candidats. NULLE : mediane R2 %.4f, q99 %.4f\n", axisName, len(results), med, q99)
		for i := 0; i < 5 && i < len(results); i++ {
			r := results[i]
			fmt.Printf("   off %2d  w %2d   R2 %.6f   pente %.6g   etendue implicite %.2f   min..max %d..%d\n",
				r.cand.off, r.cand.w, r.r2, r.a, r.extent, r.minQ, r.maxQ)
		}
		best := results[0]
		perm := permutationNull(pairs, axis, best.cand)
		fmt.Printf("   nulle par permutation sur le gagnant : R2 %.6f (attendu ~0)\n", perm)
		if best.r2 < 0.90 {
			fmt.Printf("   VERDICT axe %s : AUCUN champ ne rend la position. R2 max %.4f (q99 de sa propre\n"+
				"                   distribution : %.4f — le gagnant n'est donc PAS aberrant).\n",
				axisName, best.r2, q99)
		} else {
			ref := float64(wr[a].Max - wr[a].Min)
			switch regime {
			case regimeNorm:
				ref = 1
			case regimeDelta:
				ref = 200 // l'etendue de DAT_143b8c6d0, +-100
			}
			fmt.Printf("   VERDICT axe %s : champ off %d largeur %d, R2 %.6f. Etendue implicite %.2f "+
				"(reference : %.2f)\n", axisName, best.cand.off, best.cand.w, best.r2, best.extent, ref)
		}
		fmt.Println()
	}
}

// collectPairs rend, pour chaque record de la branche visee, ses 64 bits d'i0 et la position
// vraie interpolee depuis les records de la branche BASSE qui l'encadrent dans la meme vie.
//
// Sur la branche basse (controle positif) le record cible est EXCLU de son propre
// encadrement : sinon on regresserait une valeur sur elle-meme.
func collectPairs(all []sample, hi bool) []pair {
	var out []pair
	for _, l := range groupLives(all) {
		for i, s := range l {
			if s.hi != hi {
				continue
			}
			v, nv, ok := bracket(l, i)
			if !ok {
				continue
			}
			prev, ok := prevLowPos(l, i)
			if !ok {
				continue
			}
			p := pair{bits: s.bits}
			for a := 0; a < 3; a++ {
				p.val[a] = float64(v[a])
				p.val[a+3] = float64(nv[a])
				p.val[a+6] = float64(v[a] - prev[a]) // le DEPLACEMENT depuis la position precedente
			}
			out = append(out, p)
		}
	}
	return out
}

// pair porte les 64 bits d'i0 et la cible de la regression : 0..2 position absolue en
// coordonnee MONDE, 3..5 position absolue NORMALISEE dans l'AABB de la carte du film,
// 6..8 DEPLACEMENT depuis la derniere position connue.
type pair struct {
	bits uint64
	val  [9]float64
}

// prevLowPos rend la position du dernier record de la branche BASSE qui precede l'indice i
// dans la vie. C'est la reference du regime DELTA.
func prevLowPos(l []sample, i int) ([3]float32, bool) {
	for j := i - 1; j >= 0; j-- {
		if !l[j].hi {
			return l[j].pos, true
		}
	}
	return [3]float32{}, false
}

// fieldAt extrait le champ (off, w) des 64 bits d'i0. PeekBits range le PREMIER bit lu en
// poids fort : le bit de position de flux `at+i` est le bit 63-i du mot.
func fieldAt(bits uint64, off, w int) uint64 {
	return (bits >> uint(64-off-w)) & ((uint64(1) << uint(w)) - 1)
}

// fitOne regresse la coordonnee vraie sur l'entier brut du champ candidat.
func fitOne(pairs []pair, axis int, c candidate) (fitResult, bool) {
	n := len(pairs)
	var sx, sy, sxx, sxy, syy float64
	minQ, maxQ := ^uint64(0), uint64(0)
	for _, p := range pairs {
		q := fieldAt(p.bits, c.off, c.w)
		if q < minQ {
			minQ = q
		}
		if q > maxQ {
			maxQ = q
		}
		x := float64(q)
		y := p.val[axis]
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		syy += y * y
	}
	fn := float64(n)
	den := fn*sxx - sx*sx
	if den == 0 {
		return fitResult{}, false // champ constant : rien a regresser
	}
	a := (fn*sxy - sx*sy) / den
	b := (sy - a*sx) / fn
	var ssRes, ssTot float64
	meanY := sy / fn
	for _, p := range pairs {
		x := float64(fieldAt(p.bits, c.off, c.w))
		y := p.val[axis]
		e := y - (a*x + b)
		ssRes += e * e
		ssTot += (y - meanY) * (y - meanY)
	}
	if ssTot == 0 {
		return fitResult{}, false
	}
	return fitResult{
		cand: c, r2: 1 - ssRes/ssTot, a: a, b: b, n: n,
		minQ: minQ, maxQ: maxQ,
		extent: math.Abs(a) * math.Exp2(float64(c.w)),
	}, true
}

// permutationNull rejoue la regression du gagnant en appariant les entiers bruts a des
// positions vraies TIREES D'AUTRES records. Si le R2 tient encore, c'est que la statistique
// mesure autre chose que le lien cherche.
func permutationNull(pairs []pair, axis int, c candidate) float64 {
	if len(pairs) < 4 {
		return 0
	}
	shifted := make([]pair, len(pairs))
	for i := range pairs {
		shifted[i] = pair{bits: pairs[i].bits, val: pairs[(i+len(pairs)/2)%len(pairs)].val}
	}
	r, ok := fitOne(shifted, axis, c)
	if !ok {
		return 0
	}
	return r.r2
}
