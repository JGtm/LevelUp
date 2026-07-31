package killsource

// bijection.go — INDICE DE REPLICATION -> JOUEUR.
//
// Le dead-state porte deux indices sur 5 bits, pas des gamertags. Il faut donc une bijection, et
// elle ne doit PAS etre ajustee sur ses propres reponses.
//
// UN SEUL CHEMIN, quel que soit le nombre de joueurs : matrice SEPARABLE de votes indice ->
// joueur (chaque mort du feed dans la fenetre vote independamment pour la victime et pour le
// tueur), resolue EXACTEMENT par l algorithme hongrois, puis montee locale par transpositions sur
// le VRAI objectif — quadratique, non separable : le COUPLE doit etre juste. Redemarrages a
// graine FIXE.
//
// A 8 JOUEURS CE CHEMIN REND LA MEME BIJECTION QUE L ENUMERATION EXHAUSTIVE des 8! = 40 320
// permutations, score compris. C etait l exigence de non-regression pour le poser.
//
// LA MARGE EST LA QUANTITE QUI DECIDE DE CE QU ON A LE DROIT DE PUBLIER. A 8 joueurs le
// deuxieme meilleur score vaut EXACTEMENT le plancher mecanique (les couples justes n impliquant
// aucun des deux joueurs echanges) : la transposition ne recupere aucun couple conteste, la marge
// est large, les lignes sont publiables. En BTB elle est NULLE — au moins deux joueurs sont
// interchangeables — et alors SEUL L AGREGAT est publiable. C est [Result.LineByLinePublishable].
//
// RESERVE HISTORIQUE SUPPRIMEE, PAS NUANCEE : << une transposition preserve la quasi-totalite des
// couples >> est FAUSSE sur ses deux moities a 8 joueurs (les joueurs echanges se tuent entre eux,
// et la transposition perd 27 a 37 % des couples).
//
// LES SLOTS DE BOT SONT EPINGLES et sortent de l espace de recherche : voir roster.go.

import "sort"

// nearIndex : pour chaque candidat, les indices des morts du feed dans sa fenetre. Cette liste ne
// depend PAS de la permutation ; la calculer une fois rend la montee locale utilisable a
// 27 joueurs (351 transpositions par tour) sans rebalayer le feed a chaque essai.
type nearIndex struct {
	cs   []candidate
	near [][]int
}

func buildNear(pairs []feedEvent, cs []candidate) *nearIndex {
	n := &nearIndex{cs: cs, near: make([][]int, len(cs))}
	for i, c := range cs {
		for j := range pairs {
			dt := pairs[j].timeMS - c.ms
			if dt >= -tolMS && dt <= tolMS {
				n.near[i] = append(n.near[i], j)
			}
		}
	}
	return n
}

// quadScore : nombre de candidats dont le COUPLE (tueur, victime) se retrouve exactement au
// kill-feed sous la permutation. C est l objectif REEL.
func quadScore(nr *nearIndex, pairs []feedEvent, names []string, perm []int) int {
	n := 0
	for i, c := range nr.cs {
		if c.victim >= len(perm) || c.killer >= len(perm) {
			continue
		}
		v, k := names[perm[c.victim]], names[perm[c.killer]]
		for _, j := range nr.near[i] {
			if pairs[j].victim == v && pairs[j].killer == k {
				n++
				break
			}
		}
	}
	return n
}

// voteMatrix : la matrice SEPARABLE des votes (indice -> joueur), l initialisation lineaire.
func voteMatrix(nr *nearIndex, pairs []feedEvent, names []string, nPlay int) [][]int {
	idx := map[string]int{}
	for i, nm := range names {
		idx[nm] = i
	}
	m := make([][]int, nPlay)
	for i := range m {
		m[i] = make([]int, len(names))
	}
	for i, c := range nr.cs {
		for _, j := range nr.near[i] {
			addVote(m, idx, c.victim, pairs[j].victim, nPlay)
			addVote(m, idx, c.killer, pairs[j].killer, nPlay)
		}
	}
	return m
}

func addVote(m [][]int, idx map[string]int, slot int, name string, nPlay int) {
	if slot < 0 || slot >= nPlay {
		return
	}
	if p, ok := idx[name]; ok {
		m[slot][p]++
	}
}

// refine : montee locale par transpositions sur l objectif quadratique. Les positions EPINGLEES
// ne sont jamais echangees.
func refine(nr *nearIndex, pairs []feedEvent, names []string, perm, free []int) ([]int, int) {
	cur := append([]int(nil), perm...)
	best := quadScore(nr, pairs, names, cur)
	for {
		bi, bj, bs := -1, -1, best
		for a := 0; a < len(free); a++ {
			for b := a + 1; b < len(free); b++ {
				i, j := free[a], free[b]
				cur[i], cur[j] = cur[j], cur[i]
				if s := quadScore(nr, pairs, names, cur); s > bs {
					bi, bj, bs = i, j, s
				}
				cur[i], cur[j] = cur[j], cur[i]
			}
		}
		if bi < 0 {
			return cur, best
		}
		cur[bi], cur[bj] = cur[bj], cur[bi]
		best = bs
	}
}

// hungarianStart : affectation initiale sur le sous-probleme LIBRE, puis reinsertion des
// positions epinglees.
func hungarianStart(votes [][]int, r *roster, free, freeNames []int) []int {
	n := len(free)
	cost := make([][]int, n)
	for a := 0; a < n; a++ {
		cost[a] = make([]int, n)
		for b := 0; b < n; b++ {
			cost[a][b] = -votes[free[a]][freeNames[b]]
		}
	}
	sub := hungarian(cost)
	perm := make([]int, r.nPlay)
	for i, p := range r.pin {
		perm[i] = p
	}
	for a := 0; a < n; a++ {
		perm[free[a]] = freeNames[sub[a]]
	}
	return perm
}

// solveBijection : LE chemin unique. Rend la bijection et son score quadratique.
func solveBijection(r *roster, pairs []feedEvent, cs []candidate, restarts int) ([]int, int) {
	nr := buildNear(pairs, cs)
	free, freeNames := r.freeSlots()
	if len(free) == 0 {
		perm := make([]int, r.nPlay)
		for i, p := range r.pin {
			perm[i] = p
		}
		return perm, 0
	}
	votes := voteMatrix(nr, pairs, r.names, r.nPlay)
	bestPerm, bestScore := refine(nr, pairs, r.names, hungarianStart(votes, r, free, freeNames), free)
	rng := newRNG()
	for i := 0; i < restarts; i++ {
		p := make([]int, r.nPlay)
		for k, v := range r.pin {
			p[k] = v
		}
		sh := rng.Perm(len(freeNames))
		for a, k := range free {
			p[k] = freeNames[sh[a]]
		}
		pp, s := refine(nr, pairs, r.names, p, free)
		if s > bestScore || (s == bestScore && permLess(pp, bestPerm)) {
			bestPerm, bestScore = pp, s
		}
	}
	return bestPerm, bestScore
}

// permLess : ordre total sur les permutations. Il ne sert QU A departager les ex aequo de facon
// deterministe — sans lui, deux executions du meme code publieraient deux bijections differentes
// de meme score, et la non-regression serait invérifiable.
func permLess(a, b []int) bool {
	for i := range a {
		if i >= len(b) {
			return false
		}
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// bijectionMargin : le meilleur score atteignable en imposant UNE transposition depuis l optimum.
// ZERO = au moins deux joueurs sont interchangeables : les attributions ligne par ligne ne sont
// alors PAS publiables, seul l agregat l est.
func bijectionMargin(r *roster, pairs []feedEvent, cs []candidate, best int) int {
	nr := buildNear(pairs, cs)
	free, _ := r.freeSlots()
	cur := append([]int(nil), r.perm...)
	second := -1
	for a := 0; a < len(free); a++ {
		for b := a + 1; b < len(free); b++ {
			i, j := free[a], free[b]
			cur[i], cur[j] = cur[j], cur[i]
			if s := quadScore(nr, pairs, r.names, cur); s > second {
				second = s
			}
			cur[i], cur[j] = cur[j], cur[i]
		}
	}
	if second < 0 {
		return 0
	}
	return best - second
}

// hungarian : affectation de COUT MINIMAL (O(n^3), version a potentiels). On lui passe l oppose
// des votes pour maximiser. Rend `perm[i]` = colonne affectee a la ligne i.
func hungarian(cost [][]int) []int {
	n := len(cost)
	const inf = 1 << 60
	u, v := make([]int, n+1), make([]int, n+1)
	p, way := make([]int, n+1), make([]int, n+1)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]int, n+1)
		used := make([]bool, n+1)
		for j := range minv {
			minv[j] = inf
		}
		for {
			used[j0] = true
			j1 := hungarianStep(cost, u, v, p, way, minv, used, j0, n)
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}
	perm := make([]int, n)
	for j := 1; j <= n; j++ {
		if p[j] > 0 {
			perm[p[j]-1] = j - 1
		}
	}
	return perm
}

// hungarianStep : une etape de relaxation. Extraite pour tenir le seuil de taille de fonction ;
// la logique est celle de l algorithme de reference, sans variante.
func hungarianStep(cost [][]int, u, v, p, way, minv []int, used []bool, j0, n int) int {
	const inf = 1 << 60
	i0, delta, j1 := p[j0], inf, 0
	for j := 1; j <= n; j++ {
		if used[j] {
			continue
		}
		cur := cost[i0-1][j-1] - u[i0] - v[j]
		if cur < minv[j] {
			minv[j], way[j] = cur, j0
		}
		if minv[j] < delta {
			delta, j1 = minv[j], j
		}
	}
	for j := 0; j <= n; j++ {
		if used[j] {
			u[p[j]] += delta
			v[j] -= delta
		} else {
			minv[j] -= delta
		}
	}
	return j1
}

// sortCandidates : ordre stable des candidats, pour que deux passes rendent la meme chose.
func sortCandidates(cs []candidate) {
	sort.Slice(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		switch {
		case a.chunk != b.chunk:
			return a.chunk < b.chunk
		case a.pidx != b.pidx:
			return a.pidx < b.pidx
		default:
			return a.bit < b.bit
		}
	})
}
