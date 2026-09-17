package killsource

// equivalence_lecteur_test.go — L EQUIVALENCE BIT A BIT, APPEL PAR APPEL (lot 2.4.1).
//
// # LA METHODE, ET POURQUOI ELLE SUFFIT
//
// Toute la consommation de bits de la chaine d evenements passe par QUATRE primitives : `rd`,
// `g1` (qui est `rd(1)`), `skip`, et les lectures par position du paquet (`bitAt`, `bits32`,
// `bitsN`, plus `bitsWide` sous `rd`). Si chacune rend la MEME valeur, avance le curseur de la
// MEME distance et leve son drapeau de debordement DANS LES MEMES CAS, alors la marche entiere
// est identique par recurrence. C est ce que ce fichier prouve ; le golden des triplets
// (`chaines_evenements_test.go`) le confirme ensuite de bout en bout.
//
// LES COPIES DE REFERENCE ci-dessous sont les implantations d AVANT le lot 2.4.1, recopiees ici
// et NULLE PART AILLEURS — la production n en garde aucune (`evReader`, `bitsWide`, `bitAt`,
// `bits32`, `bitsN` ont ete supprimes dans le commit qui ajoute ce fichier). Meme patron que
// `filmdec/bits_word_test.go`.
//
// LES POSITIONS SONT LES POSITIONS REELLES DE LA CHAINE (arbitrage V15 (3)), pas un balayage
// aveugle : le depart de la liste d evenements, chaque ancre du generateur de kill-events, la
// fin des champs de chaque kill-event, et les bornes de chaque evenement enchaine. On y AJOUTE
// les quatre-vingts derniers bits de chaque paquet, parce que la FIN DE FLUX est precisement
// l ecart que l absorption devait traiter : c est la que `evReader` refusait la ou le lecteur
// canonique bourre a zero, et une equivalence qui ne visiterait pas ce bord ne prouverait rien
// de l ecart qu elle ferme.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// --- Copies de reference : les implantations d AVANT le lot 2.4.1 (oracles du differentiel) ---

// refEvReader est `killsource.evReader` d avant : tampon propre, position propre, REFUS de lire
// au-dela du paquet.
type refEvReader struct {
	pl   []byte
	bp   int
	over bool
}

func (r *refEvReader) rd(n int) uint64 {
	if n <= 0 {
		return 0
	}
	if r.bp+n > len(r.pl)*8 {
		r.over = true
		return 0
	}
	v := refBitsWide(r.pl, r.bp, n)
	r.bp += n
	return v
}

func (r *refEvReader) skip(n int) {
	if n <= 0 {
		return
	}
	if r.bp+n > len(r.pl)*8 {
		r.over = true
		return
	}
	r.bp += n
}

// refBitsWide est `bitsWide` d avant : boucle bit a bit, sans chemin par mot.
func refBitsWide(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = v<<1 | uint64(refBitAt(d, bp+i))
	}
	return v
}

// refBitAt est `bitAt` d avant : hors tampon = 0, des DEUX cotes.
func refBitAt(d []byte, p int) int {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return int(d[p>>3]>>uint(7-(p&7))) & 1
}

// refBits32 est `bits32` d avant : CINQ octets accumules puis decales, ce qui n est pas la meme
// boucle que la lecture par mot — l equivalence devait donc etre prouvee pour elle aussi.
func refBits32(d []byte, p int) uint32 {
	i, sh := p>>3, uint(p&7)
	var v uint64
	for k := 0; k < 5; k++ {
		v <<= 8
		if i+k < len(d) {
			v |= uint64(d[i+k])
		}
	}
	return uint32(v >> (8 - sh))
}

// refBitsN est `bitsN` d avant : n <= 8 bits, bit a bit.
func refBitsN(d []byte, p, n int) int {
	v := 0
	for k := 0; k < n; k++ {
		v = v<<1 | refBitAt(d, p+k)
	}
	return v
}

// largeursEprouvees : toutes les largeurs de 0 a 64, plus celles que `evBody15` atteint quand sa
// longueur R(10) depasse le mot (`r.rd(n)`, n jusqu a 1023).
func largeursEprouvees() []int {
	out := make([]int, 0, 72)
	for n := 0; n <= 64; n++ {
		out = append(out, n)
	}
	return append(out, 65, 71, 100, 127, 128, 255, 1023)
}

// TestEquivalenceLecteurEvBitAbit : appel par appel, sur les positions REELLES des chaines.
func TestEquivalenceLecteurEvBitAbit(t *testing.T) {
	largeurs := largeursEprouvees()
	paquets, positions := 0, 0
	for _, dir := range bobinesVersionnees(t) {
		for _, pl := range payloadsAEvents(t, dir) {
			paquets++
			for _, p := range positionsReelles(pl) {
				positions++
				comparerPrimitives(t, dir, pl, p)
				for _, n := range largeurs {
					comparerLecture(t, dir, pl, p, n)
				}
			}
		}
	}
	if paquets == 0 || positions == 0 {
		t.Fatalf("balayage muet : %d paquet(s), %d position(s) — le test ne garderait rien",
			paquets, positions)
	}
	t.Logf("%d paquets a events, %d positions reelles, %d largeurs par position",
		paquets, positions, len(largeurs))
}

// comparerLecture oppose `rd` et `skip` des deux lecteurs a une position et une largeur.
func comparerLecture(t *testing.T, dir string, pl []byte, p, n int) {
	t.Helper()
	ref := &refEvReader{pl: pl, bp: p}
	vRef := ref.rd(n)
	cur := nouveauCurseurEv(pl, p)
	vCur := cur.rd(n)
	if vRef != vCur || ref.bp != cur.pos() || ref.over != cur.over {
		t.Fatalf("%s : rd(%d) a la position %d — reference (v=%#x, bp=%d, over=%v) contre "+
			"canonique (v=%#x, bp=%d, over=%v)", dir, n, p, vRef, ref.bp, ref.over,
			vCur, cur.pos(), cur.over)
	}
	refS := &refEvReader{pl: pl, bp: p}
	refS.skip(n)
	curS := nouveauCurseurEv(pl, p)
	curS.skip(n)
	if refS.bp != curS.pos() || refS.over != curS.over {
		t.Fatalf("%s : skip(%d) a la position %d — reference (bp=%d, over=%v) contre "+
			"canonique (bp=%d, over=%v)", dir, n, p, refS.bp, refS.over, curS.pos(), curS.over)
	}
}

// comparerPrimitives oppose les lectures par position du paquet aux primitives de la couche
// source.
func comparerPrimitives(t *testing.T, dir string, pl []byte, p int) {
	t.Helper()
	if a, b := refBitAt(pl, p), source.BitAt(pl, p); a != b {
		t.Fatalf("%s : bitAt(%d) — reference %d contre canonique %d", dir, p, a, b)
	}
	if a, b := refBits32(pl, p), uint32(source.BitsAt(pl, p, 32)); a != b {
		t.Fatalf("%s : bits32(%d) — reference %#x contre canonique %#x", dir, p, a, b)
	}
	for n := 0; n <= 8; n++ {
		if a, b := refBitsN(pl, p, n), int(source.BitsAt(pl, p, uint(n))); a != b {
			t.Fatalf("%s : bitsN(%d, %d) — reference %d contre canonique %d", dir, p, n, a, b)
		}
	}
	for n := 0; n <= 64; n++ {
		if a, b := refBitsWide(pl, p, n), source.BitsAt(pl, p, uint(n)); a != b {
			t.Fatalf("%s : bitsWide(%d, %d) — reference %#x contre canonique %#x",
				dir, p, n, a, b)
		}
	}
}

// payloadsAEvents : les payloads des paquets type-0 A EVENTS d une bobine.
func payloadsAEvents(t *testing.T, dir string) [][]byte {
	t.Helper()
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("%s : %v", dir, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		return nil // bobine sans paquet type-0 (fixture partielle) : rien a comparer
	}
	var out [][]byte
	for i := range f.t0 {
		p := &f.t0[i]
		if hasEvents(p) {
			out = append(out, p.payload)
		}
	}
	return out
}

// bitsDeQueueEprouves : combien de bits de FIN de paquet sont eprouves en plus des positions de
// la chaine. Quatre-vingts couvre la plus longue suite obligatoire d un kill-event (32 + 32 + 5
// + 5 + 3 bits) a cheval sur la fin du tampon.
const bitsDeQueueEprouves = 80

// positionsReelles : les positions que la marche d evenements visite vraiment dans ce paquet,
// plus la queue du tampon (cf. l en-tete).
func positionsReelles(pl []byte) []int {
	vues := map[int]bool{2: true}
	for _, x := range positionsCandidates(pl) {
		vues[x], vues[x+7] = true, true
		r := nouveauCurseurEv(pl, x+7)
		if !evPresence(r, killEventCode) {
			continue
		}
		vues[r.pos()] = true
		k := readKillEvent(pl, r.pos())
		if !killEventPlausible(k) {
			continue
		}
		vues[k.end] = true
		marquerBornesDeChaine(pl, k.end, vues)
	}
	marquerBornesDeChaine(pl, 2, vues)
	nb := len(pl) * 8
	for p := max(0, nb-bitsDeQueueEprouves); p <= nb; p++ {
		vues[p] = true
	}
	out := make([]int, 0, len(vues))
	for p := range vues {
		out = append(out, p)
	}
	return out
}

// marquerBornesDeChaine enregistre les bornes de chaque evenement d une chaine, pour les DEUX
// valeurs de `gate15` : celle du film ne se lit pas ici, et les deux sont des marches reelles.
func marquerBornesDeChaine(pl []byte, depart int, vues map[int]bool) {
	for _, g15 := range []bool{false, true} {
		r := nouveauCurseurEv(pl, depart)
		for n := 0; n < maxChainProbe; n++ {
			vues[r.pos()] = true
			fin, ok := evStep(r, g15)
			vues[r.pos()] = true
			if fin || !ok {
				break
			}
		}
	}
}
