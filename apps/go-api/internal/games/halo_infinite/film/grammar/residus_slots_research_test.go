package grammar

// residus_slots_research_test.go — PHASE 5b, RESIDU 2 DE LA TABLE DES SLOTS.
//
// LE RESIDU, LAISSE OUVERT PAR LA PHASE 4 (NOTE_PROFIL_PAR_BUILD, section B.5) :
//
//	R2 LE LECTEUR CANONIQUE EST MORT SUR LES BUILDS ANCIENS (le residu 1 vit dans
//	   `residus_vacants_research_test.go`). `s3sChaine` avance du pas PREDIT
//	   par la grammaire lue dans l'exe courant ; sur un film d'avant `HI_1_12_0` ce pas est faux
//	   de -2 880, -4 320 ou +1 600 bits DES LE PREMIER enregistrement, et le lecteur s'arrete a
//	   1 enregistrement. La constante est MESUREE depuis la phase 2 mais jamais EXPLIQUEE : quel
//	   champ change de largeur ?
//
// LA MESURE QUI TRANCHE R2, ET POURQUOI ELLE EST POSSIBLE SANS EXE DE CES BUILDS. L'ordre
// d'ecriture de `FUN_1407edea8` place TOUS les champs de longueur variable AVANT le gamertag
// (masque, liste d'octets, liste de mots) et tous les champs de largeur fixe APRES lui :
//
//	[... listes variables ...] [gamertag sub+0xc14] [bloc16] [repr] [q64] [six champs courts]
//	[PERSONNALISATION sub+0xcc0, 14 816 bits] [BLOC DE QUEUE sub+0x1400, 352 bits] [u32]
//
// Le bloc de queue porte le MEME gamertag, en UTF-16 PETIT-BOUTISTE (phase 2, A.4 : 640/640).
// Il est donc LOCALISABLE dans le flux sans rien supposer de ce qui le precede : on cherche les
// deux premieres unites du nom deja lu. Cela coupe l'enregistrement en deux moities mesurables
// separement, et la transposition du build tombe dans l'une des deux :
//
//	AVANT = (position du nom petit-boutiste) - (fin du champ de gamertag)
//	APRES = (debut de l'enregistrement suivant) - (position du nom petit-boutiste)
//
// Le controle qui va avec est gratuit et il est ecrit d'avance : entre la fin du gamertag et le
// bloc de queue, le seul champ assez gros pour porter une constante de 2 880 a 4 320 bits est le
// bloc de personnalisation, mesure ENTIEREMENT A ZERO par la phase 2 (0 octet non nul sur
// 81 488). On mesure donc AUSSI la plus longue suite de bits nuls de cet intervalle : si elle
// recule exactement de la transposition, le champ est nomme.
//
// Gardes CHUNK00_FILMS (+ CHUNK00_CORPUS pour le controle d'echelle). Lecture seule, aucun code
// de production touche.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"unicode/utf16"
)

const (
	// rsApresRef : la moitie APRES du nom petit-boutiste, telle que la grammaire de l'exe
	// courant la predit quand le nom est a l'offset 0 du bloc de queue : 352 bits de bloc
	// moins 0, plus le u32 de `slot+0x1448`.
	rsApresRef = s3sBloc44 + 32
	// rsFenetreQueue : borne de la recherche du nom petit-boutiste apres la fin du champ de
	// gamertag. Large : la moitie fixe vaut 15 086 bits sur l'exe courant, et un build
	// anterieur peut la depasser (`HI_1_4_1` : +1 600).
	rsFenetreQueue = 24000
)

// rsNomLE rend la position de bit de la premiere unite UTF-16 PETIT-BOUTISTE du gamertag de `e`,
// cherchee a partir de `from`. Rend -1 si elle n'est pas trouvee dans la fenetre.
//
// La recherche porte sur les DEUX premieres unites (32 bits) : une seule unite (`4d 00`) se
// rencontre par hasard dans un flux bit-packe, deux non — et le controle de non-ambiguite est
// publie par l'appelant (nombre de touches dans la fenetre).
func rsNomLE(d []byte, e *s3sEnr, from, to int) (pos, touches int) {
	u := utf16.Encode([]rune(e.gamertag))
	if len(u) < 2 {
		return -1, 0
	}
	motif := uint64(u[0]&0xff)<<24 | uint64(u[0]>>8)<<16 | uint64(u[1]&0xff)<<8 | uint64(u[1]>>8)
	pos = -1
	for p := from; p+32 <= to; p++ {
		if s3rBit(d, p, 32) != motif {
			continue
		}
		touches++
		if pos < 0 {
			pos = p
		}
	}
	return pos, touches
}

// rsFinGamertag rend la position de bit qui suit le champ de gamertag (`sub+0xc14`) d'un
// enregistrement deja decode. Calculee, pas cherchee : tous les termes sont lus dans le flux.
func rsFinGamertag(e *s3sEnr) int {
	unites := len(utf16.Encode([]rune(e.gamertag))) + 1
	if unites > s3sGtMax {
		unites = s3sGtMax
	}
	return e.debut + s3rEnteteBits + 64 + s3sPrefixeMasque + e.compteMasque +
		s3sLargeurN + e.n*8 + s3sLargeurM + e.m*32 + s3sBloc104 + unites*16
}

// rsPlusLongueSuiteNulle rend la longueur de la plus longue suite de bits nuls de `[a, b)` et
// la position ou elle commence.
func rsPlusLongueSuiteNulle(d []byte, a, b int) (long, deb int) {
	cur, curDeb := 0, a
	for p := a; p < b; p++ {
		if s3rBit(d, p, 1) == 0 {
			if cur == 0 {
				curDeb = p
			}
			cur++
			if cur > long {
				long, deb = cur, curDeb
			}
			continue
		}
		cur = 0
	}
	return long, deb
}

// rsCoupe : la decomposition d'un enregistrement en ses deux moities, autour du nom
// petit-boutiste du bloc de queue.
//
// `reste` est la mesure DERIVEE qui porte la conclusion : la largeur de la moitie AVANT une fois
// retiree sa plus longue suite de bits nuls ET l'offset du nom dans le bloc de queue (lequel
// depend du build, phase 2 A.4 : 0 sur `HI_1_13_0`, 12 octets sur `HI_1_12_0`). Si ce reste est
// le MEME sur tous les builds, alors aucun champ autre que le bloc nul n'a change de largeur.
type rsCoupe struct {
	avant, apres, zeros, reste, touches int
	ok                                  bool
}

// rsDecoupe mesure les deux moities d'un enregistrement dont on connait l'ecart au suivant.
func rsDecoupe(d []byte, e *s3sEnr, ecart int) rsCoupe {
	fin := rsFinGamertag(e)
	pos, touches := rsNomLE(d, e, fin, fin+rsFenetreQueue)
	if pos < 0 {
		return rsCoupe{}
	}
	z, _ := rsPlusLongueSuiteNulle(d, fin, pos)
	c := rsCoupe{avant: pos - fin, apres: ecart - (pos - e.debut), zeros: z,
		touches: touches, ok: true}
	c.reste = c.avant - c.zeros - (rsApresRef - c.apres)
	return c
}

// les slots vacants (R1).
func rsEcarts(d []byte) ([]*s3sEnr, []int) {
	fin := (dernierNonNul(d) + 1) * 8
	var es []*s3sEnr
	for _, h := range profilRosterTable(d) {
		if e := s3sDecode(d, h.bit-s3rEnteteBits, fin); e != nil && s3sImprimable(e.gamertag) {
			es = append(es, e)
		}
	}
	ecarts := make([]int, len(es))
	for i := range es {
		if i+1 < len(es) {
			ecarts[i] = es[i+1].debut - es[i].debut
		}
	}
	return es, ecarts
}

// TestResidusSlotTransposition execute R2-COUPE : ou tombe la transposition du build ?
//
// Critere ecrit avant la mesure : si la moitie APRES est la MEME sur tous les builds, alors la
// transposition est entierement AVANT le bloc de queue ; et si la plus longue suite de bits
// nuls de cette moitie recule de la meme quantite, le champ qui change est celui qui est ecrit
// a zero — le bloc de personnalisation `sub+0xcc0`.
func TestResidusSlotTransposition(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		es, ecarts := rsEcarts(d)
		avants, apress, zeross, transpo := map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{}
		restes := map[int]int{}
		amb := 0
		for i, e := range es {
			if ecarts[i] == 0 {
				continue
			}
			transpo[ecarts[i]-s3sPredite(e)]++
			c := rsDecoupe(d, e, ecarts[i])
			if !c.ok {
				continue
			}
			if c.touches > 1 {
				amb++
			}
			avants[c.avant]++
			apress[c.apres]++
			zeross[c.zeros]++
			restes[c.reste]++
		}
		t.Logf("%-10s %-10q transposition %s | AVANT %s | APRES %s (reference %d) | "+
			"plus longue suite nulle %s (= %s octets) | RESTE hors bloc nul %s | "+
			"%d position(s) ambigue(s)",
			filepath.Base(dir), build, s3pDist(transpo), s3pDist(avants), s3pDist(apress),
			rsApresRef, s3pDist(zeross), s3pDist(rsEnOctets(zeross)), s3pDist(restes), amb)
	}
}

// rsDelta rend la transposition MODALE d'un film : la valeur de `mesure - predit` qui couvre le
// plus d'ecarts. C'est la calibration du lecteur corrige, et elle se fait SUR LE FILM, sans
// table de build ecrite a la main — donc elle vaut aussi pour un build inconnu.
func rsDelta(d []byte) (delta, couverts, total int) {
	es, ecarts := rsEcarts(d)
	comptes := map[int]int{}
	for i, e := range es {
		if ecarts[i] == 0 {
			continue
		}
		comptes[ecarts[i]-s3sPredite(e)]++
		total++
	}
	best := -1
	for k, n := range comptes {
		if n > best || (n == best && k < delta) {
			delta, best = k, n
		}
	}
	if best < 0 {
		return 0, 0, 0
	}
	return delta, best, total
}

// rsChaine est le LECTEUR CANONIQUE CORRIGE : `s3sChaine`, mais le pas predit est corrige de la
// constante du build, CALIBREE SUR LE FILM par `rsDelta`. Le depart est le premier
// enregistrement a nom imprimable du balayage CORRIGE (critere `slot+0x08` leve, phase 4).
func rsChaine(d []byte, delta int) (enrs []*s3sEnr, vacants int) {
	fin := (dernierNonNul(d) + 1) * 8
	hs := profilRosterBalaye(d, profilRosterCritCorrigee())
	deb := -1
	for _, h := range hs {
		e := s3sDecode(d, h.bit-s3rEnteteBits, fin)
		if e != nil && s3sImprimable(e.gamertag) {
			deb = e.debut
			break
		}
	}
	if deb < 0 {
		return nil, 0
	}
	for p, lus := deb, 0; lus < 32; lus++ {
		if rsVacant(d, p) {
			// Slot VACANT : le balayage ne peut pas le voir, mais la grammaire en connait la
			// longueur EXACTE. On l'enjambe sans le compter comme un joueur — c'est la cause
			// des deux ecarts aberrants de R1.
			vacants++
			p += rsVide(delta)
			continue
		}
		e := s3sDecode(d, p, fin)
		if e == nil || !s3sImprimable(e.gamertag) {
			break
		}
		enrs = append(enrs, e)
		p += s3sPredite(e) + delta
	}
	return enrs, vacants
}

// TestResidusSlotChaineParBuild execute R2-LEC : le lecteur par GRAMMAIRE, calibre sur le film,
// contre le lecteur d'origine et contre le balayage brut corrige.
//
// Critere ecrit avant la mesure : sur chaque film, le lecteur corrige doit rendre le MEME nombre
// d'enregistrements que le balayage corrige — c'est-a-dire qu'il doit cesser d'etre mort sur les
// builds anciens, sans rien perdre sur le build courant.
func TestResidusSlotChaineParBuild(t *testing.T) {
	egaux, vus := 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build, _ := s3bBuild(d)
		delta, couverts, total := rsDelta(d)
		orig := len(s3sChaine(d))
		es, vac := rsChaine(d, delta)
		corr := len(es)
		brut := len(profilRosterTable(d))
		vus++
		verdict := "DIFFERENT"
		if corr == brut {
			egaux++
			verdict = "EGAL"
		}
		t.Logf("%-10s %-10q delta %+6d (%d/%d ecarts) | s3sChaine %2d | rsChaine %2d | "+
			"balayage corrige %2d | %d slot(s) VACANT(s) enjambe(s) | %s",
			filepath.Base(dir), build, delta, couverts, total, orig, corr, brut, vac, verdict)
	}
	t.Logf("=== BILAN R2-LEC === %d/%d film(s) ou le lecteur par grammaire calibre rend "+
		"exactement le compte du balayage corrige", egaux, vus)
}

// TestResidusSlotChaineCorpus est le controle d'echelle de R2-LEC : tout un repertoire de films,
// tous builds confondus. Garde CHUNK00_CORPUS (+ CHUNK00_CORPUS_MAX, defaut 1400).
func TestResidusSlotChaineCorpus(t *testing.T) {
	racine := os.Getenv("CHUNK00_CORPUS")
	if racine == "" {
		t.Skip("CHUNK00_CORPUS absent : controle d'echelle saute")
	}
	max := 1400
	if v, err := strconv.Atoi(os.Getenv("CHUNK00_CORPUS_MAX")); err == nil && v > 0 {
		max = v
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	par, lus := map[string]*rsBilan{}, 0
	for _, e := range entrees {
		if !e.IsDir() || lus >= max {
			continue
		}
		d, err := os.ReadFile(filepath.Join(racine, e.Name(), "chunk_00.bin"))
		if err != nil || len(d) < s3bEnteteFin {
			continue
		}
		lus++
		rsAjoute(par, d)
	}
	rsRapport(t, par, lus)
}

// rsBilan : le bilan par build du controle d'echelle.
type rsBilan struct {
	films, egaux, origMorts, vacants, complets int
	deltas                                     map[int]int
	orig, corr, brut                           int
}

// rsAjoute mesure un film et l'ajoute au bilan de son build.
func rsAjoute(par map[string]*rsBilan, d []byte) {
	build, off := s3bBuild(d)
	if off < 0 {
		build = "(sans section d'identification)"
	}
	b := par[build]
	if b == nil {
		b = &rsBilan{deltas: map[int]int{}}
		par[build] = b
	}
	delta, _, _ := rsDelta(d)
	es, vac := rsChaine(d, delta)
	orig, corr, brut := len(s3sChaine(d)), len(es), len(profilRosterTable(d))
	b.films++
	b.deltas[delta]++
	b.orig += orig
	b.corr += corr
	b.brut += brut
	b.vacants += vac
	if corr+vac == 32 {
		b.complets++
	}
	if corr == brut {
		b.egaux++
	}
	if orig <= 1 && brut > 1 {
		b.origMorts++
	}
}

// rsRapport imprime le bilan du controle d'echelle, un build par ligne.
func rsRapport(t *testing.T, par map[string]*rsBilan, lus int) {
	t.Helper()
	var cles []string
	for k := range par {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return par[cles[i]].films > par[cles[j]].films })
	t.Logf("%d chunk_00 lus", lus)
	for _, k := range cles {
		b := par[k]
		t.Logf("=== %-32s %4d film(s) ; delta(s) %s ; enregistrements lus : s3sChaine %5d, "+
			"rsChaine %5d, balayage corrige %5d ; %d/%d film(s) au compte du balayage ; "+
			"%d film(s) ou s3sChaine est MORT (<= 1) ; %d slot(s) VACANT(s) enjambe(s) ; "+
			"%d/%d film(s) ou lus+vacants == 32, la borne de l'ecrivain",
			k, b.films, s3pDist(b.deltas), b.orig, b.corr, b.brut, b.egaux, b.films,
			b.origMorts, b.vacants, b.complets, b.films)
	}
}

// rsEnOctets convertit une distribution de largeurs en bits en une distribution en octets.
func rsEnOctets(m map[int]int) map[int]int {
	out := map[int]int{}
	for k, n := range m {
		out[k/8] += n
	}
	return out
}
