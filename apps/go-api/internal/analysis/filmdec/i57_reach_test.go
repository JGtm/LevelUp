package filmdec

// i57_reach_test.go — INSTRUMENT DE MESURE de l'ÉTAPE 3 du plan
// .ai/V7.5/replay2d/PLAN_EQUIPEMENT_ACTIF.md : « pourquoi le saut vers i56/i57 n'aboutit
// pas ». Il INSTRUMENTE avant de conclure, comme l'item 3.1 l'exige.
//
// CE QU'IL COMPTE, ET POURQUOI CES COMPTEURS-LÀ. Le plan supposait un échec de TRAVERSÉE :
// un composant non porté avant i56 ferait perdre l'alignement, et `repairUnportedComponent`
// n'arriverait pas à le sauter. Cet instrument sépare les trois causes possibles, qui ne se
// corrigent pas du tout de la même façon :
//
//	(A) le record N'EST PAS RECONNU — le détecteur offline n'accepte que le masque CREUX
//	    (porte 0 : R(3) count + count×R(6)). Le masque DENSE (porte 1 : R(64), cf.
//	    consumeMask / FUN_1406d7610) est rejeté par `matchBipedHeaderRaw`, qui exige gate==0.
//	    Ces records ne sont donc ni lus ni comptés — ils sont INVISIBLES, pas ratés ;
//	(B) le record est reconnu mais la MARCHE échoue — un composant rend ported=false. Le nom
//	    du composant fautif est alors publié, avec son décompte ;
//	(C) le record est reconnu et marché, mais le masque n'annonce simplement PAS i56/i57 —
//	    c'est une question de fréquence de transmission, pas de décodage.
//
// Sans cette séparation, « i56 lu sur 0,10 % des records » se lit à tort comme un échec de
// décodage alors que c'est peut-être un dénominateur qui ne contient pas la population.
//
// LECTURE SEULE, gardé par I57_FILM, sauté partout ailleurs (CI comprise).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I57_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI57Reach$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i57FilmEnv = "I57_FILM"

// i57Index est l'index d'itérateur du composant biped-spartan-ability-component.
const i57Index = 57

// i57DenseMaskBits est la largeur du masque dense (porte à 1), cf. consumeMask.
const i57DenseMaskBits = 64

// i57GateOffset est la position, depuis le début du record, du bit de PORTE du masque.
// L'en-tête vaut préfixe(1) + idLow(14) + tag(2) + porte(1) + [creux: count(3) | dense:
// R(64)], donc la porte est le 18e bit (index 17). `bipedHeaderBits` = 21 vaut pour le seul
// encodage creux : le dense n'a pas de champ count.
const i57GateOffset = 17

// i57Counters porte les dénominateurs de la mesure. Chaque champ répond à une des trois
// causes séparées en tête de fichier.
type i57Counters struct {
	sparse, dense           int // records reconnus, par encodage de masque
	sparseWith, denseWith   int // ... dont le masque annonce i56 ou i57
	sparseRead, denseRead   int // ... et dont la marche atteint le composant
	sparseBroke, denseBroke int // ... et dont la marche casse avant
	i57Vals                 map[uint32]int
	broken                  map[string]int
	samples                 []i57Sample
	i54On                   map[uint32][]uint64
}

// i57Sample est une lecture des 2 bits d'i57, datée et attribuée à un slot.
type i57Sample struct {
	slot uint32
	tsUS uint64
	val  uint32
}

func TestI57Reach(t *testing.T) {
	dir := os.Getenv(i57FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i57FilmEnv)
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	if len(slots) == 0 {
		t.Fatalf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	arch := i48Archetype(t, dir)

	c := i57Counters{
		i57Vals: map[uint32]int{}, broken: map[string]int{}, i54On: map[uint32][]uint64{},
	}
	for _, ch := range chunks {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			i57ScanPacket(pk.Payload(data), pk.TimestampUS, slots, lay, arch, &c)
		}
	}
	i57Report(t, &c)
	i57Correlate(t, i56Episodes(c.i54On), c.samples)
}

// i57ScanPacket balaie un payload de paquet delta et compte les deux populations.
func i57ScanPacket(
	pay []byte, tsUS uint64, slots map[uint32]bool, lay I0Layout, arch Archetype, c *i57Counters,
) {
	total := len(pay) * 8
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for p := 0; p+minRecord <= total; {
		if i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay); ok {
			c.sparse++
			i57Account(pay, i0, total, idx, lay, arch, c, false, slot, tsUS)
			p = i0 + lay.TotalBits()
			continue
		}
		if i0, slot, idx, ok := i57MatchDense(pay, p, total, slots, lay, arch); ok {
			c.dense++
			i57Account(pay, i0, total, idx, lay, arch, c, true, slot, tsUS)
			p = i0 + lay.TotalBits()
			continue
		}
		p++
	}
}

// i57MatchDense teste la grammaire d'en-tête avec un masque DENSE (porte à 1, R(64)). La
// validation est volontairement plus stricte que celle du masque creux, parce qu'un masque
// dense n'a pas de champ `count` pour borner le hasard : on exige que le bit 0 du masque
// soit armé (i0 est toujours présent — même invariant que « premier index = 0 » côté creux),
// que l'en-tête d'i0 soit nul (i0 absolu), et que la MARCHE COMPLÈTE du masque soit portée.
// Sans cette dernière condition, n'importe quel motif de 64 bits ferait un faux positif.
func i57MatchDense(
	pay []byte, p, total int, slots map[uint32]bool, lay I0Layout, arch Archetype,
) (int, uint32, []int, bool) {
	slot := readBitsAt(pay, p+1, bipedSlotBits)
	if readBitsAt(pay, p, 1) != 1 || !slots[slot] {
		return 0, 0, nil, false
	}
	if readBitsAt(pay, p+14, 2) != 1 || readBitsAt(pay, p+16, 1) != 0 {
		return 0, 0, nil, false
	}
	if readBitsAt(pay, p+i57GateOffset, 1) != 1 { // porte à 1 = masque DENSE
		return 0, 0, nil, false
	}
	maskAt := p + i57GateOffset + 1
	if maskAt+i57DenseMaskBits+lay.TotalBits() > total {
		return 0, 0, nil, false
	}
	var idx []int
	for b := 0; b < i57DenseMaskBits; b++ {
		if readBitsAt(pay, maskAt+b, 1) == 1 {
			idx = append(idx, b)
		}
	}
	if len(idx) == 0 || idx[0] != 0 {
		return 0, 0, nil, false
	}
	i0 := maskAt + i57DenseMaskBits
	if readBitsAt(pay, i0, lay.GateBits) != 0 { // i0 absolu, comme le chemin creux
		return 0, 0, nil, false
	}
	if _, _, _, ok := i57Walk(pay, i0, total, idx, lay, arch, -1); !ok {
		return 0, 0, nil, false
	}
	return i0, slot, idx, true
}

// i57Account marche le record et impute le résultat aux bons compteurs.
func i57Account(
	pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype,
	c *i57Counters, dense bool, slot uint32, tsUS uint64,
) {
	// La marche se déclenche AUSSI sur i54 seul. Ne la déclencher que sur i56/i57 rendrait
	// la corrélation CIRCULAIRE : les seuls épisodes vus seraient ceux transmis dans le
	// MÊME record qu'i57, et ils coïncideraient par construction.
	has5657 := i56InMask(idx) || i57InMask(idx)
	if !has5657 && !i54InMask(idx) {
		return
	}
	if has5657 {
		if dense {
			c.denseWith++
		} else {
			c.sparseWith++
		}
	}
	val, flag1, broke, ok := i57Walk(pay, i0, total, idx, lay, arch, i57Index)
	if flag1 == 1 {
		c.i54On[slot] = append(c.i54On[slot], tsUS)
	}
	if !has5657 {
		return
	}
	switch {
	case ok:
		if dense {
			c.denseRead++
		} else {
			c.sparseRead++
		}
		if val >= 0 {
			c.i57Vals[uint32(val)]++
			c.samples = append(c.samples, i57Sample{slot: slot, tsUS: tsUS, val: uint32(val)})
		}
	default:
		if dense {
			c.denseBroke++
		} else {
			c.sparseBroke++
		}
		if broke != "" {
			c.broken[broke]++
		}
	}
}

// i57Walk marche les composants du masque avec les désers de PRODUCTION. `stopAt` (-1 pour
// aucun) fait lire le composant demandé en clair — i57 est un R(2) en tête de composant.
// Rend le nom du composant qui casse la marche : c'est LE renseignement que l'item 3.1
// demande, et il n'existait nulle part.
func i57Walk(
	pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype, stopAt int,
) (val, flag1 int, broke string, ok bool) {
	val, flag1 = -1, -1
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		if at > total {
			return val, flag1, "", false
		}
		if id == i54Index && at+1 <= total {
			flag1 = int(readBitsAt(pay, at, 1))
		}
		if id == stopAt {
			if at+2 > total {
				return val, flag1, "", false
			}
			val = int(readBitsAt(pay, at, 2))
			return val, flag1, "", true
		}
		name := arch.component(id)
		if name == "" {
			return val, flag1, fmt.Sprintf("i%d(sans nom au registre)", id), false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return val, flag1, fmt.Sprintf("i%d %s", id, name), false
		}
		at = br.BitPos()
	}
	return val, flag1, "", stopAt < 0
}

// i57Correlate confronte les épisodes `i54` aux valeurs d'`i57`, valeur par valeur, contre
// les MÊMES témoins décalés que la mesure d'énergie du 2026-08-13 (±5 s). Le protocole est
// repris à l'identique pour que les deux mesures se comparent.
//
// CE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT : si aucune valeur d'i57 n'a de taux réel qui écrase ses
// deux témoins, alors i57 ne date pas l'usage d'un équipement, et rien ne s'affiche.
func i57Correlate(t *testing.T, eps []i56Episode, samples []i57Sample) {
	if len(eps) == 0 || len(samples) == 0 {
		t.Logf("VERDICT : %d épisodes i54, %d lectures i57 — corrélation non calculable",
			len(eps), len(samples))
		return
	}
	vals := map[uint32]bool{}
	for _, s := range samples {
		vals[s.val] = true
	}
	keys := make([]uint32, 0, len(vals))
	for v := range vals {
		keys = append(keys, v)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	t.Logf("ÉPISODES i54 %d · LECTURES i57 %d", len(eps), len(samples))
	for _, v := range keys {
		real := i57Hit(eps, samples, v, 0)
		plus := i57Hit(eps, samples, v, i56ControlShiftUS)
		minus := i57Hit(eps, samples, v, -i56ControlShiftUS)
		t.Logf("  i57==%d : réel %d/%d (%.1f %%) · témoin +5 s %d (%.1f %%) · témoin -5 s %d (%.1f %%)",
			v, real, len(eps), 100*float64(real)/float64(len(eps)),
			plus, 100*float64(plus)/float64(len(eps)),
			minus, 100*float64(minus)/float64(len(eps)))
	}
	// LA MESURE QUI COMPTE VRAIMENT : une VALEUR présente près d'un épisode peut n'être que
	// de la co-transmission (le film retransmet plusieurs composants au même instant). Un
	// CHANGEMENT de valeur, lui, est un événement d'état.
	tr := i57Transitions(samples)
	t.Logf("TRANSITIONS de valeur i57 (changement sur un même slot) : %d", len(tr))
	if len(tr) > 0 {
		real := i57Hit(eps, tr, i57AnyVal, 0)
		plus := i57Hit(eps, tr, i57AnyVal, i56ControlShiftUS)
		minus := i57Hit(eps, tr, i57AnyVal, -i56ControlShiftUS)
		t.Logf("  TRANSITION : réel %d/%d (%.1f %%) · témoin +5 s %d (%.1f %%) · témoin -5 s %d (%.1f %%)",
			real, len(eps), 100*float64(real)/float64(len(eps)),
			plus, 100*float64(plus)/float64(len(eps)),
			minus, 100*float64(minus)/float64(len(eps)))
	}
	t.Log("RAPPEL du critère : si aucun taux réel n'écrase ses deux témoins décalés, " +
		"i57 ne date pas un usage — verdict NON, aucun effet affiché")
}

// i57AnyVal fait ignorer la valeur à `i57Hit` : la coïncidence porte alors sur la seule
// présence de l'échantillon (utilisé pour les transitions).
const i57AnyVal = ^uint32(0)

// i57Transitions rend les échantillons où la valeur d'i57 CHANGE par rapport à la lecture
// précédente du même slot. Une valeur retransmise à l'identique n'est pas un événement.
func i57Transitions(samples []i57Sample) []i57Sample {
	bySlot := map[uint32][]i57Sample{}
	for _, s := range samples {
		bySlot[s.slot] = append(bySlot[s.slot], s)
	}
	var out []i57Sample
	for _, list := range bySlot {
		sort.Slice(list, func(a, b int) bool { return list[a].tsUS < list[b].tsUS })
		for i := 1; i < len(list); i++ {
			if list[i].val != list[i-1].val {
				out = append(out, list[i])
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].tsUS < out[b].tsUS })
	return out
}

// i57Hit compte les épisodes ayant une lecture d'i57 de valeur `val`, MÊME SLOT, dans la
// fenêtre de coïncidence autour de l'épisode décalé de `shift`.
func i57Hit(eps []i56Episode, samples []i57Sample, val uint32, shift int64) int {
	n := 0
	for _, e := range eps {
		ts := int64(e.tsUS) + shift
		for _, s := range samples {
			if s.slot != e.slot || (val != i57AnyVal && s.val != val) {
				continue
			}
			diff := int64(s.tsUS) - ts
			if diff < 0 {
				diff = -diff
			}
			if diff <= i56CoincidenceWindowUS {
				n++
				break
			}
		}
	}
	return n
}

// i57InMask dit si le masque du record annonce le composant i57.
func i57InMask(idx []int) bool {
	for _, id := range idx {
		if id == i57Index {
			return true
		}
	}
	return false
}

// i57Report publie les trois causes séparément. Un taux sans son dénominateur ne se juge pas.
func i57Report(t *testing.T, c *i57Counters) {
	tot := c.sparse + c.dense
	if tot == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS reconnus %d — masque CREUX %d (%.1f %%) · masque DENSE %d (%.1f %%)",
		tot, c.sparse, 100*float64(c.sparse)/float64(tot),
		c.dense, 100*float64(c.dense)/float64(tot))
	t.Logf("CREUX : masque ∋ i56/i57 %d · marche ABOUTIE %d · marche CASSÉE %d",
		c.sparseWith, c.sparseRead, c.sparseBroke)
	t.Logf("DENSE : masque ∋ i56/i57 %d · marche ABOUTIE %d · marche CASSÉE %d",
		c.denseWith, c.denseRead, c.denseBroke)
	if len(c.broken) > 0 {
		type kv struct {
			n string
			c int
		}
		var rows []kv
		for k, v := range c.broken {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a].c > rows[b].c })
		t.Log("COMPOSANTS QUI CASSENT LA MARCHE (le renseignement que l'étape 3.1 demande) :")
		for _, r := range rows {
			t.Logf("  %-52s %d", r.n, r.c)
		}
	} else {
		t.Log("AUCUN composant ne casse la marche : la traversée n'est PAS le facteur limitant")
	}
	if len(c.i57Vals) > 0 {
		t.Logf("i57 valeurs R(2) lues : %s", i48RenderU32(c.i57Vals))
	}
}
