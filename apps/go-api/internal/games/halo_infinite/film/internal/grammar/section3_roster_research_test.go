package grammar

// section3_roster_research_test.go — LA TABLE DES SLOTS DU MATCH, DANS chunk_00.
//
// ## CE QUE L'ECRIVAIN DIT, ET QUI EST LA SEULE RAISON DE CHERCHER ICI
//
// Le corps de `chunk_00` (« section 3 » de la carte du 2026-08-30) est serialise par
// `FUN_1407ec560(writer, jeu)`. Sa DERNIERE instruction est une boucle, lue au desassemblage :
//
//	LEA RDI,[RSI + 0xeaaf0]      ; premier enregistrement
//	LEA RSI,[RDI + 0x28a00]      ; borne de fin
//	... FUN_1407ecb08(enregistrement, writer) ; enregistrement += 0x1450
//
// 0x28A00 / 0x1450 = **32 enregistrements de 0x1450 octets**. Et `FUN_1407ecb08` donne l'en-tete
// de chacun, champ par champ, avec sa largeur en bits :
//
//	slot+0x00  1 bit    FUN_1406d49c4 (booleen)
//	slot+0x01  1 bit    booleen
//	slot+0x02  1 bit    booleen
//	slot+0x04  32 bits  u32
//	slot+0x08  2 bits   octet signe
//	slot+0x09  48 bits  FUN_1406d60f4(…, 0x30)
//	slot+0x10  64 bits  FUN_1406d6498(…, 0x40)   <= UN ENTIER DE 64 BITS
//	slot+0x18  …        FUN_1407edea8 (sous-enregistrement, jusqu'a slot+0x1448)
//	slot+0x1448 32 bits u32
//
// Soit **85 bits d'en-tete avant l'entier de 64 bits**. Un entier de 64 bits par joueur, dans une
// table de 32 slots, au debut d'un enregistrement : l'hypothese est **le XUID**.
//
// ## LES CONTROLES, TOUS ECRITS AVANT LA MESURE
//
//	R-POS  Les XUID du roster de `match_participants` sont cherches BIT A BIT (MSB-first) dans
//	       `chunk_00`. Critere : chaque XUID humain est trouve AU PLUS UNE FOIS, et la position
//	       trouvee est a 85 bits d'un motif d'en-tete conforme. Roster fourni par
//	       `CHUNK00_ROSTERS` (`film=xuid,xuid;…`).
//	R-NEG  Pour chaque film, 40 leurres de 64 bits tires au hasard DANS LA PLAGE DES XUID XBOX
//	       (`0x0009000000000000`..`0x000A000000000000`) sont cherches de la meme facon. C'est le
//	       plancher de faux positifs, MESURE et non calcule (methode, regle 4).
//	R-INT  ORACLE INTERNE, sans aucune entree externe : le balayage cherche le MOTIF d'en-tete
//	       (trois booleens 1/0/0, u32 nul, champ de 2 bits nul) suivi d'un entier de 64 bits dans
//	       la plage XUID. S'il retrouve le roster sans qu'on le lui donne, la lecture est bonne.
//	R-CRO  CONTROLE CROISE INTER-FILMS, egalement interne : si la longueur d'un enregistrement
//	       depend du JOUEUR (son nom, sa personnalisation) et non du match, alors un joueur
//	       present dans deux films doit y avoir la MEME longueur d'enregistrement. Un decoupage
//	       faux ne produirait pas cette egalite.
//
// Garde `CHUNK00_FILMS`. Lecture seule, aucun code de production touche.

import (
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	// s3rXuidLo / s3rXuidHi : la plage des XUID Xbox Live. Bornes ecrites avant la mesure ;
	// elles servent AUSSI a fabriquer les leurres du controle R-NEG, pour que ceux-ci aient
	// exactement la meme forme que les vraies valeurs cherchees.
	s3rXuidLo = uint64(0x0009000000000000)
	s3rXuidHi = uint64(0x000A000000000000)
	// s3rEnteteBits : les 85 bits d'en-tete qui precedent l'entier de 64 bits (1+1+1+32+2+48).
	s3rEnteteBits = 85
	// s3rCorpsBit : borne basse du balayage, le debut du corps dans le flux.
	s3rCorpsBit = s3wCorpsOff * 8
	// s3rEcartMax : au-dela de cet ecart en bits, deux touches n'appartiennent pas a la meme
	// grappe d'enregistrements. Les longueurs mesurees vont de 16 611 a 28 145 bits ; le seuil
	// est pose a 40 000, soit 1,4 fois la plus longue.
	s3rEcartMax = 40000
	s3rLeurres  = 40
)

// s3rTouche : une occurrence d'entier de 64 bits precedee d'un en-tete conforme.
type s3rTouche struct {
	bit   int
	xuid  uint64
	token uint64 // le champ de 48 bits a slot+0x09
}

// s3rBit lit n bits MSB-first.
func s3rBit(d []byte, bit, n int) uint64 {
	var acc uint64
	for k := 0; k < n; k++ {
		b := bit + k
		if b>>3 >= len(d) {
			return 0
		}
		acc = acc<<1 | uint64((d[b>>3]>>(7-uint(b&7)))&1)
	}
	return acc
}

// s3rCherche rend toutes les positions de bit ou la valeur 64 bits `v` apparait MSB-first.
func s3rCherche(d []byte, v uint64, from, to int) []int {
	var out []int
	hi := v >> 48
	for bit := from; bit+64 <= to; bit++ {
		if s3rBit(d, bit, 16) != hi {
			continue
		}
		if s3rBit(d, bit, 64) == v {
			out = append(out, bit)
		}
	}
	return out
}

// s3rEnteteConforme dit si les 85 bits qui precedent `bit` forment l'en-tete que
// `FUN_1407ecb08` ecrit : booleens 1/0/0, u32 nul, champ de 2 bits nul.
func s3rEnteteConforme(d []byte, bit int) bool {
	s := bit - s3rEnteteBits
	if s < 0 {
		return false
	}
	return s3rBit(d, s, 3) == 4 && s3rBit(d, s+3, 32) == 0 && s3rBit(d, s+35, 2) == 0
}

// TestSection3RosterParXuid execute R-POS et R-NEG : les XUID du registre sont-ils dans
// `chunk_00`, et a quel plancher de faux positifs ?
func TestSection3RosterParXuid(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	rosters := s3rRosters(os.Getenv("CHUNK00_ROSTERS"))
	if len(rosters) == 0 {
		t.Skip("CHUNK00_ROSTERS absent : R-POS/R-NEG sautes (voir TestSection3RosterInterne)")
	}
	totalVus, totalCherches, totalFaux := 0, 0, 0
	for _, dir := range dirs {
		id := filepath.Base(dir)
		xs, ok := rosters[id]
		if !ok {
			t.Logf("=== %s === absent de CHUNK00_ROSTERS", id)
			continue
		}
		_, d := readChunk00(t, dir)
		fin := (dernierNonNul(d) + 1) * 8
		vus := s3rTrouve(t, d, id, xs, fin)
		totalVus += vus
		totalCherches += len(xs)
		faux := s3rPlancher(d, fin)
		totalFaux += faux
		t.Logf("  R-NEG : %d leurres de la plage XUID -> %d touche(s)", s3rLeurres, faux)
	}
	t.Logf("=== BILAN === R-POS %d/%d XUID trouves exactement une fois ; "+
		"R-NEG %d touches sur %d leurres", totalVus, totalCherches, totalFaux, s3rLeurres*len(dirs))
}

// s3rTrouve cherche chaque XUID et rend le nombre trouve exactement une fois.
func s3rTrouve(t *testing.T, d []byte, id string, xs []uint64, fin int) int {
	t.Helper()
	var hits []s3rTouche
	vus := 0
	for _, x := range xs {
		pos := s3rCherche(d, x, s3rCorpsBit, fin)
		if len(pos) == 1 {
			vus++
		}
		for _, p := range pos {
			hits = append(hits, s3rTouche{bit: p, xuid: x, token: s3rBit(d, p-48, 48)})
		}
		if len(pos) != 1 {
			t.Logf("  %s : xuid %d -> %d touche(s) (attendu 1)", id, x, len(pos))
		}
	}
	t.Logf("=== %s === %d/%d XUID trouves exactement une fois", id, vus, len(xs))
	s3rImprime(t, d, hits)
	return vus
}

// s3rImprime trie les touches et imprime l'ordre des enregistrements, leur longueur et la
// conformite de leur en-tete.
func s3rImprime(t *testing.T, d []byte, hits []s3rTouche) {
	t.Helper()
	sort.Slice(hits, func(i, j int) bool { return hits[i].bit < hits[j].bit })
	conformes := 0
	for i, h := range hits {
		lg := 0
		if i+1 < len(hits) {
			lg = hits[i+1].bit - h.bit
		}
		ok := s3rEnteteConforme(d, h.bit)
		if ok {
			conformes++
		}
		t.Logf("  slot %2d  bit %9d (octet 0x%06x+%d)  xuid %19d  token48 %012x  "+
			"longueur %6d bits  en-tete conforme %t",
			i, h.bit, h.bit/8, h.bit%8, h.xuid, h.token, lg, ok)
	}
	t.Logf("  R-POS en-tete : %d/%d touches precedees du motif 1/0/0 + u32 nul + 2 bits nuls",
		conformes, len(hits))
}

// s3rPlancher execute R-NEG : des leurres de la meme plage, tires au hasard.
func s3rPlancher(d []byte, fin int) int {
	rng := rand.New(rand.NewSource(7))
	n := 0
	for i := 0; i < s3rLeurres; i++ {
		v := s3rXuidLo | (rng.Uint64() & 0x0000ffffffffffff)
		n += len(s3rCherche(d, v, s3rCorpsBit, fin))
	}
	return n
}

// TestSection3RosterInterne execute R-INT : retrouver la table des slots SANS aucune entree
// externe, par le seul motif d'en-tete de `FUN_1407ecb08`.
func TestSection3RosterInterne(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		fin := (dernierNonNul(d) + 1) * 8
		brutes := s3rBalayage(d, fin)
		grappe := s3rGrappe(brutes)
		t.Logf("=== %s === build %q ; fin de flux <= bit %d", filepath.Base(dir),
			s3wChaine(d, s3wBuildOff, 32), fin)
		t.Logf("  balayage brut : %d position(s) conformes ; apres regroupement de fin de flux "+
			"et rejet des valeurs degenerees : %d", len(brutes), len(grappe))
		for i, h := range grappe {
			lg := 0
			if i+1 < len(grappe) {
				lg = grappe[i+1].bit - h.bit
			}
			t.Logf("  slot %2d  bit %9d  xuid %19d  token48 %012x  longueur %6d bits",
				i, h.bit, h.xuid, h.token, lg)
		}
	}
}

// s3rBalayage cherche le motif d'en-tete suivi d'un entier de 64 bits dans la plage XUID.
func s3rBalayage(d []byte, fin int) []s3rTouche {
	var out []s3rTouche
	for p := s3rCorpsBit; p+s3rEnteteBits+64 <= fin; p++ {
		if s3rBit(d, p, 3) != 4 || s3rBit(d, p+3, 32) != 0 || s3rBit(d, p+35, 2) != 0 {
			continue
		}
		x := s3rBit(d, p+s3rEnteteBits, 64)
		if x < s3rXuidLo || x >= s3rXuidHi {
			continue
		}
		out = append(out, s3rTouche{bit: p + s3rEnteteBits, xuid: x, token: s3rBit(d, p+37, 48)})
	}
	return out
}

// s3rGrappe garde la grappe terminale : la plus longue suite de touches dont les ecarts
// consecutifs restent sous s3rEcartMax, en fin de flux. Les deux formes degenerees mesurees
// (XUID egal a la borne basse de la plage, jeton de 48 bits nul) sont rejetees d'abord.
func s3rGrappe(in []s3rTouche) []s3rTouche {
	var f []s3rTouche
	for _, h := range in {
		if h.xuid == s3rXuidLo || h.token == 0 {
			continue
		}
		f = append(f, h)
	}
	if len(f) == 0 {
		return nil
	}
	deb := len(f) - 1
	for deb > 0 && f[deb].bit-f[deb-1].bit <= s3rEcartMax {
		deb--
	}
	return f[deb:]
}

// TestSection3LongueurParJoueur execute R-CRO : la longueur d'un enregistrement suit le JOUEUR,
// pas le match. Un joueur present dans deux films doit y avoir la meme longueur.
func TestSection3LongueurParJoueur(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	if len(dirs) < 2 {
		t.Skip("R-CRO exige au moins deux films")
	}
	par := map[uint64]map[int][]string{}
	for _, dir := range dirs {
		_, d := readChunk00(t, dir)
		g := s3rGrappe(s3rBalayage(d, (dernierNonNul(d)+1)*8))
		for i := 0; i+1 < len(g); i++ {
			lg := g[i+1].bit - g[i].bit
			if par[g[i].xuid] == nil {
				par[g[i].xuid] = map[int][]string{}
			}
			par[g[i].xuid][lg] = append(par[g[i].xuid][lg], filepath.Base(dir))
		}
	}
	var xs []uint64
	for x, m := range par {
		total := 0
		for _, f := range m {
			total += len(f)
		}
		if total >= 2 {
			xs = append(xs, x)
		}
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	stables, vus := 0, 0
	for _, x := range xs {
		m := par[x]
		vus++
		if len(m) == 1 {
			stables++
		}
		for lg, films := range m {
			t.Logf("  xuid %19d : longueur %6d bits sur %v", x, lg, films)
		}
	}
	t.Logf("  R-CRO : %d/%d joueurs vus dans au moins deux films ont une longueur "+
		"d'enregistrement CONSTANTE", stables, vus)
}

// TestSection3RosterCorpus execute le controle d'echelle : le balayage interne passe sur tout un
// repertoire de films (garde `CHUNK00_CORPUS` = la racine de `film_chunks/`, `CHUNK00_CORPUS_MAX`
// = nombre de films a lire, defaut 250) et rend la DISTRIBUTION du nombre d'enregistrements.
//
// CE QU'IL DOIT MONTRER, ecrit avant la mesure : des comptes qui ressemblent a des tailles
// d'escouade Halo, et JAMAIS plus de 32 — la borne que l'ecrivain impose (0x28A00 / 0x1450). Un
// decoupage faux produirait des comptes arbitraires, et en particulier des comptes > 32.
func TestSection3RosterCorpus(t *testing.T) {
	root := os.Getenv("CHUNK00_CORPUS")
	if root == "" {
		t.Skip("CHUNK00_CORPUS absent : controle d'echelle saute")
	}
	max := 250
	if v, err := strconv.Atoi(os.Getenv("CHUNK00_CORPUS_MAX")); err == nil && v > 0 {
		max = v
	}
	entrees, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("lecture de %s : %v", root, err)
	}
	dist, builds, lus, hors := map[int]int{}, map[string]int{}, 0, 0
	for _, e := range entrees {
		if !e.IsDir() || lus >= max {
			continue
		}
		d, err := os.ReadFile(filepath.Join(root, e.Name(), "chunk_00.bin"))
		if err != nil || len(d) < s3wBoolOff {
			continue
		}
		b := s3wChaine(d, s3wBuildOff, 32)
		if !strings.HasPrefix(b, "HI_") {
			continue
		}
		builds[b]++
		n := len(s3rGrappe(s3rBalayage(d, (dernierNonNul(d)+1)*8)))
		dist[n]++
		if n > 32 {
			hors++
			t.Errorf("%s rend %d enregistrements, or l'ecrivain en impose 32 au plus", e.Name(), n)
		}
		lus++
	}
	var ks []int
	for k := range dist {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	t.Logf("%d films lus ; builds %v", lus, builds)
	for _, k := range ks {
		t.Logf("  %2d enregistrement(s) -> %4d film(s)", k, dist[k])
	}
	t.Logf("  controle d'echelle : %d film(s) au-dela de la borne de 32", hors)
}

// s3rRosters decoupe `film=xuid,xuid;film=xuid`.
func s3rRosters(v string) map[string][]uint64 {
	out := map[string][]uint64{}
	for _, p := range strings.Split(v, ";") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		var xs []uint64
		for _, s := range strings.Split(kv[1], ",") {
			if n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64); err == nil {
				xs = append(xs, n)
			}
		}
		if len(xs) > 0 {
			out[strings.TrimSpace(kv[0])] = xs
		}
	}
	return out
}
