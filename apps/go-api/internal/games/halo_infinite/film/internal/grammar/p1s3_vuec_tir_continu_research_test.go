//go:build research

package grammar

// p1s3_vuec_tir_continu_research_test.go — SONDE P1-S3 (campagne « retours rejeu », 2026-09-23) :
// LE TIR CONTINU EST ECRIT DANS LA VUE DE CONTROLE (vue C), BLOC D ACTION DE L ENTREE DU JOUEUR.
// (mesure seule, aucun code de production ; publication : `p1s3_vuec_publication_research_test.go`.)
//
// CE QUE GHIDRA A LU (HaloInfinite.exe, lecture seule, adresses absolues) :
//
//   - EMISSION. `FUN_14202f3a0` (tir d un barillet) incremente le NUMERO DE TIR du joueur
//     (`FUN_141e2f590(joueur)+0x10`, 8 bits) a CHAQUE tir, puis n emet `action_weapon_fire`
//     (type 36 : `FUN_141fd3b00` -> `FUN_141fd8460(0x1ffffffff, 0x24, ...)`) que si l octet 1 rendu
//     par `FUN_140de87fc` vaut 1. Pour un barillet dont le TYPE DE PREDICTION (`barillet+0x70`,
//     short) vaut 1 ou 3, cet octet est un SEAU A JETONS (`etat_barillet+0x40`, capacite et debit
//     `barillet+0x74`, ou `+0x78` sur le serveur) : a debit nul, AUCUN record, et le compteur avance
//     quand meme. Un tir emis pose `etat_barillet+0xd` bit 2.
//   - ECRIVAIN DE LA VUE C. `FUN_14076a2f4` -> `FUN_1406d175c` -> `FUN_1406d1134` construit le bloc
//     d action (+0x18 du bloc de 0x68 octets) depuis L ARME TENUE (le vehicule pour son pilote,
//     `FUN_1404998d0`) : bit (main, entree) de m0/m2 (`FUN_1431ab1ec`) pour chaque GACHETTE tenue dont
//     le barillet est de type 1 et dont le dernier tir N A PAS ete emis (`FUN_140eb2f20`) ; bit
//     (main, barillet) de m4/m5 (`FUN_1431ab1cc`) pour chaque barillet de type 3 en etat 1.
//   - LECTEUR (Theater). `FUN_14076b838` (code 0xd) -> `FUN_14076b058` -> `FUN_1407ff8cc` ->
//     `FUN_1406dc5b8` : m0 bit0 -> drapeau 31, bits 1|2 -> 32, m4 -> 33/34 (main 1 : 37..40) ;
//     `FUN_14071c31c` -> `FUN_1406db688` : unite+0x308 ; `FUN_1406dba04` : bits 29..34 -> arme+0x2fa
//     (`FUN_140613f98`) ; `FUN_1407fb92c` : gachette d entree 0 « tenue » si bit 0 OU bit 2 ;
//     `FUN_1407fa928` fait tirer le barillet a sa cadence (`FUN_1408a397c`).
//
// LA GRAMMAIRE COMPLETE DE L ENTREE (`FUN_1406cd860`), dont TROIS branches que la production
// refuse (`consumeEntreeControle`) sont resolues ici au site d appel :
//
//	R(1) g ; si g : R(2)                                  (+0x00, +0x01)
//	R(6) R(6) ; R(1) ; si 1 : R(5)                        FUN_1406d6ef4 (largeur 5 @0x1422f6bdb)
//	R(1) ; si 1 : R(6)                                    +0x10 (FUN_1406d84b4 largeur 6 @0x142265fcc)
//	R(1) ; si 1 : R(5)                                    +0x14 (@0x1406cdb43, puis 0x142265fe3 ->
//	                                                      0x1406cd991 : le bloc d action SUIT)
//	FUN_1406d025c                                         le bloc d action, TOUJOURS lu
//
// ET DEUX LECTURES DU BLOC D ACTION QUE LA PRODUCTION (`consume1406d025c`) FAIT FAUX :
//   - la reference typee de queue (`FUN_140c9e4d8` -> `FUN_140c9e990`) : genre 1 -> categorie 1
//     de `FUN_1406d3140` (`MOV R8D,EBP` avec EBP=1, @0x140c9e9c7), genre 2 -> categorie 2
//     (`MOV R8D,0x2` @0x1423e5a69) ; la production lit la categorie 0 dans les deux cas ;
//   - le vecteur du bloc de visee (`FUN_1431a0cbc`) : R(2) mode (`FUN_142af27f8`) ; 1 -> R(19)
//     (`FUN_14076dc04`, R9D=0x13 @0x1431a0cdf) ; 2, 3 -> constante ; 0 -> `FUN_14076e494`, vecteur
//     quantifie dont les largeurs viennent d une table posee a l execution (NON PORTE ici : la
//     lecture du paquet s arrete et le compte le dit).
//
// ORACLE DE CADRAGE, ecrit avant la mesure : la vue C est le DERNIER rang d une trame delta ; une
// lecture juste finit sur le terminateur de la vue C avec un reste de 0 a 7 bits, tous nuls. Une
// entree n est VALIDEE que si son paquet passe cet oracle.
//
// Rejouable (voie film, un film) :
//
//	MOUV511_FILM=<depot>/data/cache/film_chunks/81c02726 \
//	MOUV511_BORNES=<depot>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	MOUV511_CARTE=isolation S3_ORIGINE_US=451125221 S3_INDEX=2 \
//	S3_EPISODES=311-1088,2064-2869 S3_FRAGS=395,623,650,2159,2228,2402 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestP1S3VueCTirContinu$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// s3Entree est UNE entree de controle (kind 0) lue sous la grammaire complete.
type s3Entree struct {
	ts                   uint64
	debut                int // bit de presence de l entree dans le paquet
	trame, index, idx7   int
	valide               bool // le paquet passe l oracle de cadrage
	bloc, g1, actions, c bool
	second, a, b         int
	troisieme, f10       int
	drapeaux             int
	m0, m2, m4, m5, r3   uint64
	arme0, arme1         int  // FUN_1406d00ec gardes : +7 et +8 du bloc d action
	modeVecteur, genre   int  // -1 absent
	vecteurNonPorte      bool // mode 0 du vecteur de visee : arret
	blocB                bool // bloc de 0xbc (FUN_141fdae44) : non porte, la lecture du paquet s arrete
}

// tir rend vrai quand l entree porte une gachette de type 1 tenue ou un barillet de type 3 en tir.
func (e s3Entree) tir() bool { return (e.m0 | e.m2 | e.m4 | e.m5) != 0 }

// Statuts d un paquet delta.
const (
	s3NonLocalise = iota // liste d evenements non localisee
	s3BOuverte           // la vue B n a pas clos sa liste
	s3BClose             // la vue B a clos : la vue C est lue
)

// s3Paquet est le bilan d UN paquet delta.
type s3Paquet struct {
	trame, statut          int
	porteProd, fermeProd   bool // la production (`consumeVueC`)
	porteBrut, fermeBrut   bool // grammaire de l entree complete, bloc d action de la production
	porteMoi, fermeMoi     bool // grammaire complete, bloc d action corrige
	kinds1ou2, blocsB, nb  int
	vecteursNonPortes, gen int
}

// s3LireCible lit `FUN_140c9e4d8` (appele par `FUN_142f26740` avec param_3 = 0) avec la categorie
// relue dans `FUN_140c9e990`. Rend le genre (-1 quand la garde est nulle).
func s3LireCible(br *Lecteur) int {
	if !br.ReadBit() {
		return -1
	}
	genre := int(br.ReadBits(2)) //nolint:gosec // FUN_1407f0278 : R(2)
	switch genre {
	case 1:
		readVarWidthInt(br, 1) // categorie 1, sonde comprise
		if br.ReadBit() {
			br.ReadBits(6)
		}
	case 2:
		readVarWidthInt(br, 2)
	}
	if !br.ReadBit() { // +0x18 bit 0
		br.ReadBits(dequant140c9e4d8Width)
		br.ReadBits(dequant140c9e4d8Width)
		if !br.ReadBit() { // +0x18 bit 1
			return genre
		}
	}
	consume140c9e738(br, false)
	return genre
}

// s3LireVecteur lit `FUN_1431a0cbc`. Rend false sur le mode 0 (non porte).
func s3LireVecteur(br *Lecteur, e *s3Entree) bool {
	e.modeVecteur = int(br.ReadBits(2)) //nolint:gosec // FUN_142af27f8 : R(2)
	switch e.modeVecteur {
	case 0:
		e.vecteurNonPorte = true
		return false
	case 1:
		br.ReadBits(19)
	}
	return true
}

// s3LireActions lit `FUN_1406d025c` en rendant ses champs. `corrige` choisit la queue et le
// vecteur relus dans Ghidra ; sinon ceux de la production (`consume1406d025c`).
func s3LireActions(br *Lecteur, e *s3Entree, corrige bool) bool {
	e.modeVecteur, e.genre = -1, -1
	if e.actions = br.ReadBit(); !e.actions {
		return true
	}
	if br.ReadBit() {
		e.m0, e.m2 = br.ReadBits(3), br.ReadBits(3)
	}
	if br.ReadBit() {
		e.m4, e.m5 = br.ReadBits(2), br.ReadBits(2)
	}
	if e.c = br.ReadBit(); e.c {
		br.ReadBits(2)
		consumeOpt1431a0bbc(br)
		consumeOpt1431a0abc(br)
		if !corrige {
			consumeQuatBlock1431a0cbc(br)
		} else if !s3LireVecteur(br, e) {
			return false
		}
	}
	e.r3 = br.ReadBits(3)
	e.arme0, e.arme1 = -2, -2 // -2 : champ absent (garde nulle), -1 : R(1) = 1
	if (e.m0&0b111) != 0 || (e.m4&0b11) != 0 {
		e.arme0 = s3LireID2(br)
	}
	if (e.m2&0b111) != 0 || (e.m5&0b11) != 0 {
		e.arme1 = s3LireID2(br)
	}
	if corrige {
		e.genre = s3LireCible(br)
	} else {
		ancienneQueue142f26740(br) // la queue d AVANT le lot M4b
	}
	return true
}

// s3LireEntree lit `FUN_1406d0388` (kind 0, branche `FUN_14048ee34() == 0`) et le bloc de 0x68
// octets `FUN_1406cd860` en entier. Rend false quand la lecture ne peut plus avancer.
func s3LireEntree(br *Lecteur, frameLen int, e *s3Entree, corrige bool) bool {
	e.idx7, e.second, e.troisieme, e.f10 = -1, -1, -1, -1
	if br.ReadBit() {
		e.idx7 = int(br.ReadBits(largeurIndexCdc04)) //nolint:gosec // 7 bits
	}
	e.index = int(br.ReadBits(largeurIndexControle)) //nolint:gosec // 5 bits
	if e.bloc = br.ReadBit(); e.bloc {
		if e.g1 = br.ReadBit(); e.g1 {
			e.second = int(br.ReadBits(largeurCourteControle)) //nolint:gosec // 2 bits
		}
		e.a = int(br.ReadBits(LargeurScalaireAnalogique)) //nolint:gosec // 6 bits
		e.b = int(br.ReadBits(LargeurScalaireAnalogique)) //nolint:gosec // 6 bits
		if br.ReadBit() {
			e.troisieme = int(br.ReadBits(5)) //nolint:gosec // FUN_1406d84b4 largeur 5
		}
		if br.ReadBit() {
			e.f10 = int(br.ReadBits(6)) //nolint:gosec // FUN_1406d84b4 largeur 6
		}
		if br.ReadBit() {
			e.drapeaux = int(br.ReadBits(5)) //nolint:gosec // +0x14, R(5)
		}
		if !s3LireActions(br, e, corrige) || br.BitPos() > frameLen {
			return false
		}
	}
	e.blocB = br.ReadBit()
	return !e.blocB
}

// s3LireVueC lit la vue C d un paquet depuis la fin de la vue B. Rend les entrees, le curseur
// final et si la vue a ete lue jusqu a son terminateur.
func s3LireVueC(pay []byte, finB int, e0 s3Entree, bilan *s3Paquet, corrige bool) ([]s3Entree, int, bool) {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.Skip(finB)
	var out []s3Entree
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1) {
			return out, br.BitPos(), false
		}
		debut := br.BitPos()
		if !br.ReadBit() {
			return out, br.BitPos(), true
		}
		if !placeDisponible(br, frameLen, LargeurKindVueC) {
			return out, br.BitPos(), false
		}
		k := int(br.ReadBits(LargeurKindVueC)) //nolint:gosec // 2 bits
		if k == kindVueCNeant {
			continue
		}
		if k != kindVueCControle {
			bilan.kinds1ou2 += s3B(corrige)
			return out, br.BitPos(), false
		}
		e := e0
		e.debut = debut
		ok := s3LireEntree(br, frameLen, &e, corrige)
		out = append(out, e)
		if !ok {
			if corrige {
				bilan.blocsB += s3B(e.blocB)
				bilan.vecteursNonPortes += s3B(e.vecteurNonPorte)
			}
			return out, br.BitPos(), false
		}
	}
	return out, br.BitPos(), false
}

// s3Cadre porte l invocation : horloge, index du pilote, episodes, frags.
type s3Cadre struct {
	origineUS uint64
	index     int
	episodes  [][2]int
	frags     []int
}

func (c s3Cadre) trame(ts uint64) int {
	return int((int64(ts) - int64(c.origineUS)) / 100_000) //nolint:gosec // horloge du film
}

func (c s3Cadre) dansEpisode(tr int) bool {
	for _, e := range c.episodes {
		if tr >= e[0] && tr <= e[1] {
			return true
		}
	}
	return false
}

func s3LireCadre(t *testing.T) s3Cadre {
	t.Helper()
	o, err := strconv.ParseUint(os.Getenv("S3_ORIGINE_US"), 10, 64)
	if err != nil {
		t.Skip("S3_ORIGINE_US absent : instrument de recherche")
	}
	c := s3Cadre{origineUS: o, index: 2}
	if v, errI := strconv.Atoi(os.Getenv("S3_INDEX")); errI == nil {
		c.index = v
	}
	for _, s := range strings.Split(os.Getenv("S3_EPISODES"), ",") {
		var a, b int
		if _, errE := fmt.Sscanf(s, "%d-%d", &a, &b); errE == nil {
			c.episodes = append(c.episodes, [2]int{a, b})
		}
	}
	for _, s := range strings.Split(os.Getenv("S3_FRAGS"), ",") {
		if v, errF := strconv.Atoi(strings.TrimSpace(s)); errF == nil {
			c.frags = append(c.frags, v)
		}
	}
	return c
}

// s3Passe decode le film UNE fois et rend les entrees de controle et le bilan par paquet.
func s3Passe(tc t516Temoin, cad s3Cadre) ([]s3Entree, []s3Paquet) {
	var ents []s3Entree
	var paqs []s3Paquet
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			b := s3Paquet{trame: cad.trame(pk.TimestampUS), statut: s3NonLocalise}
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocate(pay, w, tc.cfg); debut < 0 {
					paqs = append(paqs, b)
					continue
				}
			}
			mar := t519Marcher(pay, w, tc.cfg, debut)
			if b.statut = s3BOuverte; mar.m.HitEndB {
				ents = append(ents, s3LirePaquet(pay, mar.m, pk.TimestampUS, &b)...)
			}
			paqs = append(paqs, b)
		}
	}
	return ents, paqs
}

// s3LirePaquet lit la vue C d un paquet dont la vue B a clos, sous les trois grammaires.
func s3LirePaquet(pay []byte, m t515Marche, ts uint64, b *s3Paquet) []s3Entree {
	b.statut, b.porteProd = s3BClose, m.PorteC
	b.fermeProd = m.PorteC && s3Ferme(pay, m.FinVueC)
	e0 := s3Entree{ts: ts, trame: b.trame}
	_, finBrut, porteBrut := s3LireVueC(pay, m.FinVueB, e0, b, false)
	b.porteBrut, b.fermeBrut = porteBrut, porteBrut && s3Ferme(pay, finBrut)
	es, fin, porte := s3LireVueC(pay, m.FinVueB, e0, b, true)
	b.porteMoi, b.fermeMoi, b.nb = porte, porte && s3Ferme(pay, fin), len(es)
	for i := range es {
		es[i].valide = b.fermeMoi
		b.gen += s3B(es[i].genre >= 0)
	}
	return es
}

// s3Ferme applique l oracle : reste de 0 a 7 bits, tous nuls.
func s3Ferme(pay []byte, curseur int) bool {
	reste := len(pay)*8 - curseur
	return reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, curseur)
}

// TestP1S3VueCTirContinu est la mesure de la sonde P1-S3 (en-tete du fichier).
func TestP1S3VueCTirContinu(t *testing.T) {
	cad := s3LireCadre(t)
	tc := t516Cadre(t)
	ents, paqs := s3Passe(tc, cad)
	s3PublierOracle(t, paqs)
	s3PublierIndex(t, ents, cad)
	for _, seulValide := range []bool{false, true} {
		sel := s3Filtrer(ents, seulValide)
		s3PublierGate(t, sel, paqs, cad, seulValide)
		s3PublierRafales(t, sel, cad, seulValide)
	}
	s3PublierAutourDesFrags(t, ents, cad)
	s3PublierCadence(t, ents, cad)
}

func s3Filtrer(ents []s3Entree, seulValide bool) []s3Entree {
	if !seulValide {
		return ents
	}
	var out []s3Entree
	for _, e := range ents {
		if e.valide {
			out = append(out, e)
		}
	}
	return out
}

// s3PublierOracle publie l oracle de cadrage sous les trois grammaires.
func s3PublierOracle(t *testing.T, paqs []s3Paquet) {
	t.Helper()
	var st [3]int
	var pP, fP, pB, fB, pM, fM, k12, bB, vNP, gen, gagnes, perdus int
	for _, b := range paqs {
		st[b.statut]++
		pP, fP = pP+s3B(b.porteProd), fP+s3B(b.fermeProd)
		pB, fB = pB+s3B(b.porteBrut), fB+s3B(b.fermeBrut)
		pM, fM = pM+s3B(b.porteMoi), fM+s3B(b.fermeMoi)
		k12, bB, vNP, gen = k12+b.kinds1ou2, bB+b.blocsB, vNP+b.vecteursNonPortes, gen+b.gen
		gagnes += s3B(b.fermeMoi && !b.fermeProd)
		perdus += s3B(b.fermeProd && !b.fermeMoi)
	}
	t.Logf("== ORACLE : %d paquets delta · liste non localisee %d · vue B ouverte %d · vue B close %d",
		len(paqs), st[s3NonLocalise], st[s3BOuverte], st[s3BClose])
	t.Logf("   vue C lue jusqu au terminateur : production %d · entree complete %d · + bloc d action "+
		"corrige %d", pP, pB, pM)
	t.Logf("   paquet CLOS (reste 0..7 bits nuls) : production %d · entree complete %d · + bloc d action "+
		"corrige %d (gagnes %d, perdus %d sur la production)", fP, fB, fM, gagnes, perdus)
	t.Logf("   arrets restants : kinds 1/2 %d · bloc de 0xbc %d · vecteur de visee mode 0 %d · references "+
		"typees lues %d", k12, bB, vNP, gen)
}

// s3LireID2 lit `FUN_1406d00ec` en rendant sa valeur : R(1) ; si 0, R(2) ; sinon -1.
func s3LireID2(br *Lecteur) int {
	if br.ReadBit() {
		return -1
	}
	return int(br.ReadBits(2)) //nolint:gosec // 2 bits
}
