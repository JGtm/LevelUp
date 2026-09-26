//go:build research

package grammar

// tcg_tir_continu_research_test.go — SONDE P1 DE LA CAMPAGNE « RETOURS REJEU » (2026-09-23) :
// OU LE FILM PORTE-T-IL LE TIR CONTINU (Ghost) ? (mesure seule, aucun code de production.)
//
// FAIT DE DEPART (`.ai/V7.5/retours_rejeu_2026-09-23/RAPPORT_tirs_vehicules.md`) : `ScanFireEvents`
// ne lit que l evenement de TETE des paquets delta d octet 0 = 0xD2 (types 36 et 37), a offsets
// fixes, et jette les payloads de moins de 113 bits. Sur 81c02726 (Isolation), G MONEY 2123
// (index 2) pilote le Ghost (vies 769 puis 771 ; corps 514 puis 541) de t 311 a 1088 et de 2064 a
// 2869 (axe du document, 100 ms) et y fait 6 frags ; AUCUN evenement de tir de l index 2 dans ses
// episodes.
//
// S1 — LA LISTE COMPLETE. Chaque liste d evenements est marchee par le marcheur R7 (domaines et
// largeurs de charge sourcees de l exe, `r7_*_research_test.go`), en gardant le bit de debut de
// chaque charge. ORACLE DE CADRAGE, ecrit avant la mesure : une marche est VALIDEE quand son
// dernier bit (apres le terminateur de liste) tombe EXACTEMENT sur le debut de la vue B que le
// localisateur de production (`marchLocate`, signature du slot 123) trouve sans rien savoir des
// evenements. Deux chaines sans etape commune ; un seul bit d ecart invalide la liste.
//
// S1 BIS / TER (`tcg_tir_continu_s1bis_*`, `tcg_tir_continu_compteur_*`) : les degats du pilote
// (`damage_aftermath`), les evenements de projectile, et le NUMERO DE TIR par joueur que porte le
// record 36 (ses sauts comptent des tirs numerotes sans record).
//
// S2 — LES COMPOSANTS (`tcg_tir_continu_s2_research_test.go`) : les records des vehicules suivis,
// de leurs pilotes et de tout objet dont `i10 object-parent-state` designe un vehicule suivi.
// S2 BIS a QUATER (`tcg_tir_continu_vuec_*`, `_rafales_*`, `_etat_*`) : la vue de controle (vue C)
// et son bloc d action, confrontes aux tirs a coup decodes et aux touches.
//
// GATE DE LA SONDE (plan §4.2 P1) : un signal dont une occurrence precede CHACUN des frags dans les
// 2 s, absent au temoin decale de +/-60 s, nul hors des episodes.
//
// LECTURE SEULE, un film par invocation, aucun artefact, aucune base. Episodes :
// `t0-t1:corps du pilote:vehicule[:index du pilote]` sur l axe du document (100 ms) ;
// TCG_ORIGINE_US = origine du document en horloge de film (`origineUS` de la sonde des faits).
//
//	MOUV511_FILM=<depot>/data/cache/film_chunks/81c02726 MOUV511_BORNES=<catalogue> \
//	MOUV511_CARTE=isolation TCG_ORIGINE_US=451125221 TCG_INDEX=2 TCG_SOURCE=F712C64A \
//	TCG_EPISODES=311-1088:514:769:2,2064-2869:541:771:2 TCG_FRAGS=395,623,650,2159,2228,2402 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestTirContinuGhost$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/
//
//	MOUV511_FILM=<depot>/data/cache/film_chunks/8a485699 MOUV511_CARTE="launch site" \
//	TCG_ORIGINE_US=2331882869 TCG_INDEX=4 TCG_SOURCE=FA4FAD21 TCG_FRAGS=401,1187 \
//	TCG_EPISODES=745-1169:516:771:4,1672-1800:535:775:5,2334-2449:545:775:7,3099-3205:558:780:7,\
//	93-334:516:774:4,160-401:512:770:0,4454-4943:581:782:0,4674-4820:571:783:2,5101-5178:588:777:7

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// tcgPasUS est le pas de l axe du document (100 ms).
const tcgPasUS = 100_000

// tcgAvantFrag est la fenetre du gate : 2 s avant le frag (20 pas), jusqu au frag inclus.
const tcgAvantFrag = 20

// tcgDecalage est le decalage du temoin : 60 s (600 pas), des deux cotes.
const tcgDecalage = 600

// tcgLoPerso est la moitie basse commune des armes personnelles (famille des GlobalID de variante).
const tcgLoPerso = uint64(0x42C9679F)

// tcgEpisode est un episode de pilotage : bornes sur l axe du document, corps du pilote, vehicule.
type tcgEpisode struct {
	t0, t1   int
	pilote   uint32
	vehicule uint32
	index    int // index de controle / de tireur du pilote (TCG_INDEX par defaut)
}

// tcgCadre porte ce que l invocation donne : horloge, episodes, frags, index du pilote.
type tcgCadre struct {
	origineUS uint64
	episodes  []tcgEpisode
	frags     []int
	index     int
}

// tcgLireCadre lit l environnement de la sonde.
func tcgLireCadre(t *testing.T) tcgCadre {
	t.Helper()
	var c tcgCadre
	o, err := strconv.ParseUint(os.Getenv("TCG_ORIGINE_US"), 10, 64)
	if err != nil {
		t.Skip("TCG_ORIGINE_US absent : instrument de recherche")
	}
	c.origineUS = o
	c.index, _ = strconv.Atoi(os.Getenv("TCG_INDEX"))
	for _, s := range strings.Split(os.Getenv("TCG_EPISODES"), ",") {
		e := tcgEpisode{index: c.index}
		n, _ := fmt.Sscanf(s, "%d-%d:%d:%d:%d", &e.t0, &e.t1, &e.pilote, &e.vehicule, &e.index)
		if n >= 4 {
			c.episodes = append(c.episodes, e)
		}
	}
	for _, s := range strings.Split(os.Getenv("TCG_FRAGS"), ",") {
		if f, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			c.frags = append(c.frags, f)
		}
	}
	return c
}

// trame place un horodatage de paquet sur l axe du document.
func (c tcgCadre) trame(ts uint64) int {
	if ts < c.origineUS {
		return -int((c.origineUS-ts)/tcgPasUS) - 1
	}
	return int((ts - c.origineUS) / tcgPasUS)
}

// episodeDe rend l index de l episode qui couvre la trame (marge m), ou -1.
func (c tcgCadre) episodeDe(tr, m int) int {
	for i, e := range c.episodes {
		if tr >= e.t0-m && tr <= e.t1+m {
			return i
		}
	}
	return -1
}

// indexDe rend l index du pilote de l episode qui couvre la trame (marge 10), sinon TCG_INDEX.
func (c tcgCadre) indexDe(tr int) int {
	if i := c.episodeDe(tr, 10); i >= 0 {
		return c.episodes[i].index
	}
	return c.index
}

// suivi dit si un slot est un pilote ou un vehicule d un episode.
func (c tcgCadre) suivi(slot uint32) bool {
	for _, e := range c.episodes {
		if slot == e.pilote || slot == e.vehicule {
			return true
		}
	}
	return false
}

// vehiculeSuivi dit si un slot est un vehicule d un episode.
func (c tcgCadre) vehiculeSuivi(slot uint32) bool {
	for _, e := range c.episodes {
		if slot == e.vehicule {
			return true
		}
	}
	return false
}

// tcgRef est une reference gardee lue dans l en-tete d un evenement.
type tcgRef struct {
	present bool
	dom     int
	largeur uint
	idx     uint64
	gen     uint32
}

// slot rend le slot designe, BASE DU DOMAINE COMPRISE (`varwidth.go`, `FUN_140d10bb0` : 0x200 pour
// les domaines 0, 1, 2 et 4 — le domaine 1 a sonde tombe sur l entree 4, meme base —, 0x300 pour
// le 3, 0x400 pour le 5, 0 pour les domaines 6, 7 et 8).
func (r tcgRef) slot() uint32 {
	switch r.dom {
	case 0, 1, 2, 4:
		return 0x200 + uint32(r.idx) //nolint:gosec // index sur 13 bits au plus
	case 3:
		return 0x300 + uint32(r.idx) //nolint:gosec // index sur 8 bits au plus
	case 5:
		return 0x400 + uint32(r.idx) //nolint:gosec // index sur 8 bits au plus
	}
	return uint32(r.idx) //nolint:gosec // index sur 13 bits au plus
}

// String rend la reference lisible : domaine, largeur, slot.
func (r tcgRef) String() string {
	if !r.present {
		return "-"
	}
	return fmt.Sprintf("d%d/%d:%d", r.dom, r.largeur, r.slot())
}

// tcgTir est la charge d un `action_weapon_fire` (type 36), LUE champ par champ dans l ordre exact
// de `r7Charge36` (memes bits : l egalite des bits de fin est verifiee a chaque evenement).
type tcgTir struct {
	court, bloc  bool
	attaquant    uint64 // R(7) puis R(1) (`FUN_141fcf670`)
	idx5         int    // porte inversee [R(1) ; si 0 : R(5)] ; -1 si absent
	p2           int    // porte inversee [R(1) ; si 0 : R(2)] ; -1 si absent
	aHaut        bool
	haut, bas    uint64
	nComp, nCib  int
	cibles       []uint32
	desaccord    bool // la fin lue ici differe de celle du marcheur R7
	queueCourte  uint64
	visee        uint64
	aVisee, fini bool
}

// arme rend l identifiant d arme 64 bits (haut << 32 | bas), comme `FireEvent.WeaponID`.
func (x tcgTir) arme() uint64 { return x.haut<<32 | x.bas }

// classe range l arme : perso, vehicule (moitie basse nulle, haute posee), nul, autre.
func (x tcgTir) classe() string {
	switch {
	case x.bas == tcgLoPerso:
		return "perso"
	case x.bas == 0 && x.aHaut && x.haut != 0:
		return "vehicule"
	case x.bas == 0 && x.haut == 0:
		return "nul"
	}
	return "autre"
}

// tcgDecoder36 lit la charge du type 36 champ par champ (miroir exact de `r7Charge36`).
func tcgDecoder36(br *Lecteur, ctx r7Ctx) tcgTir {
	x := tcgTir{idx5: -1, p2: -1}
	x.court, x.bloc = br.ReadBit(), br.ReadBit()
	x.attaquant = br.ReadBits(8)
	if !br.ReadBit() {
		x.idx5 = int(br.ReadBits(5)) //nolint:gosec // 5 bits
	}
	if !br.ReadBit() {
		x.p2 = int(br.ReadBits(2)) //nolint:gosec // 2 bits
	}
	if br.ReadBit() {
		x.aHaut, x.haut = true, br.ReadBits(32)
	}
	x.bas = br.ReadBits(32)
	br.Skip(2)
	blocHoro := false
	if x.bloc {
		br.Skip(1)
		if blocHoro = br.ReadBit(); blocHoro && br.ReadBit() {
			br.Skip(10)
		}
	}
	if x.court {
		x.queueCourte, x.fini = br.ReadBits(10), true
		return x
	}
	x.nComp, x.nCib = r7Comptes(br)
	kindUn := map[int]bool{}
	for i := 0; i < x.nCib; i++ {
		kindUn[i] = br.ReadBits(2) == 1
		br.Skip(1)
		w := uint(13)
		if br.ReadBit() {
			w = 9
		}
		x.cibles = append(x.cibles, 0x200+uint32(br.ReadBits(w))) //nolint:gosec // 13 bits
		br.Skip(2)
	}
	tcgComposantes36(br, x.nComp, x.nCib, kindUn, &x)
	x.fini = r7Queue36(br, ctx, x.bloc, blocHoro)
	return x
}

// tcgComposantes36 lit la boucle des composantes puis les blocs O, P et la visee (miroir de
// `r7Charge36`).
func tcgComposantes36(br *Lecteur, nComp, nCib int, kindUn map[int]bool, x *tcgTir) {
	base := 4
	if nComp == 1 {
		base = 12
	}
	dernierQ, vuQ := uint64(1), false
	for i := 0; i < nComp; i++ {
		br.Skip(4)
		if !br.ReadBit() {
			continue
		}
		dernierQ, vuQ = br.ReadBits(3), true
		idx := 0
		if nCib < 3 {
			idx = int(br.ReadBits(1)) //nolint:gosec // 1 bit
		} else {
			idx = int(br.ReadBits(4)) //nolint:gosec // 4 bits
		}
		br.Skip(16)
		w := base
		if kindUn[idx] && w > 6 {
			w = 6
		}
		br.Skip(3 * w)
	}
	if !(vuQ && dernierQ == 0) {
		c1 := r7Composite(br, true, 15, 7)
		r7Composite(br, false, 0, 0)
		if !c1 {
			x.aVisee, x.visee = true, br.ReadBits(30)
		}
	}
}

// tcgEv est UN evenement lu dans une liste.
type tcgEv struct {
	trame, chunk, paquet, pos, typ int
	ts                             uint64
	refs                           [3]tcgRef
	charge, fin, bitsPaquet        int // fin = -1 : charge opaque (la marche s y arrete)
	valide                         bool
	tir                            *tcgTir
	degat                          *lot1DmgResult
	arme                           uint64 // type 37 : identifiant d arme lu
	aArme                          bool
	// tag / variante : le bloc variante des types 5, 6 et 7 (`r7BlocVariante`, porte inversee) —
	// [R(1) g ; si 0 : [R(1) h ; si 1 : R(32) tag] + R(32) variante].
	tag, variante  uint64
	aTag, aVariant bool
}

// refSuivie dit si l une des references designe un slot suivi.
func (e tcgEv) refSuivie(c tcgCadre) bool {
	for _, r := range e.refs {
		if r.present && c.suivi(r.slot()) {
			return true
		}
	}
	return false
}

// tcgMarcherListe marche la liste d un paquet (miroir de `r7MarcheDecalee` a decalage nul) et garde
// le debut de chaque charge. Un evenement OPAQUE est rendu avec ses references (lues avant la
// charge, donc au cadrage de la marche) et `fin = -1`.
func tcgMarcherListe(pay []byte, ctx r7Ctx) ([]tcgEv, r7Stop, int) {
	br := LecteurSur(pay)
	br.Skip(1)
	var evs []tcgEv
	for pos := 1; ; pos++ {
		if br.Remaining() < 1 {
			return evs, r7StopBuffer, br.BitPos()
		}
		if !br.ReadBit() {
			return evs, r7StopFin, br.BitPos()
		}
		if br.Remaining() < 7 {
			return evs, r7StopBuffer, br.BitPos()
		}
		typ := int(br.ReadBits(7)) //nolint:gosec // 7 bits
		if typ >= 123 {
			return evs, r7StopTypeInconnu, br.BitPos()
		}
		refs, ok := r7Refs3(br, typ)
		if !ok {
			return evs, r7StopSansDomaine, br.BitPos()
		}
		e := tcgEv{pos: pos, typ: typ, charge: br.BitPos(), fin: -1}
		for i, r := range refs {
			e.refs[i] = tcgRef{present: r.Present, dom: r.Dom, largeur: r.Width, idx: r.Index, gen: r.Gen}
		}
		if !r7SkipCharge(br, typ, ctx) {
			return append(evs, e), r7StopOpaque, br.BitPos()
		}
		if br.BitPos() > len(pay)*8 {
			return append(evs, e), r7StopBuffer, br.BitPos()
		}
		e.fin = br.BitPos()
		evs = append(evs, e)
	}
}

// tcgDetailler decode la charge des types qui portent ce que la sonde cherche (36, 0, 37).
func tcgDetailler(pay []byte, e *tcgEv, ctx r7Ctx) {
	br := LecteurSur(pay)
	br.Skip(e.charge)
	switch e.typ {
	case 36:
		x := tcgDecoder36(br, ctx)
		x.desaccord = e.fin >= 0 && br.BitPos() != e.fin
		e.tir = &x
	case 0:
		r := lot1DecodeDamageAftermath(br)
		e.degat = &r
	case 37:
		var haut uint64
		if br.ReadBit() {
			haut = br.ReadBits(32)
		}
		e.arme, e.aArme = haut<<32|br.ReadBits(32), true
	case 5, 6, 7:
		if e.typ == 5 {
			br.Skip(6) // R(6) type/enum (FUN_140809454), avant le bloc variante
		}
		if !br.ReadBit() {
			if br.ReadBit() {
				e.tag, e.aTag = br.ReadBits(32), true
			}
			e.variante, e.aVariant = br.ReadBits(32), true
		}
	}
}

// tcgGarder dit si l evenement entre dans le releve : types 36, 37, 0, 10, 35, et tout evenement
// dont une reference designe un slot suivi.
func tcgGarder(e tcgEv, c tcgCadre) bool {
	switch e.typ {
	case 0, 5, 6, 7, 10, 35, 36, 37:
		return true
	}
	return e.refSuivie(c)
}

// tcgCtxDeCarte construit le contexte de marche R7 depuis l entree du catalogue de production.
func tcgCtxDeCarte(min, max [3]float32) r7Ctx {
	return r7Ctx{
		etendues: [3]float64{float64(max[0] - min[0]), float64(max[1] - min[1]),
			float64(max[2] - min[2])},
		regionBits: 1,
		hasMap:     true,
	}
}

// tcgCompteTri rend une distribution entier -> compte, triee par compte decroissant.
func tcgCompteTri(m map[int]int, nom func(int) string, top int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if m[cles[i]] != m[cles[j]] {
			return m[cles[i]] > m[cles[j]]
		}
		return cles[i] < cles[j]
	})
	var parts []string
	for i, k := range cles {
		if top > 0 && i >= top {
			parts = append(parts, fmt.Sprintf("(+%d)", len(cles)-top))
			break
		}
		parts = append(parts, fmt.Sprintf("%s:%d", nom(k), m[k]))
	}
	return strings.Join(parts, " ")
}

// tcgNomType rend le nom d un type d evenement.
func tcgNomType(k int) string { return fmt.Sprintf("%d %s", k, r7Noms[k]) }

// TestTirContinuGhost joue la passe unique du film et publie S1 puis S2.
func TestTirContinuGhost(t *testing.T) {
	cad := tcgLireCadre(t)
	tc := t516Cadre(t)
	entree := m511Entree(t)
	p := tcgMarcher(tc, cad, tcgCtxDeCarte(entree.Min, entree.Max))
	tcgS1Recensement(t, p)
	tcgS1Tirs(t, p)
	tcgS1Suivis(t, p)
	tcgS1Frags(t, p)
	tcgS1Compteur(t, p)
	tcgS1Degats(t, p)
	tcgS1Projectiles(t, p)
	tcgS1Compteurs(t, p)
	tcgS1TypesDePaquet(t, tc, cad)
	tcgS2Rapport(t, p, tc.reg)
	tcgS2VueC(t, p)
	tcgS2NaissancesParType(t, p)
	tcgS2Blocs(t, p)
	tcgS2Couverture(t, p)
	tcgS2Rafales(t, p)
	tcgS2Chronologie(t, p)
	tcgS2Transitions(t, p)
	tcgS2TirsVehicule(t, p)
	tcgS2Episodes(t, p)
	tcgS2Silences(t, p)
}
