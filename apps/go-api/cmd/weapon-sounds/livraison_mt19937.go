package main

// livraison_mt19937.go — Mersenne Twister MT19937, PORT BIT A BIT du generateur pseudo-
// aleatoire de CPython (module `random`), pour le seul appel qui en depend dans le mode
// `livrer` : `tirerCoup` choisit une variante de couche par `random.Random(20260816).choice`
// (cf. `_outils/coups_lot.py:tirerCoup`, appele par `_outils/livraison.py:rendreEvent`).
//
// POURQUOI UN PORT EXACT ET PAS `math/rand`. Le mode `livrer` doit produire un fichier
// IDENTIQUE OCTET A OCTET a celui du script Python pour l'arme rendue par evenement
// (Covenant_provoker -> hinf_ravager) : `math/rand` de Go n'implemente PAS le meme
// algorithme que CPython et rendrait une sequence de choix differente, donc un melange de
// couches different, donc un fichier different. Verifie sur pieces (scratchpad, comparaison
// avec `python3 -c "import random; ..."`) : les tirages de `genrand_uint32` bruts ET la
// sequence `Random(20260816).choice(n)` sur plusieurs longueurs consecutives correspondent
// EXACTEMENT a l'implementation ci-dessous.
//
// ALGORITHME : Matsumoto & Nishimura, reference `mt19937ar.c` (`init_by_array` +
// `genrand_uint32` avec temperage), la meme que `_randommodule.c` de CPython. Graine ENTIERE
// POSITIVE tenant sur 32 bits (notre seul cas d'usage) : CPython decoupe l'entier en mots de
// 32 bits, poids faible en tete — pour 20260816 (< 2^32) cela donne une cle a un seul mot,
// directement la graine elle-meme.

const (
	mtN       = 624
	mtM       = 397
	mtMatrixA = 0x9908b0df
	mtUpper   = 0x80000000
	mtLower   = 0x7fffffff
)

// mt19937 : l'etat du generateur. Pas de mutex — un seul rendu utilise une instance, jamais
// partagee entre goroutines.
type mt19937 struct {
	state [mtN]uint32
	index int
}

// newMT19937FromSeed cree un generateur seede comme `random.Random(seed)` pour une graine
// ENTIERE POSITIVE tenant sur 32 bits.
func newMT19937FromSeed(seed uint32) *mt19937 {
	m := &mt19937{}
	m.initByArray([]uint32{seed})
	return m
}

func (m *mt19937) initGenrand(s uint32) {
	m.state[0] = s
	for i := 1; i < mtN; i++ {
		prev := m.state[i-1]
		m.state[i] = 1812433253*(prev^(prev>>30)) + uint32(i)
	}
	m.index = mtN
}

// initByArray reproduit `init_by_array` de la reference MT19937 (utilisee par CPython des
// que la graine ne tient pas sur un seul appel a `init_genrand`, et en pratique toujours,
// meme pour une cle a un seul mot).
func (m *mt19937) initByArray(key []uint32) {
	m.initGenrand(19650218)
	i, j := 1, 0
	k := mtN
	if len(key) > k {
		k = len(key)
	}
	for ; k > 0; k-- {
		prev := m.state[i-1]
		m.state[i] = (m.state[i] ^ ((prev ^ (prev >> 30)) * 1664525)) + key[j] + uint32(j)
		i++
		j++
		if i >= mtN {
			m.state[0] = m.state[mtN-1]
			i = 1
		}
		if j >= len(key) {
			j = 0
		}
	}
	for k = mtN - 1; k > 0; k-- {
		prev := m.state[i-1]
		m.state[i] = (m.state[i] ^ ((prev ^ (prev >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= mtN {
			m.state[0] = m.state[mtN-1]
			i = 1
		}
	}
	m.state[0] = 0x80000000
}

// next rend le prochain mot de 32 bits tempere (genrand_uint32).
func (m *mt19937) next() uint32 {
	if m.index >= mtN {
		m.genererBloc()
	}
	y := m.state[m.index]
	m.index++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}

// genererBloc recalcule les 624 mots d'un coup, exactement comme la reference C.
func (m *mt19937) genererBloc() {
	mag01 := [2]uint32{0, mtMatrixA}
	for kk := 0; kk < mtN-mtM; kk++ {
		y := (m.state[kk] & mtUpper) | (m.state[kk+1] & mtLower)
		m.state[kk] = m.state[kk+mtM] ^ (y >> 1) ^ mag01[y&1]
	}
	for kk := mtN - mtM; kk < mtN-1; kk++ {
		y := (m.state[kk] & mtUpper) | (m.state[kk+1] & mtLower)
		m.state[kk] = m.state[kk+(mtM-mtN)] ^ (y >> 1) ^ mag01[y&1]
	}
	y := (m.state[mtN-1] & mtUpper) | (m.state[0] & mtLower)
	m.state[mtN-1] = m.state[mtM-1] ^ (y >> 1) ^ mag01[y&1]
	m.index = 0
}

// getrandbits reproduit `Random.getrandbits(k)` pour k <= 32 (notre seul cas d'usage : les
// longueurs de couches candidates ne depassent jamais quelques dizaines d'elements).
func (m *mt19937) getrandbits(k int) uint32 {
	return m.next() >> uint(32-k)
}

// bitLen reproduit `int.bit_length()`.
func bitLen(n int) int {
	b := 0
	for n > 0 {
		b++
		n >>= 1
	}
	return b
}

// randbelow reproduit `Random._randbelow_with_getrandbits(n)`.
func (m *mt19937) randbelow(n int) int {
	if n <= 0 {
		return 0
	}
	k := bitLen(n)
	r := int(m.getrandbits(k))
	for r >= n {
		r = int(m.getrandbits(k))
	}
	return r
}

// choice reproduit `Random.choice(seq)` restreint a l'indice choisi dans `[0, n)` — a
// l'appelant d'indexer sa propre sequence, exactement comme `dispo[rng.choice(len(dispo))]`
// dans coups_lot.py appelle `seq[self._randbelow(len(seq))]`.
func (m *mt19937) choice(n int) int {
	return m.randbelow(n)
}
