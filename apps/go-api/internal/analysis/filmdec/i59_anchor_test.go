package filmdec

// i59_anchor_test.go — INSTRUMENT DE MESURE de la PHASE 0 du plan
// .ai/V7.5/replay2d/PLAN_GRAPPIN_LIGNE.md : PROUVER le port du corps tag==3 d'i59
// (FUN_142f25e90, components_biped_anchor.go) puis CONTRÔLER que ce qu'il lit est bien
// l'ANCRE du grappin.
//
// QUATRE TESTS, TROIS QUESTIONS :
//
//	TestI59AnchorBodyDump / TestI59AnchorTemplate (0.1) — la GRAMMAIRE. Le champ i59
//	complet de chaque record tag==3, borné par le record biped suivant du même paquet,
//	imprimé bit à bit puis empilé par classe de longueur (consensus 0/1/. par position).
//	C'est CETTE mesure qui a établi la grammaire du corps (deux classes, 64 et 164 bits
//	pleins sur 000d5950, portes lues en clair) — le décompilé n'en donnait que les
//	feuilles (largeurs 24/12, 24, 9 confirmées ; enrobage faux, écart consigné dans
//	components_biped_anchor.go).
//
//	TestI59AnchorWalkProof (0.2) — la MARCHE. Records delta biped dont le masque annonce
//	i59 : marche COMPLÈTE du masque (désers de production), AVANT le port (corps
//	désactivé — la ligne de base historique) puis APRÈS. Le TÉMOIN est la population
//	tag!=3 des MÊMES masques. CRITÈRE, ÉNONCÉ AVANT : après le port, l'écart entre la fin
//	de marche et le record biped suivant tombe à 0 EXACTEMENT, comme le témoin ; un écart
//	négatif falsifie la grammaire. C'est la cohérence AVAL du patron
//	repairUnportedComponent, en plus fort (égalité, pas seulement non-chevauchement).
//
//	TestI59AnchorControls (0.3) — la POSITION. Contrôles énoncés AVANT la mesure :
//	(a) l'ancre candidate tombe dans l'emprise du nuage des bipedes du même film (AABB
//	    des positions de production — contrôle non circulaire établi) ;
//	(b) l'ancre est DEVANT : distance(joueur -> ancre) plausible d'un tir de grappin puis
//	    DÉCROISSANTE sur la trajectoire de la même vie (le grappin TIRE le joueur), contre
//	    un témoin mélangé (ancre d'un événement contre la trajectoire d'un autre) ;
//	(c) les PAIRES à ~0,15 s : distribution des écarts, et si les deux membres portent des
//	    quanta différents — c'est ce qui décide de la fenêtre de rendu.
//	La MAGNITUDE de FUN_14076d528 dépend d'une plage (min, max) que le décompilé ne donne
//	pas : elle est MESURÉE ici par la fixité de l'ancre entre membres d'une paire (deux
//	instants, un seul point monde -> moindres carrés sur (m1, m2), puis ajustement de la
//	loi log/exp de FUN_14076d6dc sur les couples (quantum, magnitude)). Jamais devinée.
//
// LECTURE SEULE, gardé par I59A_FILM, sauté partout ailleurs (CI comprise). La carte est
// AUTO-DÉTECTÉE par la signature des largeurs d'axe du film contre le catalogue versionné
// (I59A_MAP la force si la signature est ambiguë). UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I59A_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI59Anchor' -timeout 60m -v

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	i59aFilmEnv = "I59A_FILM"
	i59aMapEnv  = "I59A_MAP"
)

// i59aPairGapUS : au-delà de cet écart, deux lectures tag==3 d'un même slot ne forment
// plus une paire (mesure phase E : ~0,15 s ; la borne laisse voir la queue de distribution).
const i59aPairGapUS = 500_000

// i59aEvent est une lecture tag==3 datée et attribuée à sa vie.
type i59aEvent struct {
	slot uint32
	tsUS uint64
	st   AbilityNonPredictedState
}

// i59aSetup charge le film ET installe la précision world-object DE CE FILM (largeurs
// d'axe de la carte, lues par le corps d'i59) — restaurée en fin de test. C'est le même
// geste que la production (installWorldObjectPrecision sous le verrou).
func i59aSetup(t *testing.T, dir string) eaFilmSetup {
	t.Helper()
	s := eaSetupBiped(t, dir)
	prev := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(s.lay)
	t.Cleanup(func() { WorldObjectPrecision = prev })
	t.Logf("précision world-object installée depuis le film : %v", WorldObjectPrecision.AxisW)
	return s
}

func i59aIndex(t *testing.T, s eaFilmSetup) int {
	t.Helper()
	idx := s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state-component")
	if idx < 0 {
		idx = s.arch.indicesOfFirst("biped-spartan-ability-non-predicted-state")
	}
	if idx < 0 {
		t.Fatalf("biped-spartan-ability-non-predicted-state absent de l'archétype — composants : %v",
			s.arch.Components)
	}
	return idx
}

// ---------------------------------------------------------------------------
// 0.2 — LA PREUVE DE MARCHE
// ---------------------------------------------------------------------------

type i59aPass struct {
	rec3, recOther int            // records par tag externe (publié au vol par le déser)
	ok3, okOther   int            // marche du masque COMPLET aboutie
	bodyBroke      int            // corps tag==3 parcourus mais cassés (interne non porté)
	inner          map[int]int    // histogramme du tag interne (tag==3 seulement)
	after59        map[int]int    // composants du masque APRÈS i59 (dit si l'aval existe)
	broke          map[string]int // composant qui casse la marche (population tag==3)
	unread         int            // marche cassée AVANT i59 : tag externe inconnu
	// gap3 / gapOther : écart (bits) entre la fin de marche et le PROCHAIN record biped
	// reconnu du même paquet. Un écart NÉGATIF = la marche a débordé dans le record
	// suivant : l'hypothèse est falsifiée sur ce record. La population tag!=3 (corps connu
	// bit-exact) CALIBRE la distribution attendue.
	gap3, gapOther   []int
	over3, overOther int // écarts négatifs (chevauchements)
}

func TestI59AnchorWalkProof(t *testing.T) {
	dir := os.Getenv(i59aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59aFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := i59aSetup(t, dir)
	idx59 := i59aIndex(t, s)
	defer SetAbilityAnchorBodyPorted(true)

	t.Log("CRITÈRE (énoncé avant la mesure) : APRÈS le port, chaque record tag==3 doit " +
		"finir EXACTEMENT au début du record biped suivant (écart 0, comme le témoin " +
		"tag!=3) ; AVANT le port, il manque le corps entier (écart = sa largeur). Un " +
		"écart négatif = chevauchement = grammaire falsifiée sur ce record.")

	SetAbilityAnchorBodyPorted(false)
	i59aLogPass(t, "AVANT (corps désactivé, ligne de base)", i59aWalkPass(s, idx59))

	SetAbilityAnchorBodyPorted(true)
	i59aLogPass(t, "APRÈS (grammaire mesurée)", i59aWalkPass(s, idx59))
}

// i59aMatch est un record biped reconnu dans un paquet (position du motif comprise).
type i59aMatch struct {
	pos, i0 int
	idx     []int
}

// i59aWalkPass balaye les records delta biped dont le masque annonce i59 et marche le
// masque COMPLET avec les désers de production. Le tag externe est publié au vol par le
// déser d'i59 (hook) — c'est lui qui classe le record. Les records de chaque paquet sont
// d'abord TOUS localisés (le motif ne dépend pas de l'hypothèse) : le PROCHAIN record
// borne la longueur vraie du record courant.
func i59aWalkPass(s eaFilmSetup, idx59 int) i59aPass {
	p := i59aPass{inner: map[int]int{}, broke: map[string]int{}, after59: map[int]int{}}
	var capt struct {
		st  AbilityNonPredictedState
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { capt.st, capt.got = st, true })
	defer SetAbilityNonPredictedHook(prev)
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			matches := i59aMatches(pay, total, s)
			for mi, m := range matches {
				if !eaMaskHas(m.idx, idx59) {
					continue
				}
				capt.got = false
				end, brokeName, okWalk := i59aWalkFull(pay, m.i0, total, m.idx, s)
				next := -1
				if mi+1 < len(matches) {
					next = matches[mi+1].pos
				}
				i59aAccount(&p, m, end, next, capt.st, capt.got, brokeName, okWalk, idx59)
			}
		}
	}
	return p
}

// i59aMatches localise tous les records biped d'un paquet (même balayage que la
// production : avance d'i0+TotalBits après un motif reconnu, bit à bit sinon).
func i59aMatches(pay []byte, total int, s eaFilmSetup) []i59aMatch {
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	var out []i59aMatch
	for pos := 0; pos+minRecord <= total; {
		i0, _, idx, ok := matchBipedHeader(pay, pos, total, s.slots, true, s.lay)
		if !ok {
			pos++
			continue
		}
		out = append(out, i59aMatch{pos: pos, i0: i0, idx: idx})
		pos = i0 + s.lay.TotalBits()
	}
	return out
}

// i59aWalkTo marche les composants du masque JUSQU'À la cible exclue et rend la position
// EXACTE du premier bit de la cible — aucune arithmétique inverse depuis une fin de
// marche (la largeur de la cible n'entre pas en jeu).
func i59aWalkTo(pay []byte, i0, total int, idx []int, s eaFilmSetup, target int) (int, bool) {
	at := i0 + s.lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return 0, false
		}
		if id == target {
			return at, true
		}
		name := s.arch.component(id)
		if name == "" {
			return 0, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), s.arch.Level(id))
		if !ported || br.BitPos() > total {
			return 0, false
		}
		at = br.BitPos()
	}
	return 0, false
}

// i59aWalkFull marche TOUS les composants du masque et rend le bit de fin.
func i59aWalkFull(pay []byte, i0, total int, idx []int, s eaFilmSetup) (end int, broke string, ok bool) {
	at := i0 + s.lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return at, "", false
		}
		name := s.arch.component(id)
		if name == "" {
			return at, fmt.Sprintf("i%d(sans nom au registre)", id), false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), s.arch.Level(id))
		if !ported || br.BitPos() > total {
			return br.BitPos(), fmt.Sprintf("i%d %s", id, name), false
		}
		at = br.BitPos()
	}
	return at, "", true
}

// i59aAccount impute une marche au bon compteur de la passe. next est la position du
// PROCHAIN record biped du paquet (-1 si aucun) : l'écart next-end borne la longueur
// vraie du record — négatif, la marche a débordé, l'hypothèse est falsifiée là.
func i59aAccount(p *i59aPass, m i59aMatch, end, next int,
	st AbilityNonPredictedState, got bool, brokeName string, okWalk bool, idx59 int) {
	if !got {
		p.unread++
		return
	}
	if st.Tag != 3 {
		p.recOther++
		if okWalk {
			p.okOther++
			if next >= 0 {
				p.gapOther = append(p.gapOther, next-end)
				if next < end {
					p.overOther++
				}
			}
		}
		return
	}
	p.rec3++
	for _, id := range m.idx {
		if id > idx59 {
			p.after59[id]++
		}
	}
	if st.BodyWalked {
		if !st.BodyOK {
			p.bodyBroke++
		}
		p.inner[st.Inner]++
	}
	switch {
	case okWalk:
		p.ok3++
		if next >= 0 {
			p.gap3 = append(p.gap3, next-end)
			if next < end {
				p.over3++
			}
		}
	case brokeName != "":
		p.broke[brokeName]++
	}
}

func i59aLogPass(t *testing.T, label string, p i59aPass) {
	t.Helper()
	t.Logf("== %s ==", label)
	t.Logf("  tag==3 : %d records · marche aboutie %d · corps cassés %d · CHEVAUCHEMENTS %d/%d · écarts %s",
		p.rec3, p.ok3, p.bodyBroke, p.over3, len(p.gap3), i59aQuantiles(p.gap3))
	t.Logf("  tag!=3 (TÉMOIN, mêmes masques) : %d records · aboutie %d · chevauchements %d/%d · écarts %s",
		p.recOther, p.okOther, p.overOther, len(p.gapOther), i59aQuantiles(p.gapOther))
	if p.unread > 0 {
		t.Logf("  marche cassée AVANT i59 (tag inconnu) : %d", p.unread)
	}
	if len(p.inner) > 0 {
		t.Logf("  tags internes (tag==3) : %s", i59aRenderInt(p.inner))
	}
	if len(p.after59) > 0 {
		t.Logf("  composants APRÈS i59 dans les masques tag==3 : %s", i59aRenderInt(p.after59))
	} else if p.rec3 > 0 {
		t.Log("  AUCUN composant après i59 dans les masques tag==3 : l'aval du record est le record SUIVANT")
	}
	for name, n := range p.broke {
		t.Logf("  casse la marche : %-52s %d", name, n)
	}
}

// i59aQuantiles rend p10/p50/p90 et min d'une liste d'écarts.
func i59aQuantiles(v []int) string {
	if len(v) == 0 {
		return "(aucun)"
	}
	s := append([]int(nil), v...)
	sort.Ints(s)
	return fmt.Sprintf("min %d · p10 %d · p50 %d · p90 %d (n=%d)",
		s[0], s[len(s)/10], s[len(s)/2], s[len(s)*9/10], len(s))
}

func i59aRenderInt(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := ""
	for _, k := range keys {
		out += fmt.Sprintf("%d:%d ", k, h[k])
	}
	return out
}

// ---------------------------------------------------------------------------
// 0.1 — LIRE LES CORPS EN CLAIR (l'assise de la grammaire mesurée)
//
// Chronologie de la mesure du 2026-08-16 : (1) la passe de marche a établi que le masque
// des records tag==3 s'arrête à i59 et que le témoin tag!=3 finit à 3 bits EXACTEMENT du
// record biped suivant (p10=p50=p90=3 sur 988 — c'est la queue R(3) d'i59, jamais lue
// tant que le déser lisait le global recordStateParam au lieu de paramForComponent) ;
// (2) les hypothèses de tag interne bâties sur le décompilé n'expliquaient que 2 à 4 des
// ~62/162 bits réels du corps ; (3) ces deux tests ont donc IMPRIMÉ les champs alignés,
// bit à bit — la grammaire du port est LUE ici, pas conjecturée.
// ---------------------------------------------------------------------------

func TestI59AnchorBodyDump(t *testing.T) {
	dir := os.Getenv(i59aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59aFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := i59aSetup(t, dir)
	idx59 := i59aIndex(t, s)
	SetAbilityAnchorBodyPorted(false) // marche minimale : R(2) + R(3), le corps reste à lire
	defer SetAbilityAnchorBodyPorted(true)

	var capt struct {
		st  AbilityNonPredictedState
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { capt.st, capt.got = st, true })
	defer SetAbilityNonPredictedHook(prev)

	lens := map[int]int{}
	dumped := 0
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			matches := i59aMatches(pay, total, s)
			for mi, m := range matches {
				if mi+1 >= len(matches) || !eaMaskHas(m.idx, idx59) {
					continue
				}
				// Position EXACTE du champ i59 (marche jusqu'à la cible EXCLUE) : aucune
				// arithmétique inverse depuis une fin de marche. Le tag externe est lu en
				// clair — seuls les champs tag==3 sont dumpés.
				at, ok := i59aWalkTo(pay, m.i0, total, m.idx, s, idx59)
				if !ok || readBitsAt(pay, at, 2) != 3 {
					continue
				}
				// Le champ va du tag externe au record suivant moins le pied (3 bits,
				// mesuré p10=p50=p90=3 sur le témoin tag!=3).
				fieldLen := matches[mi+1].pos - 3 - at
				lens[fieldLen]++
				if fieldLen > 0 && fieldLen <= 400 && dumped < 40 {
					dumped++
					t.Logf("slot %-5d t=%8.2fs len=%3d  %s", i59aMatchSlot(pay, m),
						float64(pk.TimestampUS)/1e6, fieldLen, i59aBits(pay, at, fieldLen))
				}
			}
		}
	}
	keys := make([]int, 0, len(lens))
	for k := range lens {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		t.Logf("longueur de corps %3d bits : %d records", k, lens[k])
	}
}

// i59aMatchSlot relit le slot du motif (le walk n'en a pas besoin, le dump si).
func i59aMatchSlot(pay []byte, m i59aMatch) uint32 {
	return readBitsAt(pay, m.pos+1, bipedSlotBits)
}

// TestI59AnchorFilmInfo imprime le découpage i0 du film et les cartes candidates du
// catalogue (signature des largeurs d'axe) — le renseignement qui teste l'hypothèse
// « des champs du corps i59 sont à largeur de CARTE ».
func TestI59AnchorFilmInfo(t *testing.T) {
	dir := os.Getenv(i59aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59aFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible : %v", err)
	}
	t.Logf("découpage i0 : AxisW=%v (somme %d), GateBits=%d",
		lay.AxisW, lay.AxisW[0]+lay.AxisW[1]+lay.AxisW[2], lay.GateBits)
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	var hits []string
	for name, e := range cat.Maps {
		if e.AxisWidths == lay.AxisW {
			hits = append(hits, name)
		}
	}
	sort.Strings(hits)
	t.Logf("cartes candidates par signature : %v", hits)
}

// TestI59AnchorTemplate empile les corps par classe de longueur et publie le CONSENSUS
// par position de bit : 0/1 = bit constant sur toute la classe (structure : portes,
// tags), . = bit variable (charge utile quantifiée). C'est la carte des champs, lue au
// lieu d'être conjecturée.
func TestI59AnchorTemplate(t *testing.T) {
	dir := os.Getenv(i59aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59aFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := i59aSetup(t, dir)
	idx59 := i59aIndex(t, s)
	SetAbilityAnchorBodyPorted(false)
	defer SetAbilityAnchorBodyPorted(true)
	var capt struct {
		st  AbilityNonPredictedState
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { capt.st, capt.got = st, true })
	defer SetAbilityNonPredictedHook(prev)

	classes := map[int][][]byte{} // longueur -> corps (bits en octets 0/1)
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			matches := i59aMatches(pay, total, s)
			for mi, m := range matches {
				if mi+1 >= len(matches) || !eaMaskHas(m.idx, idx59) {
					continue
				}
				at, ok := i59aWalkTo(pay, m.i0, total, m.idx, s, idx59)
				if !ok || readBitsAt(pay, at, 2) != 3 {
					continue
				}
				fieldLen := matches[mi+1].pos - 3 - at
				if fieldLen <= 0 || fieldLen > 400 {
					continue
				}
				bits := make([]byte, fieldLen)
				for i := 0; i < fieldLen; i++ {
					bits[i] = byte(readBitsAt(pay, at+i, 1))
				}
				classes[fieldLen] = append(classes[fieldLen], bits)
			}
		}
	}
	lens := make([]int, 0, len(classes))
	for k := range classes {
		lens = append(lens, k)
	}
	sort.Ints(lens)
	for _, l := range lens {
		rows := classes[l]
		tpl := make([]byte, l)
		for i := 0; i < l; i++ {
			ones := 0
			for _, r := range rows {
				ones += int(r[i])
			}
			switch {
			case ones == 0:
				tpl[i] = '0'
			case ones == len(rows):
				tpl[i] = '1'
			default:
				tpl[i] = '.'
			}
		}
		grouped := ""
		for i := 0; i < l; i += 8 {
			hi := i + 8
			if hi > l {
				hi = l
			}
			grouped += string(tpl[i:hi]) + " "
		}
		t.Logf("classe %3d bits (%d corps) : %s", l, len(rows), grouped)
	}
}

// i59aBits rend n bits en clair depuis la position bit `at`, groupés par 8.
func i59aBits(pay []byte, at, n int) string {
	out := make([]byte, 0, n+n/8)
	for i := 0; i < n; i++ {
		p := at + i
		if p >= len(pay)*8 {
			break
		}
		if i > 0 && i%8 == 0 {
			out = append(out, ' ')
		}
		out = append(out, '0'+byte(pay[p>>3]>>(7-uint(p&7))&1))
	}
	return string(out)
}

// ---------------------------------------------------------------------------
// 0.3 — LES CONTRÔLES DE POSITION
// ---------------------------------------------------------------------------

func TestI59AnchorControls(t *testing.T) {
	dir := os.Getenv(i59aFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i59aFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	s := i59aSetup(t, dir)
	idx59 := i59aIndex(t, s)

	events, read, unread := i59aCollect(s, idx59)
	t.Logf("ÉVÉNEMENTS tag==3 : %d (marches vers i59 : %d abouties, %d cassées avant)",
		len(events), read, unread)
	if len(events) == 0 {
		t.Fatal("aucun événement tag==3 : rien à contrôler")
	}
	hist := map[int]int{}
	withVecs := 0
	for _, e := range events {
		hist[e.st.Inner]++
		if e.st.Vec[0].Present || e.st.Vec[1].Present || e.st.Vec[2].Present {
			withVecs++
		}
	}
	t.Logf("  tags internes : %s· avec au moins un vecteur : %d/%d",
		i59aRenderInt(hist), withVecs, len(events))

	wr := i59aWorldRange(t, s)
	scan := DefaultScanFilmOptions()
	scan.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(s.dir, scan)
	if err != nil {
		t.Fatalf("positions de production illisibles : %v", err)
	}
	tracks := i59aTracks(pos)
	box := i59aBox(pos)
	t.Logf("NUAGE BIPED (production) : %d positions · AABB X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]",
		len(pos), box[0][0], box[0][1], box[1][0], box[1][1], box[2][0], box[2][1])

	pairs := i59aPairs(t, events)
	// CANDIDAT DIRECT : la position absolue du corps (PosQ déquantifiée aux bornes de la
	// carte). C'est elle qu'on teste d'abord — les vecteurs dir/mag ne servent que si la
	// position ne porte pas l'ancre.
	i59aPosVerdict(t, s, wr, tracks, box, events, pairs)
	for k := 0; k < 3; k++ {
		slope, n := i59aFitK(t, tracks, pairs, k)
		if n >= 4 && slope > 0 {
			i59aVerdictK(t, tracks, events, pairs, k, slope, box)
		} else {
			t.Logf("Vec[%d] : ajustement impossible (%d paires exploitables) — pas de verdict", k, n)
		}
	}
	t.Log("RAPPEL du gate 0 : si (b) échoue — l'ancre ne tire pas le joueur vers elle — " +
		"LE PLAN S'ARRÊTE et le négatif va au registre")
}

// i59aDequantPos déquantifie les trois quanta de position du corps avec les bornes de la
// carte (mêmes largeurs que le découpage i0 du film — vérifié par i59aWorldRange).
func i59aDequantPos(q [3]uint32, s eaFilmSetup, wr Vec3Range) [3]float64 {
	var out [3]float64
	for ax := 0; ax < 3; ax++ {
		out[ax] = float64(DequantBipedAxis(q[ax], ax, s.lay, wr))
	}
	return out
}

// i59aPosVerdict publie les contrôles (a) et (b) pour le CANDIDAT POSITION : P =
// dequant(PosQ). Les deux classes de corps sont jugées séparément (le tir et l'accroche
// n'ont aucune raison de porter la même chose), plus la FIXITÉ entre membres d'une paire
// (une ancre est un point monde fixe ; la position du joueur, elle, bouge entre les deux
// membres — le déplacement du joueur est publié comme référence).
func i59aPosVerdict(t *testing.T, s eaFilmSetup, wr Vec3Range, tracks map[uint32][]BipedPosition,
	box [3][2]float32, events []i59aEvent, pairs []i59aPair) {
	t.Helper()
	names := map[int]string{anchorInnerLight: "LÉGER (tir)", anchorInnerHeavy: "LOURD (accroche)"}
	for _, inner := range []int{anchorInnerLight, anchorInnerHeavy} {
		var anchors []i59aAnchor
		inBox := 0
		var d0s []float64
		for _, e := range events {
			if e.st.Inner != inner || !e.st.BodyOK {
				continue
			}
			P := i59aDequantPos(e.st.PosQ, s, wr)
			p, okp := i59aNearestPos(tracks[e.slot], e.tsUS, 250_000)
			if !okp {
				continue
			}
			d0 := i59aDist(P, p)
			d0s = append(d0s, d0)
			anchors = append(anchors, i59aAnchor{slot: e.slot, tsUS: e.tsUS, a: P, m: d0})
			if i59aInBox(P, box) {
				inBox++
			}
		}
		if len(anchors) == 0 {
			t.Logf("P/%s : aucun événement exploitable", names[inner])
			continue
		}
		sort.Float64s(d0s)
		t.Logf("P/%s : %d événements · (a) EMPRISE %d/%d dans l'AABB · d(joueur, P) à t : "+
			"p10 %.2f · médiane %.2f · p90 %.2f u", names[inner], len(anchors), inBox,
			len(anchors), d0s[len(d0s)/10], d0s[len(d0s)/2], d0s[len(d0s)*9/10])
		plaus := 0
		for _, d := range d0s {
			if d >= 2 && d <= 40 {
				plaus++
			}
		}
		t.Logf("P/%s : (b) PORTÉE %d/%d dans [2, 40] u", names[inner], plaus, len(anchors))
		i59aDecreaseReport(t, "P/"+names[inner], tracks, anchors)
	}
	var fix, disp []float64
	for _, pr := range pairs {
		if !pr.s1.BodyOK || !pr.s2.BodyOK {
			continue
		}
		P1 := i59aDequantPos(pr.s1.PosQ, s, wr)
		P2 := i59aDequantPos(pr.s2.PosQ, s, wr)
		dx, dy, dz := P1[0]-P2[0], P1[1]-P2[1], P1[2]-P2[2]
		fix = append(fix, math.Sqrt(dx*dx+dy*dy+dz*dz))
		p1, ok1 := i59aNearestPos(tracks[pr.slot], pr.t1, 250_000)
		p2, ok2 := i59aNearestPos(tracks[pr.slot], pr.t2, 250_000)
		if ok1 && ok2 {
			disp = append(disp, i59aDist([3]float64{float64(p1.X), float64(p1.Y), float64(p1.Z)}, p2))
		}
	}
	if len(fix) > 0 {
		sort.Float64s(fix)
		t.Logf("P FIXITÉ inter-membres : |P1-P2| médiane %.2f u (n=%d paires)", fix[len(fix)/2], len(fix))
	}
	if len(disp) > 0 {
		sort.Float64s(disp)
		t.Logf("  (référence : déplacement du JOUEUR entre membres, médiane %.2f u, n=%d)",
			disp[len(disp)/2], len(disp))
	}
}

// i59aDecreaseReport publie la série des distances joueur->ancre à t+delta, la part
// décroissante et le témoin mélangé — le bloc (b) partagé entre candidats.
func i59aDecreaseReport(t *testing.T, label string, tracks map[uint32][]BipedPosition, anchors []i59aAnchor) {
	t.Helper()
	deltas := []uint64{0, 500_000, 1_000_000, 2_000_000}
	medians := map[uint64][]float64{}
	decr, decrN := 0, 0
	for _, a := range anchors {
		d0, dEnd, ok := i59aDistSeries(tracks[a.slot], a, deltas, medians)
		if ok {
			decrN++
			if dEnd < d0*0.8 {
				decr++
			}
		}
	}
	for _, dl := range deltas {
		ds := medians[dl]
		if len(ds) == 0 {
			continue
		}
		sort.Float64s(ds)
		t.Logf("%s : d(joueur, ancre) à t+%.1fs : médiane %.2f u (n=%d)",
			label, float64(dl)/1e6, ds[len(ds)/2], len(ds))
	}
	t.Logf("%s : DÉCROISSANCE (d(t+1s) < 0,8*d(t)) : %d/%d", label, decr, decrN)
	sd, sn := i59aShuffledDecrease(tracks, anchors)
	t.Logf("%s : TÉMOIN MÉLANGÉ : %d/%d", label, sd, sn)
}

// i59aCollect marche les records jusqu'à i59 (cible) et capture les lectures tag==3.
func i59aCollect(s eaFilmSetup, idx59 int) (events []i59aEvent, read, unread int) {
	var capt struct {
		st  AbilityNonPredictedState
		got bool
	}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { capt.st, capt.got = st, true })
	defer SetAbilityNonPredictedHook(prev)
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for pos := 0; pos+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, pos, total, s.slots, true, s.lay)
				if !ok {
					pos++
					continue
				}
				if eaMaskHas(idx, idx59) {
					capt.got = false
					if eaWalkThrough(pay, i0, total, idx, s, idx59) && capt.got {
						read++
						if capt.st.Tag == 3 {
							events = append(events, i59aEvent{slot: slot, tsUS: pk.TimestampUS, st: capt.st})
						}
					} else {
						unread++
					}
				}
				pos = i0 + s.lay.TotalBits()
			}
		}
	}
	return events, read, unread
}

// i59aWorldRange rend les bornes monde de la carte du film. La carte est auto-détectée :
// l'entrée du catalogue dont AxisWidths == le découpage i0 LU DANS LE FILM (le contrôle de
// cohérence du catalogue, map_bounds.go). I59A_MAP force le choix si plusieurs cartes
// partagent la signature.
func i59aWorldRange(t *testing.T, s eaFilmSetup) Vec3Range {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	if name := os.Getenv(i59aMapEnv); name != "" {
		e, err := cat.Lookup(name)
		if err != nil {
			t.Fatalf("carte %q absente du catalogue : %v", name, err)
		}
		if e.AxisWidths != s.lay.AxisW {
			t.Fatalf("carte %q : largeurs d'axe %v != découpage du film %v — mauvaise carte",
				name, e.AxisWidths, s.lay.AxisW)
		}
		t.Logf("carte forcée : %s (bornes %v..%v)", name, e.Min, e.Max)
		return e.Range()
	}
	var hits []string
	var found MapQuantEntry
	for name, e := range cat.Maps {
		if e.AxisWidths == s.lay.AxisW {
			hits = append(hits, name)
			found = e
		}
	}
	sort.Strings(hits)
	if len(hits) != 1 {
		t.Fatalf("signature de largeurs %v : %d cartes candidates %v — renseigner %s",
			s.lay.AxisW, len(hits), hits, i59aMapEnv)
	}
	t.Logf("carte auto-détectée par signature %v : %s (bornes %v..%v)",
		s.lay.AxisW, hits[0], found.Min, found.Max)
	return found.Range()
}

// i59aTracks indexe les positions de production par slot, triées par temps.
func i59aTracks(pos []BipedPosition) map[uint32][]BipedPosition {
	out := map[uint32][]BipedPosition{}
	for _, p := range pos {
		out[p.Slot] = append(out[p.Slot], p)
	}
	for _, list := range out {
		sort.Slice(list, func(a, b int) bool { return list[a].TimestampUS < list[b].TimestampUS })
	}
	return out
}

// i59aBox rend l'AABB du nuage biped — le référentiel du contrôle (a), indépendant du
// décodage de l'ancre (non circulaire).
func i59aBox(pos []BipedPosition) (box [3][2]float32) {
	for i, p := range pos {
		v := [3]float32{p.X, p.Y, p.Z}
		for ax := 0; ax < 3; ax++ {
			if i == 0 || v[ax] < box[ax][0] {
				box[ax][0] = v[ax]
			}
			if i == 0 || v[ax] > box[ax][1] {
				box[ax][1] = v[ax]
			}
		}
	}
	return box
}

// i59aNearestPos rend l'échantillon de trajectoire du slot le plus proche de tsUS, à
// tolUS près.
func i59aNearestPos(tr []BipedPosition, tsUS, tolUS uint64) (BipedPosition, bool) {
	if len(tr) == 0 {
		return BipedPosition{}, false
	}
	i := sort.Search(len(tr), func(k int) bool { return tr[k].TimestampUS >= tsUS })
	best, got := BipedPosition{}, false
	var bestD int64
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(tr) {
			continue
		}
		d := int64(tr[k].TimestampUS) - int64(tsUS)
		if d < 0 {
			d = -d
		}
		if !got || d < bestD {
			best, bestD, got = tr[k], d, true
		}
	}
	if !got || bestD > int64(tolUS) {
		return BipedPosition{}, false
	}
	return best, true
}

// i59aPair est une paire de lectures tag==3 d'une même vie à moins d'i59aPairGapUS.
type i59aPair struct {
	slot   uint32
	t1, t2 uint64
	s1, s2 AbilityNonPredictedState
}

// i59aPairs groupe les événements en paires et publie la mesure (c) : distribution des
// écarts au plus proche voisin du même slot, et différence des quanta entre membres.
func i59aPairs(t *testing.T, events []i59aEvent) []i59aPair {
	t.Helper()
	bySlot := map[uint32][]i59aEvent{}
	for _, e := range events {
		bySlot[e.slot] = append(bySlot[e.slot], e)
	}
	var gaps []float64
	var pairs []i59aPair
	sameVec, diffVec := 0, 0
	for slot, list := range bySlot {
		sort.Slice(list, func(a, b int) bool { return list[a].tsUS < list[b].tsUS })
		for i := 1; i < len(list); i++ {
			gap := list[i].tsUS - list[i-1].tsUS
			gaps = append(gaps, float64(gap)/1e6)
			if gap > i59aPairGapUS {
				continue
			}
			pairs = append(pairs, i59aPair{slot: slot, t1: list[i-1].tsUS, t2: list[i].tsUS,
				s1: list[i-1].st, s2: list[i].st})
			if list[i-1].st.Vec == list[i].st.Vec {
				sameVec++
			} else {
				diffVec++
			}
		}
	}
	sort.Float64s(gaps)
	if len(gaps) > 0 {
		t.Logf("(c) ÉCARTS successifs même slot : n=%d · p10 %.3fs · médiane %.3fs · p90 %.3fs",
			len(gaps), gaps[len(gaps)/10], gaps[len(gaps)/2], gaps[len(gaps)*9/10])
	}
	t.Logf("(c) PAIRES (écart <= %.2fs) : %d — quanta des 3 vecteurs IDENTIQUES entre "+
		"membres : %d · DIFFÉRENTS : %d", float64(i59aPairGapUS)/1e6, len(pairs), sameVec, diffVec)
	return pairs
}

// i59aFitK mesure la loi de magnitude du vecteur k par la FIXITÉ DE L'ANCRE : pour une
// paire (t1, t2), un même point monde A vérifie p1 + m1*d1 = p2 + m2*d2 ; (m1, m2) se
// résout aux moindres carrés (directions unitaires exactes, plage non requise), puis la
// loi log/exp de FUN_14076d6dc s'ajuste sur les couples (quantum, magnitude) :
// log1p(m) = pente * (q + 0,5) / 2^12. Rend la pente et le nombre de paires exploitables.
func i59aFitK(t *testing.T, tracks map[uint32][]BipedPosition, pairs []i59aPair, k int) (float64, int) {
	t.Helper()
	type qm struct{ q, m float64 }
	var samples []qm
	neg, ill, res := 0, 0, []float64{}
	for _, pr := range pairs {
		v1, v2 := pr.s1.Vec[k], pr.s2.Vec[k]
		if !v1.Present || !v2.Present {
			continue
		}
		d1, ok1 := DecodeAimVectorChecked(v1.DirQ, anchorVecDirBits)
		d2, ok2 := DecodeAimVectorChecked(v2.DirQ, anchorVecDirBits)
		p1, okp1 := i59aNearestPos(tracks[pr.slot], pr.t1, 250_000)
		p2, okp2 := i59aNearestPos(tracks[pr.slot], pr.t2, 250_000)
		if !ok1 || !ok2 || !okp1 || !okp2 {
			continue
		}
		e := [3]float64{float64(p2.X - p1.X), float64(p2.Y - p1.Y), float64(p2.Z - p1.Z)}
		c := i59aDot(d1, d2)
		if math.Abs(c) > 0.995 {
			ill++
			continue
		}
		b1, b2 := i59aDotF(d1, e), i59aDotF(d2, e)
		m1 := (b1 - c*b2) / (1 - c*c)
		m2 := (c*b1 - b2) / (1 - c*c)
		if m1 <= 0 || m2 <= 0 {
			neg++
			continue
		}
		var r float64
		for ax := 0; ax < 3; ax++ {
			d := m1*float64(d1[ax]) - m2*float64(d2[ax]) - e[ax]
			r += d * d
		}
		res = append(res, math.Sqrt(r))
		samples = append(samples, qm{float64(v1.MagQ), m1}, qm{float64(v2.MagQ), m2})
	}
	if len(samples) == 0 {
		t.Logf("Vec[%d] : 0 paire exploitable (mal conditionnées %d · magnitudes négatives %d)", k, ill, neg)
		return 0, 0
	}
	// Moindres carrés sans ordonnée à l'origine : log1p(m) = pente * (q+0,5)/4096.
	var sxx, sxy float64
	for _, s := range samples {
		x := (s.q + 0.5) / 4096.0
		y := math.Log1p(s.m)
		sxx += x * x
		sxy += x * y
	}
	slope := sxy / sxx
	sort.Float64s(res)
	t.Logf("Vec[%d] AJUSTEMENT : %d paires (mal conditionnées %d · m<=0 %d) · résidu médian %.2f u · "+
		"pente %.3f => max implicite %.1f u", k, len(res), ill, neg, res[len(res)/2],
		slope, math.Expm1(slope))
	return slope, len(res)
}

func i59aDot(a, b [3]float32) float64 {
	return float64(a[0])*float64(b[0]) + float64(a[1])*float64(b[1]) + float64(a[2])*float64(b[2])
}

func i59aDotF(a [3]float32, b [3]float64) float64 {
	return float64(a[0])*b[0] + float64(a[1])*b[1] + float64(a[2])*b[2]
}

// i59aMag applique la loi ajustée : m = expm1(pente * (q+0,5)/4096).
func i59aMag(q uint32, slope float64) float64 {
	return math.Expm1(slope * (float64(q) + 0.5) / 4096.0)
}

// i59aAnchor est une ancre candidate : le point monde A = p(t) + m*d, daté et attribué à
// sa vie.
type i59aAnchor struct {
	slot uint32
	tsUS uint64
	a    [3]float64
	m    float64
}

// i59aVerdictK publie les contrôles (a) et (b) pour le vecteur k sous la loi ajustée,
// avec un TÉMOIN MÉLANGÉ pour (b) : l'ancre de l'événement i contre la trajectoire de
// l'événement i+7 — si « les joueurs convergent en général », le témoin le dira.
func i59aVerdictK(t *testing.T, tracks map[uint32][]BipedPosition, events []i59aEvent,
	pairs []i59aPair, k int, slope float64, box [3][2]float32) {
	t.Helper()
	var anchors []i59aAnchor
	inBox := 0
	for _, e := range events {
		v := e.st.Vec[k]
		if !v.Present {
			continue
		}
		d, ok := DecodeAimVectorChecked(v.DirQ, anchorVecDirBits)
		p, okp := i59aNearestPos(tracks[e.slot], e.tsUS, 250_000)
		if !ok || !okp {
			continue
		}
		m := i59aMag(v.MagQ, slope)
		a := [3]float64{float64(p.X) + m*float64(d[0]), float64(p.Y) + m*float64(d[1]),
			float64(p.Z) + m*float64(d[2])}
		anchors = append(anchors, i59aAnchor{slot: e.slot, tsUS: e.tsUS, a: a, m: m})
		if i59aInBox(a, box) {
			inBox++
		}
	}
	if len(anchors) == 0 {
		t.Logf("Vec[%d] : aucune ancre candidate décodable — pas de verdict", k)
		return
	}
	t.Logf("Vec[%d] (a) EMPRISE : %d/%d ancres dans l'AABB du nuage biped (marge 5 %%)",
		k, inBox, len(anchors))
	plaus := 0
	for _, a := range anchors {
		if a.m >= 2 && a.m <= 40 {
			plaus++
		}
	}
	t.Logf("Vec[%d] (b) PORTÉE : %d/%d à magnitude plausible d'un tir de grappin [2, 40] u",
		k, plaus, len(anchors))
	i59aDecreaseReport(t, fmt.Sprintf("Vec[%d]", k), tracks, anchors)
	fix := i59aPairFixity(tracks, pairs, k, slope)
	if len(fix) > 0 {
		sort.Float64s(fix)
		t.Logf("Vec[%d] FIXITÉ : |A(membre 1) - A(membre 2)| médiane %.2f u (n=%d paires)",
			k, fix[len(fix)/2], len(fix))
	}
}

// i59aDistSeries alimente la série des distances joueur->ancre à t+delta ; rend d(t) et
// d(t+1s) quand les deux existent.
func i59aDistSeries(tr []BipedPosition, a i59aAnchor, deltas []uint64,
	medians map[uint64][]float64) (d0, d1s float64, ok bool) {
	got0, got1 := false, false
	for _, dl := range deltas {
		p, okp := i59aNearestPos(tr, a.tsUS+dl, 300_000)
		if !okp {
			continue
		}
		d := i59aDist(a.a, p)
		medians[dl] = append(medians[dl], d)
		if dl == 0 {
			d0, got0 = d, true
		}
		if dl == 1_000_000 {
			d1s, got1 = d, true
		}
	}
	return d0, d1s, got0 && got1
}

func i59aDist(a [3]float64, p BipedPosition) float64 {
	dx, dy, dz := a[0]-float64(p.X), a[1]-float64(p.Y), a[2]-float64(p.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func i59aInBox(a [3]float64, box [3][2]float32) bool {
	for ax := 0; ax < 3; ax++ {
		lo, hi := float64(box[ax][0]), float64(box[ax][1])
		margin := (hi - lo) * 0.05
		if a[ax] < lo-margin || a[ax] > hi+margin {
			return false
		}
	}
	return true
}

// i59aShuffledDecrease rejoue la décroissance (b) en croisant l'ancre de l'événement i
// avec la trajectoire de l'événement i+7 (autre vie, autre instant).
func i59aShuffledDecrease(tracks map[uint32][]BipedPosition, anchors []i59aAnchor) (decr, n int) {
	if len(anchors) < 2 {
		return 0, 0
	}
	for i, a := range anchors {
		o := anchors[(i+7)%len(anchors)]
		tr := tracks[o.slot]
		p0, ok0 := i59aNearestPos(tr, o.tsUS, 300_000)
		p1, ok1 := i59aNearestPos(tr, o.tsUS+1_000_000, 300_000)
		if !ok0 || !ok1 {
			continue
		}
		n++
		if i59aDist(a.a, p1) < i59aDist(a.a, p0)*0.8 {
			decr++
		}
	}
	return decr, n
}

// i59aPairFixity mesure |A1 - A2| par paire : une vraie ancre est un point monde FIXE.
func i59aPairFixity(tracks map[uint32][]BipedPosition, pairs []i59aPair, k int, slope float64) []float64 {
	var out []float64
	for _, pr := range pairs {
		v1, v2 := pr.s1.Vec[k], pr.s2.Vec[k]
		if !v1.Present || !v2.Present {
			continue
		}
		d1, ok1 := DecodeAimVectorChecked(v1.DirQ, anchorVecDirBits)
		d2, ok2 := DecodeAimVectorChecked(v2.DirQ, anchorVecDirBits)
		p1, okp1 := i59aNearestPos(tracks[pr.slot], pr.t1, 250_000)
		p2, okp2 := i59aNearestPos(tracks[pr.slot], pr.t2, 250_000)
		if !ok1 || !ok2 || !okp1 || !okp2 {
			continue
		}
		m1, m2 := i59aMag(v1.MagQ, slope), i59aMag(v2.MagQ, slope)
		a1 := [3]float64{float64(p1.X) + m1*float64(d1[0]), float64(p1.Y) + m1*float64(d1[1]),
			float64(p1.Z) + m1*float64(d1[2])}
		a2 := [3]float64{float64(p2.X) + m2*float64(d2[0]), float64(p2.Y) + m2*float64(d2[1]),
			float64(p2.Z) + m2*float64(d2[2])}
		dx, dy, dz := a1[0]-a2[0], a1[1]-a2[1], a1[2]-a2[2]
		out = append(out, math.Sqrt(dx*dx+dy*dy+dz*dz))
	}
	return out
}
