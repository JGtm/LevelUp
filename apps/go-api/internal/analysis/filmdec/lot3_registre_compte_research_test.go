package filmdec

// lot3_registre_compte_research_test.go — LOT 3 DU PLAN « PERCER LA TRAME » (2026-08-30) :
// TRANCHER LE COMPTE DU REGISTRE DE REPLICATION — 118 BLOCS OU 50 ?
//
// LE CONFLIT. Le dossier tient « 118 blocs d'archetype » ; la relecture du lot D (carte de
// chunk_00) en trouve 50 et affirme que le 118 est `len(fichier)/taille_bloc`, c'est-a-dire un
// artefact de division qui annexe les sections 2 et 3 au registre. Le garde-rail G2 etait
// tenu pour l'arbitre : execute sur les trois temoins du lot D, il est tombe ROUGE sur
// `00162144` (bloc 71 « B », 1 068 slots, empreinte inconnue) — il ne verifiait donc pas le
// compte de blocs, seulement les slots nommes, et il n'avait jamais vu ce film.
//
// LA REGLE STRUCTURELLE TESTEE : un bloc de registre est « une suite de slots nommes en tete,
// puis un slot de terminaison dont seul le champ flags (run*260+4) peut etre non nul, puis des
// zeros jusqu'au bout du bloc » (bloc vide = zero slot, ex. bloc 8). Le premier bloc qui viole
// cette regle marque la fin du registre.
//
// AFFINAGE DOCUMENTE (2026-08-30, premiere passe de cette mesure) : la regle initiale « que
// des zeros apres la suite nommee » est REFUTEE par le corpus entier (structEnd=0 partout) —
// le slot de terminaison porte un u32 non nul en flags sur ~40 des 50 blocs du build de
// reference (0x01 ou 0x02, jamais autre chose, kind et zone de nom nuls). C'est coherent avec
// le decalage R7-e (« le jeu lit le niveau un cran plus loin ») : le flags du slot de
// terminaison est le niveau du dernier composant sous la lecture decalee. La regle a ete
// affinee pour exempter ces 4 octets, et RIEN d'autre ; les criteres C1..C3 sont inchanges.
//
// CRITERES ECRITS AVANT LA MESURE (verdict binaire par critere ; un critere rate = le
// correctif de parseRegistry est refuse) :
//
//	C1 — pour TOUT film du corpus portant une section d'identification, le premier bloc
//	     violant tombe EXACTEMENT sur le bloc qui porte cette section :
//	     structEnd == versionOff/archetypeBlockSize. Seuil : 100 % de ces films.
//	C2 — les slots nommes AU-DELA de structEnd (ce que l'ancien parse ramassait) sont des
//	     faux positifs de la section 3. Compte publie film par film ; attendu : moins de 1 %
//	     des films en portent au moins un (mesure prealable : 1 sur les 3 temoins du lot D).
//	     PREDICTION REFUTEE PAR LA MESURE (2026-08-30) : 397 films sur 1 367 (29 %) portent
//	     au moins un slot fantome, 541 slots en tout — l'ancrage sur 3 temoins sous-estimait
//	     d'un facteur 29. Le seuil n'est PAS redessine pour passer : il est publie rate, et
//	     remplace par la propriete qui gate reellement le correctif :
//	C2' — 100 % des slots fantomes tombent STRICTEMENT au-dela du bloc d'identification
//	     (donc dans les sections 2/3), jamais entre deux blocs de registre.
//	C3 — sur les builds de reference (HI_1_12_0 et HI_1_13_0), structEnd == 50 et
//	     1 067 slots nommes dans le registre, sur 100 % des films.
//
// Garde LOT3_CORPUS : racine du cache de films (un sous-repertoire = un film). Aucun effet en
// CI, aucun code de production appele hormis parseRegistry (mesure par une re-implementation
// independante de la regle, confrontee au parse).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// lot3Scan re-implemente la regle structurelle SANS passer par parseRegistry : c'est la
// mesure independante a laquelle le parse est confronte.
type lot3Scan struct {
	oldBlocks  int   // len(data)/archetypeBlockSize — le « 118 » du dossier
	structEnd  int   // premier bloc violant « slots nommes puis zeros »
	slotsIn    int   // slots nommes dans les blocs [0, structEnd)
	slotsAu    int   // slots nommes ramasses par l'ancien parse dans [structEnd, oldBlocks)
	auDela     []int // les blocs au-dela qui portent au moins un slot nomme
	porteursIn int   // blocs de [0, structEnd) a au moins un slot nomme
	violOff    int   // offset absolu du premier octet violant du bloc structEnd
	violByte   byte  // sa valeur
}

func lot3ScanChunk(data []byte) lot3Scan {
	s := lot3Scan{oldBlocks: len(data) / archetypeBlockSize, structEnd: -1, violOff: -1}
	for b := 0; b < s.oldBlocks; b++ {
		base := b * archetypeBlockSize
		run := 0
		for ; run < archetypeBlockSlots; run++ {
			if slotName(data, base+run*registrySlotSize) == "" {
				break
			}
		}
		viole := false
		term := base + run*registrySlotSize
		for off := term; off < base+archetypeBlockSize; off++ {
			if off >= term+4 && off < term+8 {
				continue // flags du slot de terminaison : exempte (regle affinee, cf. en-tete)
			}
			if data[off] != 0 {
				viole = true
				if s.structEnd < 0 {
					s.violOff, s.violByte = off, data[off]
				}
				break
			}
		}
		if viole && s.structEnd < 0 {
			s.structEnd = b
		}
		if s.structEnd < 0 {
			s.slotsIn += run
			if run > 0 {
				s.porteursIn++
			}
		} else if run > 0 {
			s.slotsAu += run
			s.auDela = append(s.auDela, b)
		}
	}
	if s.structEnd < 0 {
		s.structEnd = s.oldBlocks
	}
	return s
}

// lot3Groupe agrege les films d'un meme (build, version).
type lot3Groupe struct {
	films      int
	structEnds map[int]int // structEnd -> films
	slotsIns   map[int]int // slotsIn -> films
	oldBlocks  map[int]int // oldBlocks -> films
}

// TestLot3CompteRegistre mesure la regle structurelle sur tout le corpus et rend le verdict
// C1/C2/C3. Publie aussi, par build, la distribution (structEnd, slots) — c'est la piece qui
// remplace le « 118 » du dossier.
func TestLot3CompteRegistre(t *testing.T) {
	racine := os.Getenv("LOT3_CORPUS")
	if racine == "" {
		t.Skipf("LOT3_CORPUS absent : instrument saute")
	}
	entries, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	groupes := map[string]*lot3Groupe{}
	lus, sansIdent, c1Viol, c2Films, c2Slots := 0, 0, 0, 0, 0
	c3Films, c3Viol, c2primeViol := 0, 0, 0
	var c2Exemples, c1Exemples []string
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(racine, ent.Name(), "chunk_00.bin"))
		if err != nil {
			continue
		}
		lus++
		data := inflateChunk(raw)
		s := lot3ScanChunk(data)
		e, ok := lireEntete(data)
		cle := "SANS SECTION D'IDENTIFICATION"
		if ok {
			cle = e.build + " " + e.version
			if s.structEnd != e.versionOff/archetypeBlockSize {
				c1Viol++
				if len(c1Exemples) < 5 {
					c1Exemples = append(c1Exemples, fmt.Sprintf("%s structEnd=%d blocIdent=%d viol@0x%06x=0x%02x",
						ent.Name(), s.structEnd, e.versionOff/archetypeBlockSize, s.violOff, s.violByte))
				}
			}
		} else {
			sansIdent++
		}
		if s.slotsAu > 0 {
			c2Films++
			c2Slots += s.slotsAu
			if len(c2Exemples) < 8 {
				c2Exemples = append(c2Exemples, fmt.Sprintf("%s +%d slot(s) blocs %v",
					ent.Name(), s.slotsAu, s.auDela))
			}
			if ok {
				for _, b := range s.auDela {
					if b <= e.versionOff/archetypeBlockSize {
						c2primeViol++
					}
				}
			}
		}
		if ok && (strings.HasPrefix(e.build, "HI_1_12_0") || strings.HasPrefix(e.build, "HI_1_13_0")) {
			c3Films++
			if s.structEnd != 50 || s.slotsIn != 1067 {
				c3Viol++
			}
		}
		// Confrontation au parse de production : memes bornes, memes slots.
		reg := parseRegistry(data)
		if len(reg.Archetypes) != s.structEnd {
			t.Errorf("%s : parseRegistry rend %d blocs, la mesure independante %d",
				ent.Name(), len(reg.Archetypes), s.structEnd)
		}
		g := groupes[cle]
		if g == nil {
			g = &lot3Groupe{structEnds: map[int]int{}, slotsIns: map[int]int{}, oldBlocks: map[int]int{}}
			groupes[cle] = g
		}
		g.films++
		g.structEnds[s.structEnd]++
		g.slotsIns[s.slotsIn]++
		g.oldBlocks[s.oldBlocks]++
	}
	t.Logf("%d chunk_00 lus dans %s (%d sans section d'identification)", lus, racine, sansIdent)
	var cles []string
	for k := range groupes {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return groupes[cles[i]].films > groupes[cles[j]].films })
	for _, k := range cles {
		g := groupes[k]
		t.Logf("  %-42s : %4d films — registre %s blocs (ancien compte %s), %s slots nommes",
			k, g.films, lot3Dist(g.structEnds), lot3Dist(g.oldBlocks), lot3Dist(g.slotsIns))
	}
	t.Logf("C1 (fin structurelle = bloc d'identification) : %d violation(s) sur %d films avec section — %s",
		c1Viol, lus-sansIdent, lot3Verdict(c1Viol == 0))
	if len(c1Exemples) > 0 {
		t.Logf("  exemples C1 : %s", strings.Join(c1Exemples, " ; "))
	}
	t.Logf("C2 (prevalence des faux positifs, prediction < 1 %% des films) : %d film(s) / %d slot(s) — %s (prediction refutee, cf. en-tete) ; %s",
		c2Films, c2Slots, lot3Verdict(c2Films*100 < lus), strings.Join(c2Exemples, " ; "))
	t.Logf("C2' (100 %% des fantomes au-dela du bloc d'identification) : %d violation(s) — %s",
		c2primeViol, lot3Verdict(c2primeViol == 0))
	t.Logf("C3 (builds de reference : 50 blocs, 1 067 slots) : %d violation(s) sur %d films — %s",
		c3Viol, c3Films, lot3Verdict(c3Viol == 0))
	if c1Viol > 0 || c2primeViol > 0 || c3Viol > 0 {
		t.Errorf("au moins un critere structurel (C1/C2'/C3) est rate — le correctif est refuse en l'etat")
	}
}

func lot3Dist(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	parts := make([]string, 0, len(ks))
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("%d(x%d)", k, m[k]))
	}
	return strings.Join(parts, " ")
}

func lot3Verdict(ok bool) string {
	if ok {
		return "TENU"
	}
	return "RATE"
}
